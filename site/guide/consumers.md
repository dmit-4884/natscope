---
title: Consumers
description: See every JetStream consumer and what holds it back, follow a message to its consumers, and create, edit, pause and resume consumers.
---

# Consumers

Natscope supports both pull and push consumers, durable and ephemeral.

## All consumers at once

**Consumers** in the sidebar lists every consumer on the connection, across all streams. Each row shows what the consumer
has not delivered yet (**Pending**), what waits for an ack out of the most it allows (**Waiting for ack**), how many
unacknowledged messages went out more than once (**Redelivered**) and when it last delivered. Stuck consumers come first,
then the ones with the most pending messages.

The **Status** column names the problem in plain words. Hover a badge for the details.

| Badge                 | Means                                                                                                   |
|-----------------------|---------------------------------------------------------------------------------------------------------|
| **Ack limit reached** | As many messages wait for an ack as max ack pending allows. The server delivers nothing new until clients ack or the ack wait runs out. |
| **No subscriber**     | A push consumer has messages to deliver and nobody subscribes to its deliver subject.                   |
| **Nobody pulling**    | A pull consumer has messages to deliver, no pull request waits, and no client pulled in the last minute. |
| **Redelivering**      | Some unacknowledged messages were delivered again: a client rejected them or missed the ack wait.        |
| **May lose messages** | The stream is at least 90% full and drops its oldest messages, while this consumer still has messages to get. |
| **Paused**            | Delivery is paused until the time shown.                                                                |

A consumer with nothing wrong is **Catching up** while it has work left, and **Caught up** when it has none.

The page reads the consumers every 5 seconds; turn **Auto-refresh** off to freeze the numbers. **Problems only** hides the
healthy consumers, the filter matches consumer names, streams and filter subjects, and the download button exports the
list as CSV or JSON. Click a row to open the consumer on its stream.

If the NATS user may not list the consumers of some streams, the page says which streams are missing and which
permission the server refused. If it may not list streams at all, the page names that permission instead.

## Where a consumer is

The view of a consumer starts with **What holds it back** when one of the problems above applies. **Where it is** shows:

- **Oldest waiting for ack**: the first delivered message no client acknowledged. The consumer cannot move its ack floor
  past it.
- **Next to deliver**: the message a client gets on its next pull or push.
- How far the consumer delivered and how far everything is acknowledged.

**Open** shows the message on the stream's **Messages** tab. Natscope reads these messages straight from the stream, so
looking at them does not deliver or acknowledge anything.

## What happened to a message

The details of a stream message have a **Consumers** line: how many consumers acknowledged it, got it and still owe an
ack, or have not received it yet. Expand it to see each consumer:

- **Acknowledged**: delivered and acknowledged
- **Waiting for ack**: delivered, not acknowledged yet. With explicit acks a client may have acknowledged it out of order,
  which the server does not report per message.
- **Not delivered yet**: still in line for this consumer
- **Skipped**: the consumer starts after this message, so it never gets it
- **Delivered**: delivered to a consumer that does not use acks

Consumers whose filter leaves the subject out are counted separately.

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
