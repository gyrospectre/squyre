# Hella Rad!

Easy alert enrichment for overworked security teams!

![ooh so rad](https://media.giphy.com/media/l0MYylLtnC1ADCGys/giphy.gif)

Designed to be modular and extensible, it will consume your alerts, enrich them with information that helps you triage quicker, and then feed the juicy results back into your alert pipeline (or ticketing system).

The only pre-requisite is that you must have an AWS account to host HellaRad. Currently, we support Splunk or OpsGenie as alert sources, and Jira or OpsGenie as output providers.

## How can I use this?

As an example, let's say that your security team uses Splunk for alerting and investigation, and Atlassian Jira for ticketing. By using the SNS alert action in the free Splunk Add-on for AWS, you can set your alerts to send to Hella Rad, which will take the results you define as interesting, extract any public IP addresses from them, and then run them through a bunch of services to get information about then. Helle Rad will then create a Jira ticket for your alert, and add this information as comments.

Woot. Enjoy all that sweet, sweet extra time back in your day.

## Suggested Deployment Patterns
![ooh so rad](https://github.com/gyrpspectre/hellarad/blob/main/diagram.png)

## Getting Started - Splunk to Jira Deployment
This is the out of the box configuration, as it's the most generic. If you are using Splunk and Jira, but don't already have something in place to create tickets automatically when alerts fire, this is for you.

1. Clone this repo.
2. Update the consts at the top of `output/jira/main.go` with your destination Jira instance URL (`BaseURL`) and Project name (`Project`).
3. With appropriate AWS credentials in your terminal session, build and deploy the stack.
```
sam build
sam deploy --guided
```
4. Over on Splunk, install the Splunk Add-on for AWS (https://splunkbase.splunk.com/app/1876/), to give you an SNS alert action. 
5. Configure the app with some AWS creds. The IAM user or role must have SNS Publish/Get/List perms to SNS topic `hellarad-Alert`. See https://docs.splunk.com/Documentation/AddOns/released/AWS/Setuptheadd-on
https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/

6. Create a Jira API key (https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/)
7. In AWS, create a new Secrets Manager secret called `JiraApi` in the same account/region as HellaRad is deployed. Use the following content, obviously substituting your key and email.
```
{
  "apikey": <the API key you just created>,
  "user": <the email address of the Jira account the key is associated with>
}
```
8. Almost there! Update one of your Splunk saved searches, adding a `strcat` at the end to combine all the fields you think are of use to a new field called `interesting`.

`<awesome detection logic> | stats values(src_ip) as src_ip by dest_user | eval Detection="A test alert" | strcat src_ip "," dest_user interesting`

4. Add an 'AWS SNS Alert' action to your scheduled search (https://docs.splunk.com/Documentation/AddOns/released/AWS/ModularAlert), updating the 'Message' field of the action to `$result.interesting$`.
5. Also fill out the Account and Region fields per the doco for the AWS Tech Add-on. The topic should be set to `hellarad-Alert`.

Next time this alert fires, the details will be sent to HellaRad, which will create a Jira ticket for you, adding enrichment details for all extracted IP address to the same ticket as comments.

## Getting Started - OpsGenie Deployment
A more scalable pattern. If you are already using OpsGenie in your alert pipeline, you can just add HellaRad in. It doesn't matter what is creating the OpsGenie alerts in this case, and you can let OG take care of ticket creation, Slack messages etc as normal.

1. Clone this repo.
2. Edit `template.yaml` to use OpsGenie instead of Jira. In the `OutputFunction` definition, change the `CodeUri` value to `output/opsgenie`.
3. With appropriate AWS credentials in your terminal session, build and deploy the stack.
```
sam build
sam deploy --guided
```
4. Create an OpsGenie integration API key. See https://support.atlassian.com/opsgenie/docs/create-a-default-api-integration/
5. In AWS, create a new Secrets Manager secret called `OpsGenieAPI` in the same account/region as HellaRad is deployed. Use the following content, obviously substituting your key and email.
```
{
  "apikey": <the API key you just created>
}
```
6. Setup OpsGenie to send SNS messages to topic `hellarad-Alert` on alert creation only. See https://support.atlassian.com/opsgenie/docs/integrate-opsgenie-with-outgoing-amazon-sns/

## Enrichment Functions
It's easy to add enrichment functions, and more will be added over time. Feel free to PR and contribute!

Currently supports:
- Greynoise (https://www.greynoise.io/) : Tells security analysts what not to worry about. Indicator types: IP
- IP API (https://ip-api.com/) : IP address geolocation information. Indicator types: IP

## Testing

```
sam build
sam local invoke IPAPIFunction --event event/alert.json
sam local invoke GreynoiseFunction --event event/alert.json
sam local invoke ConductorFunction --event event/sns.json 
sam local invoke OutputFunction --event event/output.json 
```