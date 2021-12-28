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
	response, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s", subject.IP))

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

	message := fmt.Sprintf("Details on %s from Greynoise:\n", subject.IP)
	return message+string(j), nil
}

func main() {
	lambda.Start(HandleRequest)
}