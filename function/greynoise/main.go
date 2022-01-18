package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre"
)

const (
	provider = "GreyNoise"
	baseURL  = "https://api.greynoise.io/v3/community"
	supports = "ipv4"
)

var (
	// Client defines an abstracted HTTP client to allow for tests
	Client         HTTPClient
	responseObject greynoiseResponse
)

func init() {
	Client = &http.Client{}
}

// HTTPClient interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
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

func handleRequest(ctx context.Context, alert squyre.Alert) (string, error) {
	log.Infof("Starting %s run for alert %s", provider, alert.ID)

	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {
		if strings.Contains(supports, subject.Type) {

			// Build a result object to hold our goodies
			var result = squyre.Result{
				Source:         provider,
				AttributeValue: subject.Value,
				Success:        false,
			}

			request, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), subject.Value), nil)
			response, err := Client.Do(request)

			if err != nil {
				log.Errorf("Failed to fetch data from %s", provider)
				return "Error fetching data from API!", err
			}
			responseData, err := ioutil.ReadAll(response.Body)

			if err == nil {
				log.Infof("Received %s response for %s", provider, subject.Value)

				json.Unmarshal(responseData, &responseObject)
				prettyresponse, _ := json.MarshalIndent(responseObject, "", "    ")

				result.Success = true
				result.Message = string(prettyresponse)
			} else {
				log.Errorf("Unexpected response from %s for %s", provider, subject.Value)
				return "Error decoding response from API!", err
			}
			// Add the enriched details back to the results
			alert.Results = append(alert.Results, result)
			log.Infof("Added %s to result set", subject.Value)
		} else {
			log.Error("Subject not supported by this provider. Skipping.")
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
