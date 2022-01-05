package hellarad

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"testing"
)

type mockedSecretValue struct {
	secretsmanageriface.SecretsManagerAPI
	Resp secretsmanager.GetSecretValueOutput
}

func (m mockedSecretValue) GetSecretValue(*secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	// Return mocked response output
	return &m.Resp, nil
}

// tests GetSecret
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
		t.Fatalf("expected %s messages, got %s", expected, *value.SecretString)
	}
}
