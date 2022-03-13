---
title: "Getting Started: Jira Setup"
date: 2022-02-05T14:16:07+11:00
draft: false
---

1. [Create a Jira API key](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/).

2. In AWS, [create a new Secrets Manager secret](https://docs.aws.amazon.com/secretsmanager/latest/userguide/manage_create-basic-secret.html) called `JiraApi` in the same account/region as Squyre is deployed. Use the following content, obviously substituting your key and email. The secret should be of type `Other type of secret`.
```
{
  "apikey": <the API key you just created>,
  "user": <the email address of the Jira account the key is associated with>
}
```
