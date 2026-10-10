---
title: Subscribe
description: Watch any NATS subject over core NATS, with wildcards, per-subject counters, search, mute, resend and reply.
---

# Subscribe

**Subscribe** in the sidebar listens to any subject over core NATS. It needs no stream: you see what
is published from the moment you subscribe, and nothing is stored. To read messages a stream already
holds, use [Messages](/guide/messages) or the [live tail](/guide/live-tail) of that stream instead.

## Start a subscription

1. Open **Subscribe** in the sidebar, or press `Cmd+K` and pick **Go to Subscribe**.
2. Type a subject and press `Enter` to add it. `orders.*` matches one token, `orders.>` everything
   below `orders`. Add as many subjects as you need; each one becomes a chip.
3. Click **Start**, or press `Cmd+Enter` (`Ctrl+Enter` on Windows and Linux).

`Backspace` in the empty field removes the last chip. Before you start, **Quick add** offers JetStream
advisories, all JetStream events, everything (`>`), and subjects you subscribed to before. The subjects
and recents are kept until you disconnect or switch to another connection.

NATS delivers every subject to `>`, but Natscope hides subjects that start with `$` (`$JS`, `$SYS`,
`$KV`, `$SRV`, …), `_INBOX` and the connection's own inbox prefix under a wildcard that doesn't name
them. Subscribe to them by name, for example `$JS.EVENT.>`, to see them.

## The feed

Messages show up newest first with their subject, time and a one-line preview of the payload: the
decoded JSON when a [subject mapping](/guide/protobuf) matches, the text otherwise. An arrow marks a
message that waits for a reply. Click a message to open it in the same viewer as stream messages; the
details close with the × in their corner.

A message that matches several of your subjects, such as `orders.created` under both `orders.>` and
`>`, shows up once and is counted once. A plain NATS client with overlapping subscriptions would get one
copy per subscription.

The line under the toolbar says how many messages arrived, how many the feed keeps, how many match the
search, and how many were skipped. Next to it, a counter per subject; click one to show only that
subject, or click its eye to **mute** it. Muted subjects stay out of the feed and the counters until you
unmute them, and they do not use up the display rate. Muting or unmuting restarts the subscription, and
messages published in that moment are missed, as core NATS keeps no copy.

- **Search** filters the received messages by subject and payload text.
- **Pause** freezes the feed; messages keep arriving in the background, and **Resume** shows how many
  wait. A very busy subject can outrun the pause buffer, which then keeps the latest ones.
- **Clear** empties the feed and the counters.
- **Keep** sets how many of the latest messages the feed holds, from 25 to 1,000.
- **Display rate** shows at most that many messages per second; the server skips the rest, does not
  queue them, and counts them as skipped. It applies to this subscription only, and changing it restarts
  the subscription.

**Stop** ends the subscription and keeps the messages on screen; **Start** listens again and adds to
them.

A dot next to **Subscribe** in the sidebar shows that it is listening (amber while it reconnects). The
subscription ends when you open another page, press **Stop**, disconnect, or switch to another
connection; the messages stay on screen, and **Start** picks up from there.

## Resend and reply

**Resend** in the viewer opens the message in a dialog where you can change the subject, payload and
headers, and publishes it over core NATS. A JSON or text payload starts out byte for byte as it arrived;
a Protobuf payload is shown as JSON and encoded again with its mapping. `Nats-*` headers, which the
server manages, are left out. The feed keeps the first 64 KB of a message; a bigger one cannot be resent
intact, so **Resend** is off for it.

A message published as a request carries a reply subject. **Reply** answers it: the dialog fills in
the reply subject, and you write the payload. Only the first reply counts, and the requester waits only
until its own timeout, so send the reply soon. Messages a JetStream push consumer delivers carry a
`$JS.ACK…` reply subject; answering it would acknowledge the message, so **Reply** is off for them.

## Permissions

If your NATS user may not subscribe to a subject, its chip shows a lock and **no permission**, and the
other subjects keep working. A subject you may subscribe to still arrives when a wildcard covering it,
such as `orders.>` over `orders.created`, is refused. When every subject is refused, the feed names the
missing permissions. Publishing from **Resend** or **Reply** to a subject you may not publish to names
the missing permission in the dialog.

When your permissions deny only part of a wildcard, for example `secret.>` under `>`, the server accepts
the subscription and silently leaves the denied subjects out. Nothing tells Natscope about it, so those
subjects don't show up.
