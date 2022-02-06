---
title: "State Machine"
date: 2022-02-05T14:16:07+11:00
draft: false
---

The main state machine is called `EnrichStateMachine`, because it, uh enriches, and it's a state machine.

It is defined by `statemachine/enrich.asl.json` in the [AWS ASL language](https://docs.aws.amazon.com/step-functions/latest/dg/concepts-amazon-states-language.html).

Layout is straightforward, nested parallel branches run the enrichment tasks which are sent to the output function at the end to update alerts/tickets.

<img src="/squyre/media/statemachine.png" alt="Enrich State Machine" width="75%" />
