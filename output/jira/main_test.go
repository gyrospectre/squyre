package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/gyrospectre/hellarad"
	//"net/http"
	"testing"
)

var (
	// MockTicket is a fake ticket for tests
	MockTicket int
	Ctx        context.Context
)

func setup() {
	// Mock out calls to real things
	InitClient = mockInitClient
	CreateTicketForAlert = mockCreateTicketForAlert
	AddComment = mockAddComment

	// By default we create tickets for each alert
	CreateTicket = true

	// Reset fake ticket number count
	MockTicket = 1
}

func mockInitClient() (*jira.Client, error) {
	return &jira.Client{}, nil
}

func mockGetSecret(location string) (secretsmanager.GetSecretValueOutput, error) {
	secret := `{"user":"test","apikey": "test123"}`
	return secretsmanager.GetSecretValueOutput{
		SecretString: &secret,
	}, nil
}

func mockCreateTicketForAlert(client *jira.Client, alert hellarad.Alert) (string, error) {
	ticketnumber := fmt.Sprintf("CREATED-%d", MockTicket)
	MockTicket = MockTicket + 1
	return ticketnumber, nil
}

func mockAddComment(client *jira.Client, ticket string, rawComment string) error {
	return nil
}

func makeTestAlerts(number int, prefix string, includeResults bool) ([]string, []string) {
	alert := hellarad.Alert{
		RawMessage: "Testing",
	}
	if includeResults {
		alert.Results = []hellarad.Result{
			{
				Source:         "Gyro",
				AttributeValue: "127.0.0.1",
				Message:        "Test",
				Success:        true,
			},
		}
	}

	var alerts []string
	var alertlist []string
	for i := 1; i <= number; i++ {
		alert.ID = fmt.Sprintf("%s%d", prefix, i)

		alertlist = append(alertlist, alert.ID)
		alertJSON, _ := json.Marshal(alert)
		alerts = append(alerts, string(alertJSON))
	}

	return alerts, alertlist
}

// tests Handler when Create is set
func TestHandlerCreateSuccess(t *testing.T) {
	setup()

	alerts, _ := makeTestAlerts(5, "EXISTING-", true)
	output, err := handleRequest(Ctx, alerts)

	var alertList []string
	for i := 1; i <= 5; i++ {
		alertList = append(alertList, fmt.Sprintf("%s-%d", "CREATED", i))
	}
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	have := string(output)

	want := fmt.Sprintf("Success: %d alerts processed. Created alerts: %s", len(alerts), alertList)

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

// tests Handler when Create is not set
func TestHandlerNoCreateSuccess(t *testing.T) {
	setup()
	CreateTicket = false

	alerts, alertlist := makeTestAlerts(3, "EXISTING-", true)
	output, err := handleRequest(Ctx, alerts)

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	have := string(output)
	want := fmt.Sprintf("Success: %d alerts processed. Updated alerts: %s", len(alerts), alertlist)

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

func TestHandlerNoResults(t *testing.T) {
	setup()

	alerts, _ := makeTestAlerts(3, "EXISTING-", false)

	output, err := handleRequest(Ctx, alerts)

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	have := string(output)
	want := "No results found to process"

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
