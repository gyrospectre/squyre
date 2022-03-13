---
title: "Getting Started: Splunk Setup"
date: 2022-02-05T14:16:07+11:00
draft: false
---

1. On Splunk, install the [Splunk Add-on for AWS](https://splunkbase.splunk.com/app/1876/), which adds the ability to send alerts to SNS.

2. [Configure the app with some AWS credentials](https://docs.splunk.com/Documentation/AddOns/released/AWS/Setuptheadd-on). The IAM user or role must have SNS Publish/Get/List perms to SNS topic `squyre-Alert`.

3. Update one of your Splunk saved searches, adding a `strcat` at the end to combine all the fields you think are of use to a new field called `interesting`.

`<awesome detection logic> | stats values(src_ip) as src_ip by dest_user | eval Detection="A test alert" | strcat src_ip "," dest_user interesting`

4. [Add an `AWS SNS Alert` action to your scheduled search](https://docs.splunk.com/Documentation/AddOns/released/AWS/ModularAlert), updating the `Message` field of the action to `$result.interesting$`.

5. Also fill out the Account and Region fields per the AWS Tech Add-on documentation. The topic should be set to `squyre-Alert`.
