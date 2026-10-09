---
title: Messages
description: Browse, filter, search and export JetStream messages.
---

# Messages

The **Messages** tab is where you read a stream. It opens in **History** mode: a paginated list of stored
messages, newest first. Click a row and the side panel shows its NATS headers, timing and payload.

The same tab flips to **Realtime** for live tailing. See [Live tail](/guide/live-tail).

## Filter and search

Click **Filters** in the toolbar to open the filter panel:

- **Subject** — a NATS pattern. `*` matches one token, a trailing `>` matches one or more.
- **Payload Search** — text the decoded or raw payload must contain, ignoring case. Turn on **Regular expression** to
  match an RE2 expression as written; start it with `(?i)` to ignore case. A payload counts as decoded when a
  [Protobuf mapping](/guide/protobuf) matches its subject; one natscope only guesses the type of is searched as stored.
- **Header** — a header the message must carry, as `X-Trace` or `X-Trace=abc`. The name ignores case, a value must
  match exactly; for a header sent several times, any one of its values does, as do all of them as shown, joined by `, `.
- **Start Sequence** — begin the listing at a stream sequence.
- **Start Date & Time** — jump to the first message published at or after a timestamp. Quick chips:
  **Now**, **1h ago**, **24h ago**, **7d ago**.
- **Stop at Sequence** and **Stop at Date & Time** — where the search ends, in reading order.

**Apply** runs the filter, **Reset** clears it. Active filters show as chips above the list.

### Search the whole stream

Payload Search, Header and the stop points search the stream on the server instead of filtering the loaded page. The
server reads from the start point in the list's order: newest first by default, oldest first after a jump to a time.
It applies the subject filter itself and decodes Protobuf only when a payload does not match as stored.

A line above the list shows how many messages the search read and how many matched, with a **Stop** button. One search
reads at most 100,000 messages, runs at most 20 seconds and returns at most 500 matches, so a busy cluster never gets an
unbounded scan. When it reaches one of these limits, or you stop it, **Search further** continues from where it stopped
without skipping or repeating a match. The arrow keys
in the message details step through the matches.

Searching a work queue stream reads the messages one by one instead of through a consumer, which would remove them.

## Read a payload

The payload viewer has several views of the same bytes:

- **Decoded** when a [Protobuf mapping](/guide/protobuf) matches the subject, or
  [type detection](/guide/protobuf#type-detection) recognizes the payload
- **JSON** for a JSON payload
- **Raw** text
- **Hex**
- **Wire**, the Protobuf fields read without a schema

NATS headers sit alongside the payload.

## Paging and direction

The toolbar page-size dropdown offers 25, 50, 100, 250 and 500 messages per page. Paging works forward
and backward from any sequence. Set the defaults for both under **Settings → Preferences → Messages**.

## Compare two messages

**Diff** turns on compare mode. Select two messages and Natscope shows a field-level diff of their
payloads, so you can see what changed between two events on the same subject.

## Export

**Export** writes messages to **JSON**, **NDJSON** or **CSV**. The **full-range** scope walks the whole
stream server-side with complete payloads (no preview truncation), showing a progress bar you can
cancel.

## Act on a message

- **Edit & resend** loads a message back into the [publish form](/guide/publish) with its subject,
  payload and headers.
- **Delete** removes a message by sequence, with an optional secure erase.
- **Bookmark** saves a message with a note so you can find it again.
