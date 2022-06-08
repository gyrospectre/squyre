package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/andygrunwald/go-jira"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre/pkg/squyre"
)

const (
	secretLocation = "JiraApi"
	ticketType     = "Task"
)

var (
	baseURL = os.Getenv("BASE_URL")
	project = os.Getenv("PROJECT")

	// CreateTicketForAlert abstracts this function to allow for tests
	CreateTicketForAlert = CreateJiraIssueForAlert
	// InitClient abstracts this function to allow for tests
	InitClient = InitJiraClient
	// AddComment abstracts this function to allow for tests
	AddComment = AddJiraComment
	// CreateTicket set to true if we are to create new issues for each alert
	CreateTicket = true
)

type apiKeySecret struct {
	User string `json:"user"`
	Key  string `json:"apikey"`
}

// CreateJiraIssueForAlert creates a Jira issue with details of the supplied alert object
func CreateJiraIssueForAlert(client *jira.Client, alert squyre.Alert) (string, error) {
	i := jira.Issue{
		Fields: &jira.IssueFields{
			Description: fmt.Sprintf("For full details: %s", alert.URL),
			Summary:     fmt.Sprintf("Alert - %s", alert.Name),
			Type: jira.IssueType{
				Name: ticketType,
			},
			Project: jira.Project{
				Key: project,
			},
		},
	}
	issue, _, err := client.Issue.Create(&i)
	if err != nil {
		return "", err
	}

	return issue.Key, nil
}

// AddJiraComment adds a note to an existing Jira issue
func AddJiraComment(client *jira.Client, ticket string, rawComment string) error {
	comment := jira.Comment{
		Body: rawComment,
	}
	_, _, err := client.Issue.AddComment(ticket, &comment)

	return err
}

// InitJiraClient initialises a Jira client using credentials from AWS Secrets Manager
func InitJiraClient() (*jira.Client, error) {
	// Fetch API key from Secrets Manager
	smresponse, err := squyre.GetSecret(secretLocation)
	if err != nil {
		log.Error("Failed to fetch Jira secret")
		return nil, err
	}

	var secret apiKeySecret
	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	// Connect to Jira Cloud
	tp := jira.BasicAuthTransport{
		Username: secret.User,
		Password: secret.Key,
	}

	jiraClient, err := jira.NewClient(tp.Client(), baseURL)
	if err != nil {
		return nil, err
	}

	return jiraClient, nil
}

func handleRequest(ctx context.Context, rawAlerts [][]string) (string, error) {
	jiraClient, err := InitClient()
	if err != nil {
		log.Error("Failed to initialise client")
		return "Failed to initialise client", err
	}

	// We have separate alerts by source, combine them first to prevent creating duplicate tickets
	mergedAlerts := squyre.CombineResultsbyAlertID(rawAlerts)
	log.Infof("Merged alerts. Was %d result groups, now %d individual results.", len(rawAlerts), len(mergedAlerts))

	// Process enrichment result list
	var ticketnumber string
	var ticketnumbers []string
	var action string

	for _, alert := range mergedAlerts {
		if CreateTicket {
			ticketnumber, err = CreateTicketForAlert(jiraClient, alert)
			if err != nil {
				log.Error("Failed to create ticket")
				return "Failed to create ticket", err
			}
			action = "Create"

			log.Infof("Created ticket number %s", ticketnumber)
		} else {
			ticketnumber = alert.ID
			action = "Update"
		}

		if len(alert.Results) == 0 {
			return "No results found to process", nil
		}

		log.Infof("Sending results of enrichment to %s", ticketnumber)

		for _, result := range alert.Results {
			if result.Success {
				err = AddComment(jiraClient, ticketnumber, fmt.Sprintf("Additional information on %s from %s:\n\n%s", result.AttributeValue, result.Source, result.Message))
				if err != nil {
					log.Errorf("Failed to add comment to ticket %s", ticketnumber)
					return "Failed to add comment to ticket", err
				}
			} else {
				err = AddComment(jiraClient, ticketnumber, fmt.Sprintf("Error looking up %s on %s!\nError: %s", result.AttributeValue, result.Source, result.Message))
				if err != nil {
					log.Errorf("Failed to add comment to ticket %s", ticketnumber)
					return "Failed to add comment to ticket", err
				}
			}
		}
		ticketnumbers = append(ticketnumbers, ticketnumber)
	}
	sort.Strings(ticketnumbers)
	finalResult := fmt.Sprintf(
		"Success: %d alerts processed (%d groups). %sd alerts: %s",
		len(mergedAlerts),
		len(rawAlerts),
		action,
		ticketnumbers,
	)
	log.Info(finalResult)

	return finalResult, nil
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}
