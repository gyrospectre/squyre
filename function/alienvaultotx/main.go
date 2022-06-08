package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre/pkg/squyre"
)

const (
	provider    = "Alienvault OTX"
	baseURL     = "https://otx.alienvault.com/api/v1/"
	supports    = "ipv4,domain,url"
	retries     = 3
	timeoutSecs = 10
)

var (
	// GetIPInfo abstracts this function to allow for tests
	GetIndictatorInfo = getOTXIndictatorInfo
	responseObject    otxResponse
	InitClient        = initOTXClient
	OnlyLogMatches, _ = strconv.ParseBool(os.Getenv("ONLY_LOG_MATCHES"))
)

var template = `
Alienvault OTX has %x matches for '%s', in the following pulses:
%s

More information at: https://otx.alienvault.com/browse/global/pulses?q=%s

`

type apiClient struct {
	httpClient *http.Client
	baseURL    string
}

type otxResponse struct {
	Indicator  string `json:"indicator"`
	Reputation int    `json:"reputation"`
	PulseInfo  struct {
		Count  int        `json:"count"`
		Pulses []otxPulse `json:"pulses"`
	} `json:"pulse_info"`
}

type otxPulse struct {
	Id                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Modified          string   `json:"modified"`
	Created           string   `json:"created"`
	Tags              []string `json:"tags"`
	TargetedCountries string   `json:"targeted_countries`
	MalwareFamilies   string   `json:"malware_families`
	Industries        []string `json:"industries`
	TLP               string   `json:"tlp"`
	ModifiedText      string   `json:"modified_text"`
}

func initOTXClient() (*apiClient, error) {
	client := &apiClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Second * timeoutSecs,
		},
	}

	return client, nil
}

func getOTXIndictatorInfo(c *apiClient, indicator string, indicatorType string) (*http.Response, error) {
	if indicatorType == "ipv4" {
		return getOTXIPInfo(c, indicator)
	} else if indicatorType == "domain" {
		return getOTXDomainInfo(c, indicator)
	} else if indicatorType == "url" {
		return getOTXUrlInfo(c, indicator)
	}

	return nil, errors.New("Unknown indicator type")
}

func getOTXIPInfo(c *apiClient, ipv4 string) (*http.Response, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/indicators/IPv4/%s", c.baseURL, ipv4),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Do(request)
}

func getOTXDomainInfo(c *apiClient, domain string) (*http.Response, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/indicators/domain/%s", c.baseURL, domain),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Do(request)
}

func getOTXUrlInfo(c *apiClient, url string) (*http.Response, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/indicators/url/%s/general", c.baseURL, url),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(request)
}

func handleRequest(ctx context.Context, alert squyre.Alert) (string, error) {
	log.Infof("Starting %s run for alert %s", provider, alert.ID)
	log.Infof("OnlyLogMatches is set to %t", OnlyLogMatches)

	if len(alert.Subjects) == 0 {
		log.Info("Alert has no subjects to process.")
		finalJSON, _ := json.Marshal(alert)
		return string(finalJSON), nil
	}

	client, err := InitClient()
	if err != nil {
		return "Failed to initialise client", err
	}

	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {
		if !strings.Contains(supports, subject.Type) {
			log.Info("Subject not supported by this provider. Skipping.")
		} else {
			// Build a result object to hold our goodies
			var result = squyre.Result{
				Source:         provider,
				AttributeValue: subject.Value,
				MatchFound:     false,
				Success:        false,
			}
			var response *http.Response
			var err error
			var attempt int
			for attempt = 1; attempt <= retries; attempt++ {
				log.Infof("Get indicator attempt number %d", attempt)
				response, err = GetIndictatorInfo(client, subject.Value, subject.Type)

				if err == nil {
					break
				}
			}
			if err != nil {
				log.Errorf("Failed to fetch data from %s after %d attempts", provider, attempt-1)
				result.Success = false
				result.Message = err.Error()
				alert.Results = append(alert.Results, result)
			} else {
				log.Infof("Successfully fetched data from %s after %d attempts.", provider, attempt)
				result.Success = true

				responseData, err := ioutil.ReadAll(response.Body)

				if err == nil {
					log.Infof("Received %s response for %s", provider, subject.Value)

					json.Unmarshal(responseData, &responseObject)

					if responseObject.PulseInfo.Count == 0 {
						// Nothing was found
						result.MatchFound = false
					} else {
						result.MatchFound = true
					}

					if !result.MatchFound && OnlyLogMatches {
						log.Infof("Skipping non match for %s", subject.Value)
					} else {
						// Match found. Add the enriched details back to the results
						result.Message = messageFromResponse(responseObject)
						alert.Results = append(alert.Results, result)
						log.Infof("Added %s to result set", subject.Value)
					}
				} else {
					log.Errorf("Unexpected response from %s for %s", provider, subject.Value)
					return "Error decoding response from API!", err
				}
			}
		}
	}
	log.Infof("Finished %s run. Yielded %d results for %d subjects.", provider, len(alert.Results), len(alert.Subjects))

	// Convert the alert object into Json for the step function
	finalJSON, _ := json.Marshal(alert)
	return string(finalJSON), nil
}

func removeDuplicates(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func messageFromResponse(response otxResponse) string {
	if response.PulseInfo.Count == 0 {
		return "Indicator not found in Alienvault OTX."
	}

	var pulses []string
	for _, pulse := range response.PulseInfo.Pulses {
		pulses = append(pulses, pulse.Name)
	}

	message := fmt.Sprintf(template,
		response.PulseInfo.Count,
		response.Indicator,
		strings.Join(removeDuplicates(pulses), "\n"),
		response.Indicator,
	)

	return string(message)
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}
