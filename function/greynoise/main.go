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

func HandleRequest(ctx context.Context, subject hellarad.Subject) (string, error) {
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

	resultJson, _ := json.Marshal(result)

	return string(resultJson), nil
}

func main() {
	lambda.Start(HandleRequest)
}
