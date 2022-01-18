package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre"
)

const (
	secretLocation = "OpsGenieAPI"
	baseURL        = "https://api.opsgenie.com/v2"
)

var (
	// InitClient abstracts this function to allow for tests
	InitClient = InitOpsgenieClient
	// AddComment abstracts this function to allow for tests
	AddComment = AddNoteToAlert
)

// OpsGenieClient wraps a HTTP client with the token used to auth to Opsgenie
type OpsGenieClient struct {
	client   *http.Client
	apiToken string
}

type apiKeySecret struct {
	Key string `json:"apikey"`
}

type opsgenieNote struct {
	User   string `json:"user"`
	Source string `json:"source"`
	Note   string `json:"note"`
}

// Post wraps a standard http Post call with the required auth headers
func (opsgenie *OpsGenieClient) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", fmt.Sprintf("GenieKey %s", opsgenie.apiToken))

	return opsgenie.client.Do(req)
}

// InitOpsgenieClient initialises an Opsgenie client using credentials from AWS Secrets Manager
func InitOpsgenieClient() (*OpsGenieClient, error) {
	// Fetch API key from Secrets Manager
	smresponse, err := squyre.GetSecret(secretLocation)
	if err != nil {
		log.Errorf("Failed to fetch OpsGenie secret: %s", err)
	}
	var secret apiKeySecret
	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	return &OpsGenieClient{
		client:   &http.Client{},
		apiToken: secret.Key,
	}, nil
}

// AddNoteToAlert adds a comment to an existing Opsgenie alert
func AddNoteToAlert(client *OpsGenieClient, note *opsgenieNote, id string) error {
	// https://docs.opsgenie.com/docs/alert-api#add-note-to-alert
	ogurl := fmt.Sprintf("%s/alerts/%s/notes", strings.TrimSuffix(baseURL, "/"), id)

	jsonData, err := json.Marshal(note)
	if err != nil {
		return err
	}

	response, err := client.Post(ogurl, "application/json; charset=UTF-8", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	if response.StatusCode != 202 {
		return errors.New("Unexpected response code")
	}

	return nil
}

func handleRequest(ctx context.Context, rawAlerts [][]string) (string, error) {
	client, err := InitClient()
	if err != nil {
		log.Error("Failed to initialise client")
		return "Failed to initialise client", err
	}

	// We have separate alerts by source, combine them first to prevent creating duplicate tickets
	mergedAlerts := squyre.CombineResultsbyAlertID(rawAlerts)
	log.Infof("Merged alerts. Was %d result groups, now %d individual results.", len(rawAlerts), len(mergedAlerts))

	var alerts []string
	// Process enrichment result list
	for _, alert := range mergedAlerts {

		if len(alert.Results) == 0 {
			return "No results found to process", nil
		}

		log.Infof("Sending results of successful enrichment for alert %s", alert.ID)

		for _, result := range alert.Results {
			// Only send the output of successful enrichments
			if result.Success {
				note := &opsgenieNote{
					User:   "Squyre",
					Source: result.Source,
					Note:   fmt.Sprintf("Additional information on %s from %s:\n\n%s", result.AttributeValue, result.Source, result.Message),
				}

				err := AddComment(client, note, alert.ID)
				if err != nil {
					log.Errorf("Failed to add comment to alert '%s'", alert.ID)
					return "Failed to add comment to alert", err
				}
				log.Info("Successfully adding note to OpsGenie")
			} else {
				log.Errorf("Skipping failed enrichment from %s for alert %s", result.Source, alert.ID)
			}

		}
		alerts = append(alerts, alert.ID)
	}
	sort.Strings(alerts)
	finalResult := fmt.Sprintf(
		"Success: %d alerts processed (%d groups). Updated alerts: %s",
		len(mergedAlerts),
		len(rawAlerts),
		alerts,
	)

	log.Info(finalResult)

	return finalResult, nil
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}
