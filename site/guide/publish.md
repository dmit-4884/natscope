---
title: Publishing
description: Publish JSON or Protobuf messages with schema validation and custom NATS headers.
---

# Publishing

The **Publish** tab of a stream sends a message. Write raw JSON or text for plain subjects, or let
Natscope encode JSON into Protobuf binary when a [subject mapping](/guide/protobuf) resolves the message
type.

## Publish a message

1. Open a stream and go to the **Publish** tab.
2. Enter a **Subject Pattern**. When a mapping matches, the editor resolves the message type on its own.
3. Pick the encoding: automatic, JSON, or Protobuf.
4. Write the payload. With a resolved Protobuf type you get field autocompletion and live validation.
5. Add NATS headers if you need them.
6. Click **Publish Message**, or press `Cmd+Enter` (`Ctrl+Enter` on Windows and Linux).

## Schema validation

With a Protobuf type resolved, the editor validates as you type and shows **Schema valid**, **Schema
invalid** with a violation count, or **Validating…**. Fixing the violations before publishing beats
debugging a rejected message on the consumer side.

Natscope can also generate an example payload for any message type in the registry, which gives you a
correct skeleton to edit.

## Headers

Add any NATS headers as key/value pairs. They travel with the message and show up in the message viewer
alongside the payload.

## JetStream options

Below the headers, **JetStream options** set the headers for newer server features:

- **Message TTL** expires this one message, e.g. `30s` or `1h`. The stream needs per-message TTL (NATS 2.11+).
- **Counter increment** adds to the subject's running total on a counter stream (NATS 2.12+). The
  message goes out without a body, and the toast shows the new total.
- **Schedule this message** publishes to a target subject later: once at a time (NATS 2.12+), every
  interval, or on a cron expression with a time zone (NATS 2.14+). The stream needs message schedules
  on. Use one publish subject per schedule, such as `orders.schedule.42`.

An option the stream or server cannot take is disabled with the reason next to it.

## Publish timeout

**Settings → Preferences → Publish** sets the JetStream ack timeout in seconds.

## Templates and history

- Save the current draft with **Save as template**, or load a saved one from the **Templates** dropdown.
  See [Templates](/guide/templates).
- Every publish is logged. The **Publish History** panel sits on the right of this tab. See
  [Publish history](/guide/history).
- **Edit & resend** in the [message browser](/guide/messages) loads an existing message back into this
  form with its subject, payload and headers.
