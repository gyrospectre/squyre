package main

import (
	"fmt"
    "encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sfn"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/hellarad"
)

func getStackResourceArn(svc *cloudformation.CloudFormation, stackName string, resourceName string) (string, error) {
	req := cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackName),
	}
	for {
		resp, err := svc.ListStackResources(&req)
		if err != nil {
			return "", err
		}
		for _, s := range resp.StackResourceSummaries {
			if *s.LogicalResourceId == resourceName {
				return *s.PhysicalResourceId, nil
			}
		}
		req.NextToken = resp.NextToken
		if aws.StringValue(req.NextToken) == "" {
			break
		}
	}
	return "", errors.New("No matching stack resources found!")
}

func HandleRequest(ctx context.Context, subject hellarad.Subject) (string, error) {
	// Just for testing, generate some data
	var inputList = []hellarad.Subject { 
		hellarad.Subject {
			IP: "202.92.65.254",
		},
		hellarad.Subject {
			IP: "8.8.8.8",
		},
	}
	inputJson, _ := json.Marshal(inputList)

	sesh := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))

	cfnsvc := cloudformation.New(sesh)
	sfnArn, err := getStackResourceArn(cfnsvc, "hellarad", "IPLookupStateMachine")

	if err != nil {
		return sfnArn, err
	}

	sfnsvc := sfn.New(sesh)
	result, err := sfnsvc.StartExecution(&sfn.StartExecutionInput{
        StateMachineArn: 	&sfnArn,
		Input: aws.String(string(inputJson)),
    })

	if err != nil {
		fmt.Print(string(err.Error()))
		return string(err.Error()), err
	}
	return result.GoString(), nil
}

func main() {
	lambda.Start(HandleRequest)
}