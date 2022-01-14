package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/intel"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/crowdstrike/gofalcon/pkg/falcon_util"
)

const (
	provider       = "Crowdstrike Falcon"
	baseURL        = "https://api.crowdstrike.com"
	supports       = "ipv4,domain,sha256"
	secretLocation = "CrowdstrikeAPI"
)

var (
	// Client defines an abstracted HTTP client to allow for tests
	Client HTTPClient
	// InitClient abstracts this function to allow for tests
	InitClient = InitFalconClient
)

type apiKeySecret struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	FalconCloud  string `json:"falconCloud"`
}

// HTTPClient interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func init() {
	Client = &http.Client{}
}

// InitFalconClient initialises a Falcon client using credentials from AWS Secrets Manager
func InitFalconClient() (*client.CrowdStrikeAPISpecification, error) {
	// Fetch API key from Secrets Manager
	smresponse, err := squyre.GetSecret(secretLocation)
	if err != nil {
		log.Fatalf("Failed to fetch Crowdstrike Falcon secret: %s", err)
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
	log.Printf("Starting %s run for alert %s", provider, alert.ID)

	falconClient, err := InitClient()
	if err != nil {
		panic(err)
	}
	//falcon_host_details --filter="hostname:'A-host'"

	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {
		if strings.Contains(supports, subject.Type) {

			// Build a result object to hold our goodies
			var result = squyre.Result{
				Source:         provider,
				AttributeValue: subject.Value,
				Success:        false,
			}
			filter := fmt.Sprintf("%s:'%s'", subject.Type, subject.Value)
			fmt.Println("[")
			empty := true
			var prettyJSON string

			indicatorsChannel, errorChannel := queryIntelIndicators(falconClient, &filter, nil)
			for openChannels := 2; openChannels > 0; {
				select {
				case err, ok := <-errorChannel:
					if ok {
						panic(err)
					} else {
						openChannels--
					}
				case indicator, ok := <-indicatorsChannel:
					if ok {
						prettyJSON, err = falcon_util.PrettyJson(indicator)
						if err != nil {
							log.Printf("Failed to fetch data from %s", provider)
							return "Error fetching data from API!", err
						}
						if !empty {
							fmt.Println(",")
						} else {
							empty = false
						}
						fmt.Printf("%s", prettyJSON)
					} else {
						openChannels--
					}
				}
			}
			fmt.Println("]")

			log.Printf("Received %s response for %s", provider, subject.Value)

			result.Success = true
			if !empty {
				result.Message = string(prettyJSON)
			} else {
				result.Message = "Indicator not found."
			}

			// Add the enriched details back to the results
			alert.Results = append(alert.Results, result)
			log.Printf("Added %s to result set", subject.Value)
		} else {
			log.Printf("Subject not supported by this provider. Skipping.")
		}
	}
	log.Printf("Successfully ran %s. Yielded %d results for %d subjects.", provider, len(alert.Results), len(alert.Subjects))

	// Convert the alert object into Json for the step function
	finalJSON, _ := json.Marshal(alert)
	return string(finalJSON), nil
}

func main() {
	lambda.Start(handleRequest)
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
