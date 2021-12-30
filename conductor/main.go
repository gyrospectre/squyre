package main

import (
    "fmt"
    "encoding/json"
    "errors"
    "log"
    "regexp"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sfn"
    "github.com/aws/aws-sdk-go/service/cloudformation"
    "context"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-lambda-go/events"
    "github.com/gyrospectre/hellarad"
)

type AllPurposeAlert struct {
    Message		string `json:"message"`		// Splunk
    SearchName	string `json:"search_name"`
    Alert		string `json:"alert"`		// OpsGenie
}

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

func HandleRequest(ctx context.Context, snsEvent events.SNSEvent) (string, error) {
    var inputList []hellarad.Subject

    for _, record := range snsEvent.Records {
        snsRecord := record.SNS

        log.Printf("Processing message %s\n", snsRecord.MessageID)

        var messageObject AllPurposeAlert
        json.Unmarshal([]byte(snsRecord.Message), &messageObject)

        var details string

        if messageObject.SearchName != "" { 			// Source is Splunk
            log.Println("Got Splunk alert")
            details = messageObject.Message
        } else {
            log.Println("Got OpsGenie alert")
            details = messageObject.Alert
        }
        log.Printf("%s\n", details)

        re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
        submatchall := re.FindAllString(details, -1)
        for _, address := range submatchall {
            var subject = hellarad.Subject {
                IP: address,
            }
            inputList = append(inputList, subject)
        }
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