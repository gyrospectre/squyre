---
title: "Getting Started: Sumo Logic Setup"
date: 2022-02-05T14:16:07+11:00
draft: false
---

1. In Sumo, create a new Webhook connection under Manage Data > Monitoring > Connections. See [the official guide here](https://help.sumologic.com/Manage/Connections-and-Integrations/Webhook-Connections/Webhook_Connection_for_AWS_Lambda).

2. Use the following spec for the Payload. This matches the definition in Squyre, so that we can parse all the details correctly.

```
{
	"event_type": "trigger",
	"description": "{{Description}}",
	"client": "Sumo Logic",
	"client_url": "{{SearchQueryUrl}}",
	"name": "{{Name}}",
	"time_range": "{{TimeRange}}",
  	"time_trigger": "{{FireTime}}",
	"num_results": "{{NumQueryResults}}",
	"results": "{{ResultsJson}}",
  	"id": "{{Id}}"
}
```

3. Create a [new scheduled search](https://help.sumologic.com/Visualizations-and-Alerts/Alerts/Scheduled-Searches/Schedule_a_Search), and configure it to send notifications to the webhook you configured above.