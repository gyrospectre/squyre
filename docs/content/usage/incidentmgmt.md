---
title: "Getting Started: Opsgenie Deployment"
date: 2022-02-05T14:16:07+11:00
draft: false
---

1. Clone the repo.
```
git clone https://github.com/gyrospectre/squyre.git
```
2. [Install the AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html).
3. Edit `template.yaml` to use OpsGenie instead of Jira. In the `OutputFunction` definition, change the `CodeUri` value to `output/opsgenie`.
4. While you're at it, add a second snippet to `template.yaml`, to allow Opsgenie to send to the SNS topic.
```
  AlertTopicPolicy:
    Type: AWS::SNS::TopicPolicy
    Properties:
      PolicyDocument:
        Id: AlertTopicPolicy
        Version: 2012-10-17
        Statement:
          - Sid: OpsGenie-Publish
            Effect: Allow
            Principal:
              AWS: arn:aws:iam::089311581210:root
            Action: sns:Publish
            Resource: "*"
      Topics:
        - !Ref AlertTopic
```
4. With appropriate AWS credentials in your terminal session, build and deploy the stack. Name the stack `sqyre`.
```
sam build
sam deploy --guided
```
5. [Create an OpsGenie integration API key](https://support.atlassian.com/opsgenie/docs/create-a-default-api-integration/).
6. In AWS, [create a new Secrets Manager secret](https://docs.aws.amazon.com/secretsmanager/latest/userguide/manage_create-basic-secret.html) called `OpsGenieAPI` in the same account/region as Squyre is deployed. Use the following content, obviously substituting your key and email. The secret should be of type `Other type of secret`.
```
{
  "apikey": <the API key you just created>
}
```
7. [Setup OpsGenie to send SNS messages](https://support.atlassian.com/opsgenie/docs/integrate-opsgenie-with-outgoing-amazon-sns/) to topic `squyre-Alert` on alert creation only.

Next time an alert fires, the details will be sent to Squyre, which will add enrichment details back into the Opsgenie alert in the form of notes.
