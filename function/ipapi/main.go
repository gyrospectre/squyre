package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/hellarad"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	provider = "IP API"
	baseURL  = "http://ip-api.com/json"
)

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

func handleRequest(ctx context.Context, alert hellarad.Alert) (string, error) {
	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {

		// Build a result object to hold our goodies
		var result = hellarad.Result{
			Source:         provider,
			AttributeValue: subject.IP,
			Success:        false,
		}

		response, err := http.Get(fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), subject.IP))

		if err != nil {
			return "Error fetching data from IP API!", err
		}

		responseData, err := ioutil.ReadAll(response.Body)
		if err == nil {
			var responseObject ipapiResponse
			json.Unmarshal(responseData, &responseObject)
			prettyresponse, _ := json.MarshalIndent(responseObject, "", "    ")

			result.Success = true
			result.Message = string(prettyresponse)
		} else {
			return "Error decoding response from IP API!", err
		}
		// Add the enriched details back to the results
		alert.Results = append(alert.Results, result)
	}

	// Convert the alert object into Json for the step function
	finalJSON, _ := json.Marshal(alert)
	return string(finalJSON), nil
}

func main() {
	lambda.Start(handleRequest)
}
