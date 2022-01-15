package squyre

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

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
