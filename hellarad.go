package hellarad

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type Subject struct {
	Domain	string	`json:"domain"`
	IP 		string 	`json:"address"`
}

type Result struct {
	Source			string
	AttributeValue	string	
	Message			string
	Success			bool
}

type Alert struct {
    Details     string
    Id          string
}

func (r *Result) Prettify() string {
	var message string

	if r.Success == true {
		prettymsg, err := json.MarshalIndent(r.Message, "", "    ")
		message = fmt.Sprintf("Details on %s from %s:\n%s", r.AttributeValue, r.Source, r.prettymsg)
	} else {
		message = fmt.Sprintf("Failed to get info from %s! Error: %s", r.Source, r.Message)
	}

    return message
}

func GetSecret(location string) (secretsmanager.GetSecretValueOutput, error){
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