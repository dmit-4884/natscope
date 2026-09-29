---
title: Streams
description: Create, inspect and edit JetStream streams from the Natscope sidebar.
---

# Streams

The sidebar lists every JetStream stream on the connected server. Selecting one opens a stream view with
five tabs: **Messages**, **Config**, **Consumers**, **Relations** and **Publish**.

You can create a stream, edit its config, purge it, seal it and delete it without dropping to the
`nats` CLI.

<Video src="/media/streams.mp4" poster="/media/streams.jpg" caption="Creating a stream and reviewing its config." />

## Find and arrange streams

The sidebar loads only stream names, so it stays quick with thousands of streams. It shows the first 12;
**Show more** reveals 50 more at a time. The same tools work in the **KV Stores** and **Object Store**
sections.

- **Filter** narrows the list to names that contain the text you type.
- The **star** next to a name pins it to the top of the section.
- **Drag** a name to reorder it, or focus it and press `Alt+↑` / `Alt+↓`. Reordering is off while a
  filter is active. **Reset to A–Z order** in the list menu drops the manual order and keeps your pins.

Pins and order are saved per connection in Natscope's own database, so they survive reloads and restarts.
Deleting the connection removes them.

## Create a stream

Click the **+** next to **Streams** in the sidebar, or open `/streams/new`. The **Create New Stream**
page offers two editing modes:

- **Form View** — every config field as a form control
- **JSON View** — paste or edit the raw stream config

The form covers the name and subjects, retention policy, storage backend, limits, replicas, discard
policy, compression and the advanced flags (deny delete/purge, allow direct, rollup, per-message TTL).
The JSON view accepts a full JetStream config document. Click **Create Stream** when you are done.

### Newer JetStream options

| Option                   | Needs      | Notes                                                                        |
|--------------------------|------------|------------------------------------------------------------------------------|
| Allow Per-Message TTL    | NATS 2.11+ | Can be switched on later, never off                                          |
| Delete Marker TTL        | NATS 2.11+ | Leaves a marker when Max Age removes a subject's last message; not on mirrors |
| Atomic Publish           | NATS 2.12+ | Batch publishes that commit together                                         |
| Counter Stream           | NATS 2.12+ | Set at creation only; every message carries `Nats-Incr`                      |
| Allow Message Schedules  | NATS 2.12+ | Can be switched on later, never off; no sources or mirror                    |
| Persist Mode: Async      | NATS 2.12+ | Set at creation only; file storage, one replica                             |
| Allow Fast Batch Publish | NATS 2.14+ | High-throughput batch publishing                                             |

Natscope reads the server's JetStream API level when it connects. An option the server cannot handle
stays visible but disabled, and its hint names the NATS version it needs. **Server information** in the
header lists what the connected server supports.

<Shot src="/media/stream-create.png" alt="Create stream" />

## Inspect and edit config

The **Config** tab shows the current stream configuration and lets you edit it in place. Saving opens a
diff first: **Confirm Stream Configuration Changes** lists exactly which fields change before anything
reaches the server.

<Shot src="/media/streams-info.png" alt="Stream config" />

## Relations

The **Relations** tab draws how messages move between streams, with the open stream highlighted. Upstreams sit
on the left and data flows to the right:

- **Source** (orange): a stream collects messages from one or more streams, optionally filtered or with subject
  transforms.
- **Mirror** (blue): a stream keeps an exact copy of another one, sequence numbers included.
- **Republish** (violet): a stream republishes stored messages to a subject. The link points at every stream whose
  subjects capture that subject, or at the subject itself when none does.

Each card shows the stream's messages, size, subjects, replicas, storage, retention and consumers. Click a card to
center the graph on that stream. KV and object store buckets appear as their `KV_` and `OBJ_` streams.

Click a link's label for its details: filter subjects and transforms, start sequence or time, the external API
for cross-account or cross-domain links, lag, when the upstream was last heard from, and the last error the
server reported. A red dashed link has an error; a dashed link has never reached its upstream. Drag the details
panel by its header to move it, double-click the header to dock it again, and press `Esc` to close it.

Some upstreams cannot be shown as streams:

- A dashed card with a globe is a stream in another account or JetStream domain.
- A red card is a source or mirror whose stream does not exist on this connection.

**Depth** limits how many links away from the open stream the graph reaches. Click a type in the legend to hide
its links. Drag or use the arrow keys to pan. Scroll or press `+` and `-` to zoom, and press `0` to fit the graph.

## Purge, seal, delete

The **Config** tab holds the destructive actions:

- **Purge** — drop messages, keep the stream. Purge everything, or scope it by subject, by sequence, or
  to keep the last N messages.
- **Seal** — make the stream read-only for good
- **Delete** — remove the stream

Each one asks for confirmation. Irreversible operations keep type-to-confirm even after you disable the
other prompts. See [Settings](/guide/settings).

## Copy as `nats` CLI

The **Config** tab has a **Copy as nats CLI** action that turns the current stream configuration into a
ready-to-paste `nats stream add` command. Use it to script the same stream elsewhere, or to file the
config in a repo.
