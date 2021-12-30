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
    Provider = "GreyNoise"
    BaseURL = "https://api.greynoise.io/v3/community"
)

type greynoiseResponse struct {
    IP 				string 	`json:"ip"`
    Noise      		bool	`json:"noise"`
    Riot			bool	`json:"riot"`
    Classification	string 	`json:"classification"`
    Name			string	`json:"name"`
    Link			string 	`json:"link"`
    LastSeen		string	`json:"last_seen"`
    Message			string	`json:"message"`
}

func HandleRequest(ctx context.Context, subject hellarad.Subject) (string, error) {
    var result = hellarad.Result{Source: Provider, AttributeValue: subject.IP, Success: false}

    response, err := http.Get(fmt.Sprintf("%s/%s", strings.TrimSuffix(BaseURL, "/"), subject.IP))

    if err == nil {
        responseData, err := ioutil.ReadAll(response.Body)
        if err == nil {
            var responseObject greynoiseResponse
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