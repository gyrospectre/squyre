package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"encoding/json"
	"fmt"
	"net/http"
	"io/ioutil"
	"strings"
	"github.com/gyrospectre/hellarad"
)

const (
	Provider = "IP API"
	BaseURL = "http://ip-api.com/json"
)

type ipapiResponse struct {
	Status      string	`json:"status"`
	Country		string	`json:"country"`
	CountryCode string	`json:"countryCode"`
	Region 	 	string	`json:"region"`
	RegionName	string	`json:"regionName"`
	City		string	`json:"city"`
	Latitude	string	`json:"lat"`
	Longitude	string	`json:"lon"`
	Timezone	string	`json:"timezone"`
	ISP			string	`json:"isp"`
	Org			string	`json:"org"`
	ASN			string	`json:"as"`
}

func HandleRequest(ctx context.Context, subject hellarad.Subject) (string, error) {
	var result = hellarad.Result{Source: Provider, AttributeValue: subject.IP, Success: false}

	response, err := http.Get(fmt.Sprintf("%s/%s", strings.TrimSuffix(BaseURL, "/"), subject.IP))

	if err == nil {
		responseData, err := ioutil.ReadAll(response.Body)
		if err == nil {
			var responseObject ipapiResponse
			json.Unmarshal(responseData, &responseObject)
			prettyresponse, _ := json.MarshalIndent(responseObject, "", "\t")

			result.Success = true
			result.Message = string(prettyresponse)
		} else {
			fmt.Print("Error decoding response!")
			fmt.Print(string(err.Error()))
		}
	} else {
		fmt.Print("Error fetching data from API!")
		fmt.Print(string(err.Error()))
	}

	if err != nil {
		result.Message = string(err.Error())
		return result.Prettify(), err
	} else {
		return result.Prettify(), nil
	}
}

func main() {
	lambda.Start(HandleRequest)
}