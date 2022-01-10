package squyre

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/google/go-cmp/cmp"
)

type mockedSecretValue struct {
	secretsmanageriface.SecretsManagerAPI
	Resp secretsmanager.GetSecretValueOutput
}

func (m mockedSecretValue) GetSecretValue(*secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	// Return mocked response output
	return &m.Resp, nil
}

// tests GetSecret return an expected value
func TestGetSecret(t *testing.T) {
	expected := "ooo so secret1!"

	resp := secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(expected),
	}

	s := &Secret{
		Client:   mockedSecretValue{Resp: resp},
		SecretID: "testsecret",
	}

	value, err := s.getValue()

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	if *value.SecretString != expected {
		t.Fatalf("expected value %s, got %s", expected, *value.SecretString)
	}
}

func TestNormaliseSplunkAlert(t *testing.T) {
	splunk := SplunkAlert{
		Message:       "Testing",
		CorrelationID: "1234-1234",
		SearchName:    "Test Search",
		ResultsLink:   "https://127.0.0.1/test.html",
		Timestamp:     "2022-12-12 18:00:00",
	}

	expected, _ := json.Marshal(Alert{
		RawMessage: "Testing",
		ID:         "1234-1234",
		Name:       "Test Search",
		URL:        "https://127.0.0.1/test.html",
		Timestamp:  "2022-12-12 18:00:00",
	})
	output, _ := json.Marshal(splunk.Normaliser())
	if !cmp.Equal(expected, output) {
		t.Fatalf("expected value %s, got %s", expected, output)
	}
}

func TestNormaliseOGAlert(t *testing.T) {
	opsgenie := OpsGenieAlert{}

	opsgenie.Alert.AlertID = "1234-1234"
	opsgenie.Alert.Message = "Testing"
	opsgenie.Alert.CreatedAt = "2022-12-12 18:00:00"
	opsgenie.Details.ResultsObject = "{ 8.8.8.8 }"

	expected, _ := json.Marshal(Alert{
		RawMessage: "{ 8.8.8.8 }",
		ID:         "1234-1234",
		Name:       "Testing",
		Timestamp:  "2022-12-12 18:00:00",
	})
	output, _ := json.Marshal(opsgenie.Normaliser())
	if !cmp.Equal(expected, output) {
		t.Fatalf("expected value %s, got %s", expected, output)
	}
}
