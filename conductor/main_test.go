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
	"github.com/fatih/structs"
	"github.com/gyrospectre/squyre/pkg/squyre"
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

func mockSendAlert(alert squyre.Alert, sfnName string) error {
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
			EventSource: "aws:sns",
		},
	}
	have, _ := handleRequest(Ctx, structs.Map(event))

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

	Stack = CloudformationStack{
		Client:    mockedStackValue{Resp: resp},
		StackName: "teststack",
	}

	alert := squyre.Alert{
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

	Stack = CloudformationStack{
		Client:    mockedStackValue{Resp: resp},
		StackName: "teststack",
	}

	alert := squyre.Alert{
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

	Stack = CloudformationStack{
		Client:    mockedStackValue{Resp: resp},
		StackName: "teststack",
	}

	alert := squyre.Alert{
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

func TestIPExtraction(t *testing.T) {
	setup()
	ua1 := "Mozilla/5.0 (X11; U; Linux i686; en-US; rv:1.8.1.3) Gecko/20070517 BonEcho/2.0.0.3"
	ua2 := "Mozilla/5.0 (X11; ; Linux i686; rv:1.9.2.20) Gecko/20110805"
	ip1 := "8.8.8.8"
	ip2 := "151.101.29.67"
	ip3 := "192.168.1.1"
	ip4 := "202.92.65.254"
	ip5 := "3.3.3.3"

	message := ip1 + " " + ua1 + " " + " [" + ip4 + ", " + ip5 + "] " + ip3 + " ip=" + ip2 + " " + ua2 + " " + ip2 + "}"
	subjects := extractIPs(message)

	have := len(subjects)

	want := 4
	if have != want {
		t.Fatalf("Unexpected number of IPs. \nHave: %x\nWant: %x", have, want)
	}

	if subjects[0].Value != ip1 {
		t.Fatalf("Unxpected first IP. \nHave: %s\nWant: %s", subjects[0].Value, ip1)
	}

	if subjects[1].Value != ip4 {
		t.Fatalf("Unxpected second IP. \nHave: %s\nWant: %s", subjects[1].Value, ip4)
	}

	if subjects[2].Value != ip5 {
		t.Fatalf("Unxpected third IP. \nHave: %s\nWant: %s", subjects[1].Value, ip5)
	}

	if subjects[3].Value != ip2 {
		t.Fatalf("Unxpected forth IP. \nHave: %s\nWant: %s", subjects[1].Value, ip2)
	}
}

func TestHostExtraction(t *testing.T) {
	setup()
	host1 := "ABC-12345"
	host2 := "X123487822"
	host3 := "ABC-54321"

	message := host1 + " " + " [" + host2 + ", " + host3 + "] " + host2 + "}"
	HostRegex = `ABC-\d{5}`
	subjects := extractHosts(message)

	have := len(subjects)

	want := 2
	if have != want {
		t.Fatalf("Unexpected number of Hosts. \nHave: %x\nWant: %x", have, want)
	}

	if subjects[0].Value != host1 {
		t.Fatalf("Unxpected first host. \nHave: %s\nWant: %s", subjects[0].Value, host1)
	}

	if subjects[1].Value != host3 {
		t.Fatalf("Unxpected second host. \nHave: %s\nWant: %s", subjects[1].Value, host3)
	}
}

func TestNoHostRegex(t *testing.T) {
	setup()

	subjects := extractHosts("}")

	have := len(subjects)
	want := 0

	if have != want {
		t.Fatalf("Unexpected behaviour when host regex missing.\n Got: %s", subjects)
	}
}

func TestNoIgnoreDomain(t *testing.T) {
	setup()

	subjects := extractDomains("google.com internal.domain")

	have := len(subjects)
	want := 1

	if have != want {
		t.Fatalf("Unexpected behaviour when ignore domain missing.\n Got: %s", subjects)
	}
}

func TestUrlExtraction(t *testing.T) {
	setup()
	url1 := "http://google.com/test/awesome?test"
	url2 := "https://github.com/gyrospectre/squyre"
	url3 := "https://apc04.safelinks.protection.outlook.com/?url=https%3A%2F%2Fdocs.testsite.int%2Ffile%2Fim0w22da6434202ce486e98ae85196b5ccc76&data=02%7C01%7Cwoot.woot%40test.com%7C2990160b578248181f4008d79461f071%7C4f4f4c56a772461a967e7890c3960b3a%7C1%7C1%7C637141020687342499&sdata=MNYejoOQbAVPTD1ijNbwMIfl8LV8E4JlP396Pm4470E%3D&reserved=0"

	str1 := "ABC 12345"
	str2 := "ABC " + url1
	str3 := url2
	str4 := url3

	message := str1 + " " + " [" + str2 + ", " + str3 + "] " + str2 + "} " + str4
	subjects := extractUrls(message)

	have := len(subjects)

	want := 3
	if have != want {
		t.Fatalf("Unexpected number of Urls. \nHave: %x\nWant: %x", have, want)
	}

	if subjects[0].Value != url1 {
		t.Fatalf("Unxpected first Url. \nHave: %s\nWant: %s", subjects[0].Value, url1)
	}

	if subjects[1].Value != url2 {
		t.Fatalf("Unxpected second Url. \nHave: %s\nWant: %s", subjects[1].Value, url2)
	}

	wantUrl := "https://docs.testsite.int/file/im0w22da6434202ce486e98ae85196b5ccc76"
	if subjects[2].Value != wantUrl {
		t.Fatalf("Unxpected third Url. \nHave: %s\nWant: %s", subjects[2].Value, wantUrl)
	}
}

func TestMalformedATPUrl(t *testing.T) {
	setup()
	url1 := "https://apc04.safelinks.protection.outlook.com/?rl=https%3A%2F%2Fdocs.testsite.int%2Ffile%2Fim0w22da6434202ce486e98ae85196b5ccc76"
	url2 := "https://apc04.safelinks.protection.outlook.com/?url=https%3A%2F%2Fdocs.testsite.int%2Ffile%2Fim0w22da6434202ce486e98ae85196b5ccc76data=02%7C01%7Cwoot.woot%40test.com%7C2990160b578248181f4008d79461f071%7C4f4f4c56a772461a967e7890c3960b3a%7C1%7C1%7C637141020687342499sdata=MNYejoOQbAVPTD1ijNbwMIfl8LV8E4JlP396Pm4470E%3Dreserved=0"
	url3 := "https://apc04.safelinks.protection.outlook.com/?url=https%3A%2F%2Fdocs.testsite.int%2Ffile%2Fim0w22da6434202ce486e98ae85196b5ccc76&data=02%7C01%7Cwoot.woot%40test.com%7C2990160b578248181f4008d79461f071%7C4f4f4c56a772461a967e7890c3960b3a%7C1%7C1%7C637141020687342499&sdata=MNYejoOQbAVPTD1ijNbwMIfl8LV8E4JlP396Pm4470E%3D&reserved=0"

	message := "ABC " + url1 + ": " + url2 + " { " + url3

	subjects := extractUrls(message)

	have := len(subjects)
	want := 3

	if have != want {
		t.Fatalf("Unexpected number of Urls. \nHave: %x\nWant: %x", have, want)
	}

	if subjects[0].Value != url1 {
		t.Fatalf("Unxpected first Url. \nHave: %s\nWant: %s", subjects[0].Value, url1)
	}

	if subjects[1].Value != url2 {
		t.Fatalf("Unxpected second Url. \nHave: %s\nWant: %s", subjects[1].Value, url2)
	}

	wantUrl := "https://docs.testsite.int/file/im0w22da6434202ce486e98ae85196b5ccc76"
	if subjects[2].Value != wantUrl {
		t.Fatalf("Unxpected third Url. \nHave: %s\nWant: %s", subjects[2].Value, wantUrl)
	}
}
