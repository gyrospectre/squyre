# Hella Rad!

Easy alert enrichment for overworked security teams! Designed to be modular and extensible, it will take SNS alerts from whatever you use to detect threats. It will then pull out interesting indicators, use various services to gather infomation about them, and then feed the juicy results back into your alert pipeline (or ticketing system).

The only pre-requisite is that you must have an AWS account to host HellaRad. Currently, we support Splunk or OpsGenie as the source of the alert.

## How could I use this?

As an example, let's say you have an AWS account, and your security team uses Splunk for alerting and investigation. To setup:

1. Deploy HellaRad (see below)
2. Install the Splunk Add-on for AWS (https://splunkbase.splunk.com/app/1876/), to give you an SNS alert action. Configure the app with some AWS creds.
3. Update one of your saved searches, adding a `strcat` at the end to combine all the fields you think are of use to a new field called `interesting`.

`<awesome detection logic> | stats values(src_ip) as src_ip by dest_user | eval Detection="D002 - A test alert" | strcat src_ip "," dest_user interesting`

4. Add an 'AWS SNS Alert' action to your scheduled search, updating the 'Message' field of the action to `$result.interesting$`. Also fill out the Account and Region fields per the doco for the AWS Tech Add-on. The topic should be set to `hellarad-Alert`.

Next time this alert fires, the interesting fields will be sent to HellaRad, and the results sent to whatever you specify.

## Testing

```
sam build
sam local invoke GreynoiseFunction --event events/address.json 
sam local invoke IPAPIFunction --event events/address.json 
```