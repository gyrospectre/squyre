package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/gyrospectre/squyre/pkg/squyre"
	"sort"
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

func makeTestAlerts(number int, groups int, prefix string, includeResults bool, sameID bool) ([][]string, []string) {
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

	var grouplist [][]string
	var alerts []string
	var alertlist []string
	for j := 0; j < groups; j++ {
		for i := 1; i <= number; i++ {
			if sameID {
				alert.ID = fmt.Sprintf("%s1", prefix)
			} else {
				alert.ID = fmt.Sprintf("%s%d", prefix, i+(j*number))
			}

			alertlist = append(alertlist, alert.ID)
			alertJSON, _ := json.Marshal(alert)
			alerts = append(alerts, string(alertJSON))
		}
		grouplist = append(grouplist, alerts)
	}
	return grouplist, alertlist
}

// tests Handler when Create is set
func TestHandlerSuccess(t *testing.T) {
	setup()

	numgroups := 2
	numalerts := 5

	alerts, alertList := makeTestAlerts(numalerts, numgroups, "EXISTING-", true, false)

	output, err := handleRequest(Ctx, alerts)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	sort.Strings(alertList)

	have := string(output)
	want := fmt.Sprintf("Success: %d alerts processed (%d groups). Updated alerts: %s", numalerts*numgroups, numgroups, alertList)

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

func TestHandlerNoResults(t *testing.T) {
	setup()

	numgroups := 1
	numalerts := 3

	alerts, _ := makeTestAlerts(numalerts, numgroups, "EXISTING-", false, false)

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

	numgroups := 1
	numalerts := 3

	alerts, _ := makeTestAlerts(numalerts, numgroups, "EXISTING-", true, true)

	output, err := handleRequest(Ctx, alerts)

	var alertList []string
	alertList = append(alertList, "EXISTING-1")

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	have := string(output)

	want := fmt.Sprintf("Success: 1 alerts processed (%d groups). Updated alerts: %s", numgroups, alertList)

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
