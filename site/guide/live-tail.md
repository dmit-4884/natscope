---
title: Live tail
description: Watch messages arrive in real time, with Protobuf decoding and a display-rate throttle.
---

# Live tail

Live tail is a mode of the **Messages** tab, not a separate screen. The toolbar carries a two-way switch:
**History** shows stored messages, **Realtime** opens a live subscription and streams messages in as they
arrive.

Payloads decode on the fly. When you change a `.proto` source while a live session runs, Natscope
re-decodes with the new schema without a reconnect. See [Protobuf](/guide/protobuf).

<Video src="/media/live.mp4" poster="/media/live.jpg" caption="Realtime tail of a work queue, throttled to a readable rate." />

## Start a tail

1. Open a stream and go to the **Messages** tab.
2. Switch the toolbar toggle to **Realtime**.
3. Optional: open **Filters** and set a **Subject** pattern to narrow the feed. NATS wildcards work.

Messages arrive in batches with live throughput stats: messages per second, per-subject counts and
dropped-message tracking.

## Control the flow

Realtime mode adds three controls to the toolbar:

- **Pause** / **Resume** — stop and restart intake without dropping the subscription
- **Clear** — empty the live buffer
- **Display rate** — **No limit**, **1 msg/s**, **5 msg/s**, **10 msg/s**, **25 msg/s** or **50 msg/s**

The rate limit throttles what the UI renders. It does not slow the stream or the subscription. A busy
work queue at 1 msg/s stays readable while the server keeps running at full speed.

A separate dropdown caps how many live messages the buffer holds. Older ones fall off the top.

## Subscription mode

**Settings → Preferences → Live** picks how the tail subscribes:

- **Core NATS** — a plain subject subscription
- **JetStream Ordered** — an ordered ephemeral consumer bound to the stream

The same section sets **Max display rate** as a default, so new tails start throttled. Under **Display**,
**Default view mode** decides whether a stream opens in **History** or **Realtime**, and **Auto-scroll
live** keeps the list pinned to the newest message.
