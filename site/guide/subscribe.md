---
title: Subscribe
description: Watch any NATS subject over core NATS, with wildcards, per-subject counters, search, resend and reply.
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

`Backspace` in the empty field removes the last chip. **Quick add** offers JetStream advisories, all
JetStream events, everything (`>`), and subjects you subscribed to before. Subjects are kept per
connection until you disconnect.

`>` does not deliver system subjects such as `$JS.…`, `$SYS.…` or `_INBOX.…`: subscribe to them by name.

## The feed

Messages show up newest first with their subject, time and size. Click one to open it in the same
viewer as stream messages, decoded from Protobuf when a [subject mapping](/guide/protobuf) matches.

- **Search** filters the received messages by subject and payload text.
- The counters under the toolbar show how many messages arrived on each subject; click one to show only
  that subject.
- **Pause** freezes the feed while messages keep arriving in the background; **Resume** catches up.
- **Clear** empties the feed and the counters.
- **Keep** sets how many messages stay in the feed, and **Display rate** caps how fast new ones are drawn
  on a busy subject.

**Stop** ends the subscription and keeps the messages on screen.

## Resend and reply

**Resend** in the viewer opens the message in a dialog where you can change the subject, payload and
headers, and publishes it over core NATS.

A message published as a request carries a reply subject. **Reply** answers it: the dialog fills in
the reply subject, and you write the payload. The requester waits only until its own timeout, so send
the reply soon.

## Permissions

If your NATS user may not subscribe to a subject, its chip turns red with **no permission**, and the
other subjects keep working. When every subject is refused, the feed names the missing permission.
Publishing from **Resend** or **Reply** to a subject you may not publish to shows the refusal in the
dialog.
