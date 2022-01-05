package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/hellarad"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const (
	secretLocation = "OpsGenieAPI"
	baseURL        = "https://api.opsgenie.com/v2"
)

type apiKeySecret struct {
	Key string `json:"apikey"`
}

type opsgenieNote struct {
	User   string `json:"user"`
	Source string `json:"source"`
	Note   string `json:"note"`
}

func handleRequest(ctx context.Context, rawAlerts []string) (string, error) {
	// Fetch API key from Secrets Manager
	smresponse, err := hellarad.GetSecret(secretLocation)
	if err != nil {
		log.Fatalf("Failed to fetch OpsGenie secret: %s", err)
	}
	var secret apiKeySecret
	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	// Process enrichment result list
	for _, alertStr := range rawAlerts {
		var alert hellarad.Alert
		json.Unmarshal([]byte(alertStr), &alert)

		log.Printf("Sending results of successful enrichment for alert %s", alert.Id)

		// https://docs.opsgenie.com/docs/alert-api#add-note-to-alert
		ogurl := fmt.Sprintf("%s/alerts/%s/notes", strings.TrimSuffix(baseURL, "/"), alert.Id)
		auth := fmt.Sprintf("GenieKey %s", secret.Key)

		for _, result := range alert.Results {
			// Only send the output of successful enrichments
			if result.Success {
				note := &opsgenieNote{
					User:   "Hella Rad!",
					Source: result.Source,
					Note:   fmt.Sprintf("Additional information on %s from %s:\n\n%s", result.AttributeValue, result.Source, result.Message),
				}
				jsonData, err := json.Marshal(note)
				if err != nil {
					log.Fatalf("Could not marshal Note into JSON: %s", err)
				}

				request, _ := http.NewRequest("POST", ogurl, bytes.NewBuffer(jsonData))
				request.Header.Set("Content-Type", "application/json; charset=UTF-8")
				request.Header.Set("Authorization", auth)

				client := &http.Client{}
				response, err := client.Do(request)
				if err != nil {
					log.Fatalf("Error posting data to OpsGenie: %s", err)
				}
				respBody, _ := ioutil.ReadAll(response.Body)
				if response.StatusCode != 202 {
					return string(respBody), err
				}
				log.Printf("Sent note to OpsGenie with result %s", respBody)

				defer response.Body.Close()
			} else {
				log.Printf("Skipping failed enrichment from %s for alert %s", result.Source, alert.Id)
			}

		}
	}

	return "Success", nil
}

func main() {
	lambda.Start(handleRequest)
}
