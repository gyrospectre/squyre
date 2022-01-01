package hellarad

import (
	"encoding/json"
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
	Details string
	Id      string
	Subjects []Subject
	Results  []Result
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
