package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"log"
	"io/ioutil"
	"github.com/gyrospectre/hellarad"
)

type greynoiseResponse struct {
    Noise      		bool	`json:"noise"`
	Riot			bool	`json:"riot"`
	Message			string	`json:"message"`
	Classification	string 	`json:"classification"`
	Link			string 	`json:"link"`
	LastSeen		string	`json:"last_seen"`
}

func HandleRequest(ctx context.Context, subject hellarad.Subject) (string, error) {
	var result = hellarad.Result{Source: "GreyNoise", AttributeValue: subject.IP, Success: false}

	response, err := http.Get(fmt.Sprintf("https://api.greynoise.io/v3/community/%s", subject.IP))

	if err == nil {
		responseData, err := ioutil.ReadAll(response.Body)
		if err == nil {
			var responseObject greynoiseResponse
			json.Unmarshal(responseData, &responseObject)
			prettyresponse, _ := json.MarshalIndent(responseObject, "", "\t")
		
			result.Success = true 		
			result.Message = string(prettyresponse)
		}
	}

	if err != nil {
		result.Message = string(err.Error())
	}

	return result.Prettify(), nil
}

func main() {
	lambda.Start(HandleRequest)
}