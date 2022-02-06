---
title: "Getting started"
date: 2022-02-05T14:16:07+11:00
draft: false
---

There are a couple of ways you can deploy, either directly between your alert source and ticketing system (pattern 1), or using an incident management platform like Opsgenie (pattern 2).

<img src="/squyre/media/deploypatterns.png" alt="ooh so rad" width="700"/>

Pattern 1 is the out of the box configuration as it's the most generic. If you are using Splunk and Jira, but don't already have something in place to create tickets automatically when alerts fire, then this is for you.

Pattern 2 however, is more scalable. If you are already using Opsgenie in your alert pipeline, this is a better option. This allows you to add as many alert sources as you like, without having to change anything on the Squyre side.
