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
	SecretLocation = "JiraApi"
	BaseURL        = "https://gyrospectre-jira.atlassian.net"
	Project        = "SEC"
	TicketType     = "Task"
	CreateTicket   = true
)

type apiKeySecret struct {
	User string `json:"user"`
	Key  string `json:"apikey"`
}

func createIssue(client *jira.Client, summary string, description string) (string, error) {
	i := jira.Issue{
		Fields: &jira.IssueFields{
			Description: description,
			Summary:     summary,
			Type: jira.IssueType{
				Name: TicketType,
			},
			Project: jira.Project{
				Key: Project,
			},
		},
	}

	issue, _, err := client.Issue.Create(&i)
	if err != nil {
		return "", err
	}

	return issue.Key, nil
}

func HandleRequest(ctx context.Context, rawAlerts []string) (string, error) {
	// Fetch API key from Secrets Manager
	smresponse, err := hellarad.GetSecret(SecretLocation)
	if err != nil {
		log.Fatalf("Failed to fetch Jira secret: %s", err)
	}
	var secret apiKeySecret
	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	tp := jira.BasicAuthTransport{
		Username: secret.User,
		Password: secret.Key,
	}

	jiraClient, err := jira.NewClient(tp.Client(), BaseURL)

	if err != nil {
		panic(err)
	}
	var ticketnumber string
	if CreateTicket {
		ticketnumber, err = createIssue(jiraClient, "Test Ticket", "Just testing.")
		if err != nil {
			panic(err)
		}
		log.Printf("Created ticket number %s", ticketnumber)
	}
	// Process enrichment result list
	for _, alertStr := range rawAlerts {
		var alert hellarad.Alert
		json.Unmarshal([]byte(alertStr), &alert)

		log.Printf("Sending results of successful enrichment for alert %s", alert.Id)

		for _, result := range alert.Results {
			// Only send the output of successful enrichments
			if result.Success {
				comment := jira.Comment{
					Body: fmt.Sprintf("Additional information on %s from %s:\n\n%s", result.AttributeValue, result.Source, result.Message),
				}
				if !CreateTicket {
					ticketnumber = alert.Id
				}
				_, _, err := jiraClient.Issue.AddComment(ticketnumber, &comment)
				if err != nil {
					panic(err)
				}
				log.Printf("Added comment to ticket number %s", ticketnumber)
			} else {
				log.Printf("Skipping failed enrichment from %s for alert %s", result.Source, alert.Id)
			}
		}
	}

	return "Success", nil
}

func main() {
	lambda.Start(HandleRequest)
}
