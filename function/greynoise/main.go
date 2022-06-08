package main

import (
	"context"
	"encoding/json"
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
	provider = "GreyNoise"
	baseURL  = "https://api.greynoise.io/v3/community"
	supports = "ipv4"
)

var (
	// GetIPInfo abstracts this function to allow for tests
	GetIPInfo         = getIPInfo
	responseObject    greynoiseResponse
	InitClient        = initGreynoiseClient
	OnlyLogMatches, _ = strconv.ParseBool(os.Getenv("ONLY_LOG_MATCHES"))
)

var template = `
Greynoise believes %s is %s.

Noise? %t
In the RIOT database? %t
Last seen %s.

More information at: %s

`

type apiClient struct {
	httpClient *http.Client
	baseURL    string
}

type greynoiseResponse struct {
	IP             string `json:"ip"`
	Noise          bool   `json:"noise"`
	Riot           bool   `json:"riot"`
	Classification string `json:"classification"`
	Name           string `json:"name"`
	Link           string `json:"link"`
	LastSeen       string `json:"last_seen"`
	Message        string `json:"message"`
}

func initGreynoiseClient() (*apiClient, error) {
	client := &apiClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}

	return client, nil
}

func getIPInfo(c *apiClient, ipv4 string) (*http.Response, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s", c.baseURL, ipv4),
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
		if strings.Contains(supports, subject.Type) {

			// Build a result object to hold our goodies
			var result = squyre.Result{
				Source:         provider,
				AttributeValue: subject.Value,
				MatchFound:     false,
				Success:        false,
			}

			response, err := GetIPInfo(client, subject.Value)

			if err != nil {
				log.Errorf("Failed to fetch data from %s", provider)
				result.Success = false
				result.Message = err.Error()
				alert.Results = append(alert.Results, result)
			} else {
				result.Success = true
				responseData, err := ioutil.ReadAll(response.Body)

				if err == nil {
					log.Infof("Received %s response for %s", provider, subject.Value)

					json.Unmarshal(responseData, &responseObject)

					if responseObject.Classification != "" {
						// A blank classification means nothing was found
						result.MatchFound = true
					} else {
						result.MatchFound = false
					}

					if !result.MatchFound && OnlyLogMatches {
						log.Infof("Skipping non match for %s", subject.Value)
					} else {
						// Match found. Add the enriched details back to the results
						result.Message = messageFromResponse(responseObject)
						alert.Results = append(alert.Results, result)
						log.Infof("Added %s to result set", subject.Value)
					}

					// Clear results ready for next cycle
					responseObject.Classification = ""
					responseObject.Name = ""
					responseObject.Link = ""
					responseObject.LastSeen = ""
				} else {
					log.Errorf("Unexpected response from %s for %s", provider, subject.Value)
					return "Error decoding response from API!", err
				}
			}
		} else {
			log.Info("Subject not supported by this provider. Skipping.")
		}
	}

	log.Infof("Successfully ran %s. Yielded %d results for %d subjects.", provider, len(alert.Results), len(alert.Subjects))
	// Convert the alert object into Json for the step function
	finalJSON, _ := json.Marshal(alert)
	return string(finalJSON), nil
}

func messageFromResponse(response greynoiseResponse) string {
	if response.Classification == "" {
		return response.Message
	}

	message := fmt.Sprintf(template,
		response.IP,
		response.Classification,
		response.Noise,
		response.Riot,
		response.LastSeen,
		response.Link,
	)

	return string(message)
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}
