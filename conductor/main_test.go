package main

import (
	"context"
	//	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/aws/aws-sdk-go/service/sfn/sfniface"
	"github.com/gyrospectre/hellarad"
	"testing"
)

var (
	Ctx            context.Context
	FunctionResult string
)

func setup() {
	// Reset SendAlert
	SendAlert = sendAlertToSfn
	// Reset FunctionResult
	FunctionResult = ""
}

type mockedStackValue struct {
	cloudformationiface.CloudFormationAPI
	Resp cloudformation.ListStackResourcesOutput
}

func (m mockedStackValue) ListStackResources(input *cloudformation.ListStackResourcesInput) (*cloudformation.ListStackResourcesOutput, error) {
	// Return mocked response output
	return &m.Resp, nil
}

type mockedSfnValue struct {
	sfniface.SFNAPI
	Resp sfn.DescribeExecutionOutput
}

func (m mockedSfnValue) DescribeExecution(input *sfn.DescribeExecutionInput) (*sfn.DescribeExecutionOutput, error) {
	sfnresp := sfn.DescribeExecutionOutput{
		Status: aws.String(FunctionResult),
	}

	// Return mocked response output
	return &sfnresp, nil
}

func (m mockedSfnValue) StartExecution(*sfn.StartExecutionInput) (*sfn.StartExecutionOutput, error) {
	sfnresp := sfn.StartExecutionOutput{
		ExecutionArn: aws.String("testExecArn"),
	}

	// Return mocked response output
	return &sfnresp, nil
}

func mockSendAlert(alert hellarad.Alert, sfnName string) error {
	return nil
}

func mockBuildDestination(arn string) StateMachine {
	sfnresp := sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("testExecArn"),
	}

	return StateMachine{
		Client:      mockedSfnValue{Resp: sfnresp},
		FunctionArn: "testArn",
	}

}

// tests main handler
func TestHandlerSuccess(t *testing.T) {
	setup()

	SendAlert = mockSendAlert

	event := events.SNSEvent{}
	event.Records = []events.SNSEventRecord{
		{
			SNS: events.SNSEntity{
				Message:   "{\"search_name\": \"Test Alert\", \"results_link\": \"http://127.0.0.1\", \"message\": \"hi 8.8.8.8 172.16.0.1\", \"correlation_id\": \"1234\"}",
				MessageID: "test-message-id",
			},
		},
	}
	have, _ := handleRequest(Ctx, event)

	want := "Processed 1 SNS messages."

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}

}

func TestSendAlertSuccess(t *testing.T) {
	setup()
	BuildDestination = mockBuildDestination

	resp := cloudformation.ListStackResourcesOutput{
		NextToken: aws.String(""),
		StackResourceSummaries: []*cloudformation.StackResourceSummary{
			{
				LogicalResourceId:  aws.String("testStepFunction"),
				PhysicalResourceId: aws.String("testStepFunctionArn"),
			},
		},
	}

	HellaRadStack = Stack{
		Client:    mockedStackValue{Resp: resp},
		StackName: "teststack",
	}

	alert := hellarad.Alert{
		RawMessage: "Testing",
	}
	FunctionResult = "SUCCEEDED"
	err := sendAlertToSfn(alert, "testStepFunction")

	if err != nil {
		fmt.Printf("Unexpected error %s", err)
	}
}

func TestSendAlertFailed(t *testing.T) {
	setup()
	BuildDestination = mockBuildDestination

	resp := cloudformation.ListStackResourcesOutput{
		NextToken: aws.String(""),
		StackResourceSummaries: []*cloudformation.StackResourceSummary{
			{
				LogicalResourceId:  aws.String("testStepFunction"),
				PhysicalResourceId: aws.String("testStepFunctionArn"),
			},
		},
	}

	HellaRadStack = Stack{
		Client:    mockedStackValue{Resp: resp},
		StackName: "teststack",
	}

	alert := hellarad.Alert{
		RawMessage: "Testing",
	}
	FunctionResult = "FAILED"
	err := sendAlertToSfn(alert, "testStepFunction")
	if err == nil {
		fmt.Print("Unexpected non error")
	}

	have := err.Error()
	want := "Step function execution failed"

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

func TestSendAlertTimedOut(t *testing.T) {
	setup()
	BuildDestination = mockBuildDestination

	resp := cloudformation.ListStackResourcesOutput{
		NextToken: aws.String(""),
		StackResourceSummaries: []*cloudformation.StackResourceSummary{
			{
				LogicalResourceId:  aws.String("testStepFunction"),
				PhysicalResourceId: aws.String("testStepFunctionArn"),
			},
		},
	}

	HellaRadStack = Stack{
		Client:    mockedStackValue{Resp: resp},
		StackName: "teststack",
	}

	alert := hellarad.Alert{
		RawMessage: "Testing",
	}
	FunctionResult = "TIMED_OUT"
	err := sendAlertToSfn(alert, "testStepFunction")
	if err == nil {
		fmt.Print("Unexpected non error")
	}

	have := err.Error()
	want := "Step function timed out"

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
