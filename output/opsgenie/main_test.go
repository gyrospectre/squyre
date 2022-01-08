package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/gyrospectre/squyre"
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
	AddComment = mockAddComment

	// Reset fake ticket number count
	MockTicket = 1
}

func mockInitClient() (*OpsGenieClient, error) {
	return &OpsGenieClient{}, nil
}

func mockGetSecret(location string) (secretsmanager.GetSecretValueOutput, error) {
	secret := `{"user":"test","apikey": "test123"}`
	return secretsmanager.GetSecretValueOutput{
		SecretString: &secret,
	}, nil
}

func mockAddComment(client *OpsGenieClient, note *opsgenieNote, id string) error {
	return nil
}

func makeTestAlerts(number int, prefix string, includeResults bool, sameId bool) ([]string, []string) {
	alert := squyre.Alert{
		RawMessage: "Testing",
	}
	if includeResults {
		alert.Results = []squyre.Result{
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
		if sameId {
			alert.ID = fmt.Sprintf("%s%d", prefix, i)
		} else {
			alert.ID = fmt.Sprintf("%s%d", prefix, i)
		}

		alertlist = append(alertlist, alert.ID)
		alertJSON, _ := json.Marshal(alert)
		alerts = append(alerts, string(alertJSON))
	}

	return alerts, alertlist
}

// tests Handler when Create is set
func TestHandlerSuccess(t *testing.T) {
	setup()

	alerts, alertList := makeTestAlerts(5, "EXISTING-", true, false)
	output, err := handleRequest(Ctx, alerts)

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	have := string(output)

	want := fmt.Sprintf("Success: %d alerts processed. Updated alerts: %s", len(alerts), alertList)

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

func TestHandlerNoResults(t *testing.T) {
	setup()

	alerts, _ := makeTestAlerts(3, "EXISTING-", false, false)

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

func TestHandlerSameIds(t *testing.T) {
	setup()

	alerts, _ := makeTestAlerts(5, "EXISTING-", true, true)
	output, err := handleRequest(Ctx, alerts)

	var alertList []string
	alertList = append(alertList, "CREATED-1")

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	have := string(output)

	want := fmt.Sprintf("Success: 1 alerts processed. Created alerts: %s", alertList)

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
