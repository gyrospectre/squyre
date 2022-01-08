package squyre

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

// Subject defines attributes about a thing that we want to know about
type Subject struct {
	Domain string `json:"domain"`
	IP     string `json:"address"`
}

// Result holds enrichment results, and where they came from
type Result struct {
	Source         string
	AttributeValue string
	Message        string
	Success        bool
}

// Alert holds information about an incoming alert
type Alert struct {
	Timestamp  string
	Name       string
	RawMessage string
	URL        string
	ID         string
	Subjects   []Subject
	Results    []Result
}

// Alerter defines common functions for all alert types
type Alerter interface {
	Normaliser() Alert
}

// SplunkAlert defines the standard format alerts come to us from Splunk
type SplunkAlert struct {
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
	SearchName    string `json:"search_name"`
	Timestamp     string `json:"timestamp"`
	Entity        string `json:"entity"`
	Source        string `json:"source"`
	Event         string `json:"event"`
	ResultsLink   string `json:"results_link"`
	App           string `json:"app"`
	Owner         string `json:"owner"`
}

// Normaliser comverts a Splunk alert to our standard form
func (alert SplunkAlert) Normaliser() Alert {
	return Alert{
		RawMessage: alert.Message,
		ID:         alert.CorrelationID,
		Name:       alert.SearchName,
		URL:        alert.ResultsLink,
		Timestamp:  alert.Timestamp,
	}
}

// OpsGenieAlert defines the standard format alerts come to us from OpsGenie
// See https://support.atlassian.com/opsgenie/docs/opsgenie-edge-connector-alert-action-data/
type OpsGenieAlert struct {
	Action          string `json:"action"`
	IntegrationID   string `json:"integrationId"`
	IntegrationName string `json:"integrationName"`
	Source          struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"source"`
	Alert struct {
		AlertID   string `json:"alertId"`
		Message   string `json:"message"`
		CreatedAt string `json:"createdAt"`
	} `json:"alert"`
}

// Normaliser comverts an OpsGenie alert to our standard form
func (alert OpsGenieAlert) Normaliser() Alert {
	return Alert{
		RawMessage: alert.Alert.Message,
		ID:         alert.Alert.AlertID,
		Name:       alert.Alert.AlertID,
		Timestamp:  alert.Alert.CreatedAt,
	}
}

// Secret abstracts AWS Secrets Manager secrets
type Secret struct {
	Client   secretsmanageriface.SecretsManagerAPI
	SecretID string
}

func (s *Secret) getValue() (*secretsmanager.GetSecretValueOutput, error) {

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(s.SecretID),
	}
	output, err := s.Client.GetSecretValue(input)

	return output, err
}

// GetSecret fetches a secret value from AWS Secrets Manager given a secret location
func GetSecret(location string) (secretsmanager.GetSecretValueOutput, error) {
	sess := session.Must(session.NewSession())

	s := Secret{
		Client:   secretsmanager.New(sess),
		SecretID: location,
	}
	output, err := s.getValue()

	return *output, err
}
