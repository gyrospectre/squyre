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
	Provider = "IP API"
	BaseURL  = "http://ip-api.com/json"
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

func HandleRequest(ctx context.Context, subject hellarad.Subject) (string, error) {
	var result = hellarad.Result{
		Source:         Provider,
		AttributeValue: subject.IP,
		Success:        false,
	}

	response, err := http.Get(fmt.Sprintf("%s/%s", strings.TrimSuffix(BaseURL, "/"), subject.IP))

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

	resultJson, _ := json.Marshal(result)

	return string(resultJson), nil
}

func main() {
	lambda.Start(HandleRequest)
}
