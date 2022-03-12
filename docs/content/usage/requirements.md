---
title: "Requirements"
date: 2022-02-05T14:16:07+11:00
draft: false
---

You will need 3 things in place in order to use Squyre.

1. **You must have an AWS account to host it**. It runs solely in AWS using serverless services (lambdas and step functions). If you don't have one, don't be too concerned with signing up - if you're only running a few test alerts through Squyre [AWS "Free Tier"](https://aws.amazon.com/free/) should mean the cost is negligible (if not free).

2. **You need something that is generating security alerts for you**. Well, obviously! Currently, we support Splunk or Opsgenie as alert sources (and early, experimental Sumo Logic support). If you don't use either, but your platform supports sending alerts to AWS SNS or a Webhook, raise an issue and we can look at adding support - should be fairly easy.

3. **You need something capturing the steps taken to investigate alerts, like a ticketing system**. This is commonly a task management platform like Jira, ServiceNow etc. We support Jira or Opsgenie as output providers right now.