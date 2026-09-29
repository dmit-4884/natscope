---
title: Consumers
description: Create, edit, pause and resume JetStream consumers from the stream view.
---

# Consumers

The **Consumers** tab of a stream lists every consumer bound to it with its pending, ack and redelivery
counters. Selecting one opens the full config for editing.

Natscope supports both pull and push consumers, durable and ephemeral.

<Video src="/media/consumers.mp4" poster="/media/consumers.jpg" caption="Consumer list and detail view with live counters." />

## Create a consumer

1. Open a stream, go to the **Consumers** tab.
2. Click **Create New Consumer**.
3. Fill in the config and save.

The form covers the JetStream consumer surface:

- **Delivery policy** — all, last, new, by start sequence, by start time
- **Ack policy** — none, all, explicit
- **Filter subjects** — one or several
- **Backoff** — an array of redelivery delays
- **Max deliver** and **max ack pending**
- **Flow control** for push consumers
- **Deliver subject** and **deliver group** for push consumers
- **Replicas**

<Shot src="/media/consumers-detail.png" alt="Consumer detail" />

## Edit a consumer

Select a consumer, change the fields, and save. Natscope shows **Confirm Consumer Configuration
Changes** with a field-level diff before it sends anything to the server.

## Pause and resume

**Pause** stops delivery until a chosen time. Paused consumers carry a paused-until badge in the list.
**Resume** lifts it early. Needs NATS 2.11+.

## Reset

**Reset** clears the delivery state so unacknowledged messages are delivered again, without deleting the
consumer. Leave the sequence empty to keep the ack floor, or enter a stream sequence to replay from it.
Replaying from a sequence works for consumers that deliver all messages or start at a sequence or time.
Needs NATS 2.14+.

## Priority groups

Pull consumers can split work between clients through priority groups (NATS 2.11+). Set them in the
**Priority Groups** section of the form:

- **Pinned client** sends every message to one client until it stays idle longer than the pinned TTL
- **Overflow** serves a client only when the others fall behind
- **Prioritized** prefers clients with a lower priority number (NATS 2.12+)

The consumer view shows which client each group is pinned to. **Unpin** releases it, so the next pull
request gets pinned instead.

## Delete

Deleting a consumer asks for confirmation by default. Turn that prompt off under **Settings →
Preferences → Behavior**.

## Copy as `nats` CLI

**Copy as nats CLI** turns the selected consumer config into a `nats consumer add` command you can paste
into a terminal or commit to a repo.
