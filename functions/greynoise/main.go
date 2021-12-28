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

type Response struct {
    Noise      		bool	`json:"noise"`
	Riot			bool	`json:"riot"`
	Message			string	`json:"message"`
	Classification	string 	`json:"classification"`
	Link			string 	`json:"link"`
	LastSeen		string	`json:"last_seen"`
}

func HandleRequest(ctx context.Context, subject hellarad.Subject) (string, error) {
	var result hellarad.Result

	response, err := http.Get(fmt.Sprintf("https://api.greynoise.io/v3/community/%s", subject.IP))

    if err != nil {
        fmt.Print(err.Error())
        os.Exit(1)
    }

	responseData, err := ioutil.ReadAll(response.Body)
    if err != nil {
        log.Fatal(err)
    }

	var responseObject Response
	json.Unmarshal(responseData, &responseObject)
	j, _ := json.MarshalIndent(responseObject, "", "\t")

	result.Message = string(j)
	result.Success = true 
	return result.stringify(), nil
}

func main() {
	lambda.Start(HandleRequest)
}