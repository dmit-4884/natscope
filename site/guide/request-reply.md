---
title: Request / Reply
description: Send a core NATS request to a service subject and inspect the first reply, decoded from Protobuf when needed.
---

# Request / Reply

**Request / Reply** in the sidebar sends a core NATS request: the message goes to a subject with a
private reply inbox, and the first subscriber that answers wins. Use it to call a service, check that
it is up, or look at what it returns. Nothing here goes through JetStream, and requests are not
recorded in the [publish history](/guide/history).

## Send a request

1. Open **Request / Reply** in the sidebar, or press `Cmd+K` and pick **Go to Request / Reply**.
2. Enter a literal **Subject**, such as `orders.validate` or `$SRV.PING`. Wildcards are not allowed.
   Recent subjects and the literal patterns of your mappings show up as suggestions.
3. Pick a **timeout**: how long to wait for the first reply, from 500 ms to 60 s.
4. Write the payload, or leave it empty for a request without a body. Text that is not JSON is sent as
   plain text.
5. Add NATS headers if the service expects them.
6. Click **Send Request**, or press `Cmd+Enter` (`Ctrl+Enter` on Windows and Linux).

Dynamic values such as `{{uuid}}` (listed under **Dynamic value helpers** in the editor toolbar) work in
the subject, payload and header values. The last subject, payload, headers and timeout are kept as a draft
until you disconnect.

## The reply

The reply panel shows the round-trip time, the payload size, the inbox the reply arrived on, the
payload as JSON, raw text or a hex dump, and the reply headers.

A reply that carries the `Nats-Service-Error` header, as services built on the NATS micro framework
send, is flagged as a service error with its `Nats-Service-Error-Code`.

When no reply arrives, the panel tells you why:

- **No responders**: nothing is subscribed to the subject. NATS reports it at once instead of letting the
  request wait for the timeout.
- **No reply within the timeout**: something is subscribed but did not answer in time.
- A permissions error when your user may not publish to the subject or subscribe to its reply inbox.

## Protobuf

When a [subject mapping](/guide/protobuf) matches the request subject, the JSON payload is encoded to
that Protobuf type before it is sent, with the mapping's framing and pinned schema, just like in the
Publish tab. The editor completes fields at any depth and enum values, and offers an example message.

Replies arrive on an inbox subject, so no mapping applies to them. Pick the reply type in **Decode as**.
If the request type ends in `Request` and the same source has a matching `Response` or `Reply` type,
Natscope selects it for you. The choice is remembered per subject.

## Templates

**Templates** and **Save as template** work as in the Publish tab. A template whose subject has
wildcards is loaded with its saved wildcard values filled in.
