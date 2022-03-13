---
title: "Getting Started: OpsGenie Setup"
date: 2022-02-05T14:16:07+11:00
draft: false
---

1. [Create an OpsGenie integration API key](https://support.atlassian.com/opsgenie/docs/create-a-default-api-integration/).

2. In AWS, [create a new Secrets Manager secret](https://docs.aws.amazon.com/secretsmanager/latest/userguide/manage_create-basic-secret.html) called `OpsGenieAPI` in the same account/region as Squyre is deployed. Use the following content, obviously substituting your key and email. The secret should be of type `Other type of secret`.
```
{
  "apikey": <the API key you just created>
}
```
3. [Setup OpsGenie to send SNS messages](https://support.atlassian.com/opsgenie/docs/integrate-opsgenie-with-outgoing-amazon-sns/) to topic `squyre-Alert` on alert creation only.
