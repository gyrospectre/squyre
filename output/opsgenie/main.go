package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/hellarad"
	"log"
	"net/http"
	"strings"
)

const (
	SecretLocation = "OpsGenieAPI"
	BaseURL        = "https://api.opsgenie.com/v2"
)

type apiKeySecret struct {
	Key string `json:"apikey"`
}

type opsgenieNote struct {
	User   string `json:"user"`
	Source string `json:"source"`
	Note   string `json:"note"`
}

func HandleRequest(ctx context.Context, outputs [][]string) (string, error) {
	log.Print("Starting")
	var secret apiKeySecret
	alertId := "a9ff96ea-3e45-41ee-bffa-b136f7de84d7-1640917932111"

	smresponse, err := hellarad.GetSecret(SecretLocation)
	if err != nil {
		log.Fatalf("Failed to fetch OpsGenie secret: %s", err)
	}

	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	// https://docs.opsgenie.com/docs/alert-api#add-note-to-alert
	ogurl := fmt.Sprintf("%s/alerts/%s/notes", strings.TrimSuffix(BaseURL, "/"), alertId)
	auth := fmt.Sprintf("GenieKey %s", secret.Key)

	for _, output := range outputs {
		for _, resultStr := range output {
			var result hellarad.Result
			json.Unmarshal([]byte(resultStr), &result)

			note := &opsgenieNote{
				User:   "Hella Rad!",
				Source: result.Source,
				Note:   fmt.Sprintf("Additional information on %s from %s:\n\n%s", result.AttributeValue, result.Source, result.Message),
			}
			jsonData, err := json.Marshal(note)
			if err != nil {
				log.Fatalf("Could not marshal JSON into Note: %s", err)
			}

			request, _ := http.NewRequest("POST", ogurl, bytes.NewBuffer(jsonData))
			request.Header.Set("Content-Type", "application/json; charset=UTF-8")
			request.Header.Set("Authorization", auth)

			client := &http.Client{}
			response, err := client.Do(request)
			if err != nil {
				log.Fatalf("Error posting data to OpsGenie: %s", err)
			}
			defer response.Body.Close()
		}
	}

	return "Success", nil
}

func main() {
	lambda.Start(HandleRequest)
}
