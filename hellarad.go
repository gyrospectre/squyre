package hellarad

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type Subject struct {
	Domain string `json:"domain"`
	IP     string `json:"address"`
}

type Result struct {
	Source         string
	AttributeValue string
	Message        string
	Success        bool
}

type Alert struct {
	Timestamp  string
	Name       string
	RawMessage string
	Url        string
	Id         string
	Subjects   []Subject
	Results    []Result
}

type Alerter interface {
	Normaliser() Alert
}

type SplunkAlert struct {
	Message       string `json:"message"`
	CorrelationId string `json:"correlation_id"`
	SearchName    string `json:"search_name"`
	Timestamp     string `json:"timestamp"`
	Entity        string `json:"entity"`
	Source        string `json:"source"`
	Event         string `json:"event"`
	ResultsLink   string `json:"results_link"`
	App           string `json:"app"`
	Owner         string `json:"owner"`
}

func (alert SplunkAlert) Normaliser() Alert {
	return Alert{
		RawMessage: alert.Message,
		Id:         alert.CorrelationId,
		Name:       alert.SearchName,
		Url:        alert.ResultsLink,
		Timestamp:  alert.Timestamp,
	}
}

// See https://support.atlassian.com/opsgenie/docs/opsgenie-edge-connector-alert-action-data/
type OpsGenieAlert struct {
	Action          string `json:"action"`
	IntegrationId   string `json:"integrationId"`
	IntegrationName string `json:"integrationName"`
	Source          struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"source"`
	Alert struct {
		AlertId   string `json:"alertId"`
		Message   string `json:"message"`
		CreatedAt string `json:"createdAt"`
	} `json:"alert"`
}

func (alert OpsGenieAlert) Normaliser() Alert {
	return Alert{
		RawMessage: alert.Alert.Message,
		Id:         alert.Alert.AlertId,
		Name:       alert.Alert.AlertId,
		Timestamp:  alert.Alert.CreatedAt,
	}
}

func GetSecret(location string) (secretsmanager.GetSecretValueOutput, error) {
	svc := secretsmanager.New(session.New())
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(location),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		return *result, err
	}

	return *result, nil
}
