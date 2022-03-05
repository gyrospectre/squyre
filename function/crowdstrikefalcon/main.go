package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre/pkg/squyre"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/hosts"
	"github.com/crowdstrike/gofalcon/falcon/client/intel"
	"github.com/crowdstrike/gofalcon/falcon/models"
)

const (
	provider       = "CrowdStrike Falcon"
	baseURL        = "https://api.crowdstrike.com"
	supports       = "ipv4,domain,sha256,hostname"
	secretLocation = "CrowdstrikeAPI"
)

var (
	// InitClient abstracts this function to allow for tests
	InitClient        = InitFalconClient
	OnlyLogMatches, _ = strconv.ParseBool(os.Getenv("ONLY_LOG_MATCHES"))
	getIndicator      = getFalconIndicator
)

var templateIntelIndicator = `
Found Falcon X indicator for %s:

Malicious confidence: '%s'.
Added: %s
Updated: %s

Labels: %s
Kill Chains: %s
Malware Families: %s
Vulnerabilities: %s
Threat Types: %s
Targets: %s

More information at: https://falcon.crowdstrike.com/search/?term=_all:~'%s'

`

var templateHost = `
Found host %s in Falcon:

Last seen: %s

Type: %s %s
Serial: %s
OS: %s
External IP: %s

Policies:
- %s

More information at: https://falcon.crowdstrike.com/hosts/hosts?filter=_all:~'%s'

`

type apiKeySecret struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	FalconCloud  string `json:"falconCloud"`
}

// InitFalconClient initialises a Falcon client using credentials from AWS Secrets Manager
func InitFalconClient() (*client.CrowdStrikeAPISpecification, error) {
	// Fetch API key from Secrets Manager
	smresponse, err := squyre.GetSecret(secretLocation)
	if err != nil {
		log.Errorf("Failed to fetch Crowdstrike Falcon secret: %s", err)
	}

	var secret apiKeySecret
	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	// Connect to Crowdstrike Falcon
	client, err := falcon.NewClient(&falcon.ApiConfig{
		ClientId:     secret.ClientID,
		ClientSecret: secret.ClientSecret,
		MemberCID:    "",
		Cloud:        falcon.Cloud(secret.FalconCloud),
		Context:      context.Background(),
		Debug:        false,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func handleRequest(ctx context.Context, alert squyre.Alert) (string, error) {
	log.Infof("Starting %s run for alert %s", provider, alert.ID)
	log.Infof("OnlyLogMatches is set to %t", OnlyLogMatches)

	if len(alert.Subjects) == 0 {
		log.Info("Alert has no subjects to process.")
		finalJSON, _ := json.Marshal(alert)
		return string(finalJSON), nil
	}

	falconClient, err := InitClient()
	if err != nil {
		log.Error("Failed to initialise client")
		return "Failed to initialise client", err
	}

	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {
		if !strings.Contains(supports, subject.Type) {
			log.Info("Subject not supported by this provider. Skipping.")
		} else {
			// Subject supported. Build a result object to hold our goodies
			var result = squyre.Result{
				Source:         provider,
				AttributeValue: subject.Value,
				Success:        false,
			}
			if subject.Type == "hostname" {
				hostDetail, err := getHost(falconClient, subject.Value)
				if err != nil {
					return "Error fetching data from API!", err
				}
				result.Success = true

				if hostDetail != nil {
					log.Infof("Received %s response for %s", provider, subject.Value)
					result.Message = messageFromHostDetail(hostDetail)

				} else {
					log.Infof("Host %s not found in %s", subject.Value, provider)
					result.Message = fmt.Sprintf("Host '%s' not found in Falcon X.", subject.Value)
				}

				// Add the enriched details back to the results
				alert.Results = append(alert.Results, result)
				log.Infof("Added %s to result set", subject.Value)
			} else {
				var indicator *models.DomainPublicIndicatorV3
				indicator, err = getIndicator(falconClient, subject.Value)
				if err != nil {
					log.Error("Error fetching data from API!")
					return "Error fetching data from API!", err
				}
				result.Success = true
				if indicator != nil {
					log.Infof("Received %s response for %s", provider, subject.Value)
					result.Success = true
				} else {
					result.Success = false
				}

				if !result.Success && OnlyLogMatches {
					log.Infof("Skipping non match for %s", subject.Value)
				} else {
					if !result.Success {
						result.Message = "Indicator not found in Falcon X."
					} else {
						result.Message = messageFromIndicator(indicator)
					}
					alert.Results = append(alert.Results, result)
					log.Infof("Added %s to result set", subject.Value)
				}
			}
		}
	}
	log.Infof("Successfully ran %s. Yielded %d results for %d subjects.", provider, len(alert.Results), len(alert.Subjects))

	// Convert the alert object into Json for the step function
	finalJSON, _ := json.Marshal(alert)
	return string(finalJSON), nil
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}

func messageFromIndicator(indicator *models.DomainPublicIndicatorV3) string {
	var labels []string
	for _, label := range indicator.Labels {
		labels = append(labels, *label.Name)
	}

	message := fmt.Sprintf(templateIntelIndicator,
		*indicator.Indicator,
		*indicator.MaliciousConfidence,
		time.Unix(*indicator.PublishedDate, 0),
		time.Unix(*indicator.LastUpdated, 0),
		strings.Join(labels, ","),
		strings.Join(indicator.KillChains, ","),
		strings.Join(indicator.MalwareFamilies, ","),
		strings.Join(indicator.Vulnerabilities, ","),
		strings.Join(indicator.ThreatTypes, ","),
		strings.Join(indicator.Targets, ","),
		*indicator.Indicator,
	)

	return string(message)
}

func messageFromHostDetail(host *models.DomainDeviceSwagger) string {
	var policies []string
	var state string
	for _, policy := range host.Policies {
		if policy.Applied {
			state = fmt.Sprintf("%s (%s) applied at %s", *policy.PolicyType, *policy.PolicyID, policy.AppliedDate)
		} else {
			state = fmt.Sprintf("%s (%s) not applied!", *policy.PolicyType, *policy.PolicyID)
		}
		policies = append(policies, state)
	}

	message := fmt.Sprintf(templateHost,
		host.Hostname,
		host.LastSeen,
		host.SystemManufacturer,
		host.SystemProductName,
		host.SerialNumber,
		host.OsVersion,
		host.ExternalIP,
		strings.Join(policies, "\n- "),
		host.Hostname,
	)

	return string(message)
}

func getFalconIndicator(client *client.CrowdStrikeAPISpecification, name string) (*models.DomainPublicIndicatorV3, error) {
	filter := fmt.Sprintf("indicator:'%s'", name)

	indicatorsChannel, errorChannel := queryIntelIndicators(client, &filter, nil)
	for openChannels := 2; openChannels > 0; {
		select {
		case err, ok := <-errorChannel:
			if ok {
				log.Errorf("Failed to fetch data from %s", provider)
				return nil, err
			}
			openChannels--
		case indicator, ok := <-indicatorsChannel:
			if ok {
				return indicator, nil
			}
			openChannels--
		}
	}
	return nil, nil
}

func getHost(client *client.CrowdStrikeAPISpecification, name string) (*models.DomainDeviceSwagger, error) {
	filter := fmt.Sprintf("hostname:'%s'", name)

	var hostDetailBatch []*models.DomainDeviceSwagger
	hostIDs, err := getHostIds(client, &filter)
	if err != nil {
		log.Error(falcon.ErrorExplain(err))
		return nil, err
	}

	for hostIDBatch := range hostIDs {
		if len(hostIDBatch) == 0 {
			return nil, nil
		}
		hostDetailBatch, err = getHostsDetails(client, hostIDBatch)
		if err != nil {
			log.Error(falcon.ErrorExplain(err))
			return nil, err
		}
		break
	}
	return hostDetailBatch[0], nil
}

func queryIntelIndicators(client *client.CrowdStrikeAPISpecification, filter, sort *string) (<-chan *models.DomainPublicIndicatorV3, <-chan error) {
	indicatorsChannel := make(chan *models.DomainPublicIndicatorV3)
	errorChannel := make(chan error)

	go func() {
		limit := int64(1000)
		var err error

		for response := (*intel.QueryIntelIndicatorEntitiesOK)(nil); response.HasNextPage(); {
			response, err = client.Intel.QueryIntelIndicatorEntities(&intel.QueryIntelIndicatorEntitiesParams{
				Context: context.Background(),
				Filter:  filter,
				Sort:    sort,
				Limit:   &limit,
			},
				response.Paginate(),
			)
			if err != nil {
				errorChannel <- err
			}
			if response == nil || response.Payload == nil {
				break
			}

			if err = falcon.AssertNoError(response.Payload.Errors); err != nil {
				errorChannel <- err
			}

			indicators := response.Payload.Resources
			for _, indicator := range indicators {
				indicatorsChannel <- indicator
			}
		}
		close(indicatorsChannel)
		close(errorChannel)
	}()
	return indicatorsChannel, errorChannel
}

func getHostsDetails(client *client.CrowdStrikeAPISpecification, hostIds []string) ([]*models.DomainDeviceSwagger, error) {
	response, err := client.Hosts.GetDeviceDetails(&hosts.GetDeviceDetailsParams{
		Ids:     hostIds,
		Context: context.Background(),
	})
	if err != nil {
		log.Error(falcon.ErrorExplain(err))
		return nil, err
	}
	if err = falcon.AssertNoError(response.Payload.Errors); err != nil {
		log.Error(falcon.ErrorExplain(err))
		return nil, err
	}

	return response.Payload.Resources, nil
}

func getHostIds(client *client.CrowdStrikeAPISpecification, filter *string) (<-chan []string, error) {
	hostIds := make(chan []string)

	var err error
	err = nil
	go func() {
		limit := int64(500)
		for offset := int64(0); ; {
			response, err := client.Hosts.QueryDevicesByFilter(&hosts.QueryDevicesByFilterParams{
				Limit:   &limit,
				Offset:  &offset,
				Filter:  filter,
				Context: context.Background(),
			})
			if err != nil {
				log.Error(falcon.ErrorExplain(err))
			}
			if err = falcon.AssertNoError(response.Payload.Errors); err != nil {
				log.Error(falcon.ErrorExplain(err))
			}

			hosts := response.Payload.Resources
			hostIds <- hosts
			offset = offset + int64(len(hosts))
			if offset >= *response.Payload.Meta.Pagination.Total {
				break
			}
		}
		close(hostIds)
	}()
	return hostIds, err
}
