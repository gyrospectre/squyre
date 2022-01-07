package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/hellarad"
	"log"
)

const (
	secretLocation = "JiraApi"
	baseURL        = "https://your-jira.atlassian.net"
	project        = "SEC"
	ticketType     = "Task"
)

var (
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
func CreateJiraIssueForAlert(client *jira.Client, alert hellarad.Alert) (string, error) {
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
	smresponse, err := hellarad.GetSecret(secretLocation)
	if err != nil {
		log.Fatalf("Failed to fetch Jira secret: %s", err)
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

func handleRequest(ctx context.Context, rawAlerts []string) (string, error) {
	jiraClient, err := InitClient()
	if err != nil {
		panic(err)
	}

	// Process enrichment result list
	var ticketnumber string
	var ticketnumbers []string
	var action string

	for _, alertStr := range rawAlerts {
		var alert hellarad.Alert
		json.Unmarshal([]byte(alertStr), &alert)

		if CreateTicket {
			ticketnumber, err = CreateTicketForAlert(jiraClient, alert)
			if err != nil {
				panic(err)
			}
			action = "Create"

			log.Printf("Created ticket number %s", ticketnumber)
		} else {
			ticketnumber = alert.ID
			action = "Update"
		}

		if len(alert.Results) == 0 {
			return "No results found to process", nil
		}

		log.Printf("Sending results of successful enrichments to %s", ticketnumber)

		for _, result := range alert.Results {
			// Only send the output of successful enrichments
			if result.Success {
				err = AddComment(jiraClient, ticketnumber, fmt.Sprintf("Additional information on %s from %s:\n\n%s", result.AttributeValue, result.Source, result.Message))
				if err != nil {
					panic(err)
				}
			} else {
				log.Printf("Skipping failed enrichment from %s for alert %s", result.Source, alert.ID)
			}
		}
		fmt.Println(ticketnumber)
		ticketnumbers = append(ticketnumbers, ticketnumber)
	}
	return fmt.Sprintf("Success: %d alerts processed. %sd alerts: %s", len(rawAlerts), action, ticketnumbers), nil
}

func main() {
	lambda.Start(handleRequest)
}
