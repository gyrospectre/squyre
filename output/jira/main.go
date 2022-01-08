package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/andygrunwald/go-jira"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre"
)

const (
	secretLocation = "JiraApi"
	baseURL        = "https://your-jira.atlassian.net"
	project        = "SECURITY"
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

func combineResultsbyAlertID(raw []string) map[string]squyre.Alert {
	resultsmap := make(map[string][]squyre.Result)
	alerts := make(map[string]squyre.Alert)

	for _, alertStr := range raw {
		var alert squyre.Alert
		json.Unmarshal([]byte(alertStr), &alert)
		for _, result := range alert.Results {
			resultsmap[alert.ID] = append(resultsmap[alert.ID], result)
		}
		alert.Results = nil
		alerts[alert.ID] = alert
	}
	for id, results := range resultsmap {
		temp := alerts[id]
		temp.Results = results
		alerts[id] = temp
	}
	return alerts
}

func handleRequest(ctx context.Context, rawAlerts []string) (string, error) {
	jiraClient, err := InitClient()
	if err != nil {
		panic(err)
	}

	// We have separate alerts by source, combine them first to prevent creating duplicate tickets
	mergedAlerts := combineResultsbyAlertID(rawAlerts)

	// Process enrichment result list
	var ticketnumber string
	var ticketnumbers []string
	var action string

	for _, alert := range mergedAlerts {
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
		ticketnumbers = append(ticketnumbers, ticketnumber)
	}
	finalResult := fmt.Sprintf("Success: %d alerts processed. %sd alerts: %s", len(mergedAlerts), action, ticketnumbers)
	log.Print(finalResult)

	return finalResult, nil
}

func main() {
	lambda.Start(handleRequest)
}
