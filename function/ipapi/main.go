package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre"
)

const (
	provider       = "IP API"
	baseURL        = "http://ip-api.com/json"
	supportsIP     = true
	supportsDomain = false
)

var (
	// Client defines an abstracted HTTP client to allow for tests
	Client         HTTPClient
	responseObject ipapiResponse
)

func init() {
	Client = &http.Client{}
}

// HTTPClient interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ipapiResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Region      string `json:"region"`
	RegionName  string `json:"regionName"`
	City        string `json:"city"`
	Latitude    string `json:"lat"`
	Longitude   string `json:"lon"`
	Timezone    string `json:"timezone"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	ASN         string `json:"as"`
}

func isSupported(sub squyre.Subject) bool {
	supported := false
	if sub.IP != "" && supportsIP {
		supported = true
	}
	if sub.Domain != "" && supportsDomain {
		supported = true
	}
	return supported
}

func handleRequest(ctx context.Context, alert squyre.Alert) (string, error) {
	log.Printf("Starting %s run for alert %s", provider, alert.ID)

	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {
		if isSupported(subject) {

			// Build a result object to hold our goodies
			var result = squyre.Result{
				Source:         provider,
				AttributeValue: subject.IP,
				Success:        false,
			}

			request, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), subject.IP), nil)
			response, err := Client.Do(request)

			if err != nil {
				log.Printf("Failed to fetch data from %s", provider)
				return "Error fetching data from API!", err
			}
			responseData, err := ioutil.ReadAll(response.Body)

			if err == nil {
				log.Printf("Received %s response for %s", provider, subject.IP)

				json.Unmarshal(responseData, &responseObject)
				prettyresponse, _ := json.MarshalIndent(responseObject, "", "    ")

				result.Success = true
				result.Message = string(prettyresponse)
			} else {
				log.Printf("Unexpected response from %s for %s", provider, subject.IP)
				return "Error decoding response from API!", err
			}
			// Add the enriched details back to the results
			alert.Results = append(alert.Results, result)
			log.Printf("Added %s to result set", subject.IP)
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
