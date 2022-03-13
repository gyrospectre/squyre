---
title: "Getting started"
date: 2022-02-05T14:16:07+11:00
draft: false
---

<img src="/squyre/media/deploypatterns.png" alt="patterns" width="550" align="right" />

There are a couple of ways you can deploy, either directly between your alert source and ticketing system (pattern 1), or using an incident management platform like Opsgenie (pattern 2).

Pattern 1 is the out of the box configuration as it's the most generic. If you don't already have something in place to create tickets automatically when alerts fire, then this is for you. We currently support Splunk and Sumo Logic for alert sources. Jira is the only supported ticket management system right now.

Pattern 2 however, is more scalable. Using an incident management platform allows you to add as many alert sources as you like, without having to change anything on the Squyre side. We only support Ogsgenie today, with PagerDuty likely to come next.

Either way, you start the same way to deploy Squyre to AWS! It's pretty easy.


1. Clone the repo.
```
git clone https://github.com/gyrospectre/squyre.git
```
2. [Install the AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html).

3. Run the setup wizard. You'll be asked to specify your alert source and output platforms, and which enrichment functions to use. Hint: Choose only functions that don't require API keys to get started quicker in your just want to play around!

```
make setup
```

- Note: If you choose Jira for your output platform, you'll need to enter the Project name to create tickets in, and the base URL of your Jira Cloud instance.

4. In AWS, create an IAM user to use for deployment. Whilst you can definitely cut down things further, a user with the `IAMFullAccess` and `PowerUserAccess` managed policies will work fine. You don't need console access here, just choose `Access key - Programmatic access`.

5. Pop the credentials of this new deployment user into your shell. See [this guide](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html#envvars-set) if you need help.

6. Build and deploy the stack. Just use the defaults when prompted, to deploy a stack named `squyre`.

```
make build
make deploy-guided
```

7. Depending on what options you chose in step 3, see the child pages of the Functions and Getting Started sections of this documentation for specific setup requirements for each.