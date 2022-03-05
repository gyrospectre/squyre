---
title: "Getting Started: Splunk to Jira Deployment"
date: 2022-02-05T14:16:07+11:00
draft: false
---

1. Clone the repo.
```
git clone https://github.com/gyrospectre/squyre.git
```
2. [Install the AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html).
3. Update the consts at the top of `output/jira/main.go` with your destination Jira instance URL (`BaseURL`) and Project name (`Project`).
4. *!Optional!* Customise which functions should be run for your environment. See [Customising]({{< ref "/usage/customise" >}} "Customising").
5. With appropriate AWS credentials in your terminal session, build and deploy the stack. Name the stack `squyre`.
```
sam build
sam deploy --guided
```
6. Over on Splunk, install the [Splunk Add-on for AWS](https://splunkbase.splunk.com/app/1876/), which adds the ability to send alerts to SNS.
7. [Configure the app with some AWS credentials](https://docs.splunk.com/Documentation/AddOns/released/AWS/Setuptheadd-on). The IAM user or role must have SNS Publish/Get/List perms to SNS topic `squyre-Alert`.

8. [Create a Jira API key](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/).
9. In AWS, [create a new Secrets Manager secret](https://docs.aws.amazon.com/secretsmanager/latest/userguide/manage_create-basic-secret.html) called `JiraApi` in the same account/region as Squyre is deployed. Use the following content, obviously substituting your key and email. The secret should be of type `Other type of secret`.
```
{
  "apikey": <the API key you just created>,
  "user": <the email address of the Jira account the key is associated with>
}
```
10. Almost there! Update one of your Splunk saved searches, adding a `strcat` at the end to combine all the fields you think are of use to a new field called `interesting`.

`<awesome detection logic> | stats values(src_ip) as src_ip by dest_user | eval Detection="A test alert" | strcat src_ip "," dest_user interesting`

11. [Add an `AWS SNS Alert` action to your scheduled search](https://docs.splunk.com/Documentation/AddOns/released/AWS/ModularAlert), updating the `Message` field of the action to `$result.interesting$`.
12. Also fill out the Account and Region fields per the AWS Tech Add-on documentation. The topic should be set to `squyre-Alert`.

Next time this alert fires, the details will be sent to Squyre, which will create a Jira ticket for you, adding enrichment details in the form of comments.
