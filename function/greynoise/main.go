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
	Provider = "GreyNoise"
	BaseURL  = "https://api.greynoise.io/v3/community"
)

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

func HandleRequest(ctx context.Context, alert hellarad.Alert) (string, error) {
	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {

		// Build a result object to hold our goodies
		var result = hellarad.Result{
			Source:         Provider,
			AttributeValue: subject.IP,
			Success:        false,
		}

		response, err := http.Get(fmt.Sprintf("%s/%s", strings.TrimSuffix(BaseURL, "/"), subject.IP))

		if err != nil {
			return "Error fetching data from Greynoise API!", err
		}

		responseData, err := ioutil.ReadAll(response.Body)
		if err == nil {
			var responseObject greynoiseResponse
			json.Unmarshal(responseData, &responseObject)
			prettyresponse, _ := json.MarshalIndent(responseObject, "", "    ")

			result.Success = true
			result.Message = string(prettyresponse)
		} else {
			return "Error decoding response from Greynoise API!", err
		}
		// Add the enriched details back to the results
		alert.Results = append(alert.Results, result)
	}

	// Convert the alert object into Json for the step function
	finalJson, _ := json.Marshal(alert)
	return string(finalJson), nil
}

func main() {
	lambda.Start(HandleRequest)
}
