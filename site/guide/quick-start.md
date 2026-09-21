---
title: Quick start
description: Add a NATS connection, pick a stream, and browse its messages.
---

# Quick start

Three steps from a running binary to reading messages. This assumes Natscope is
[installed](/guide/installation) and serving on <http://127.0.0.1:4280>.

## 1. Add a connection

The first screen lists **Saved Connections**. Click **New Connection**, then:

1. Give it a **Name**.
2. Enter one or more URLs under **Servers**. Several URLs give you cluster failover. Schemes:
   `nats://`, `tls://`, `ws://`, `wss://`.
3. Pick an auth method: **None**, **User/Pass**, **Token**, **NKey** or **Credentials**.
4. Click **Test**. The result reports round-trip time, server version and whether JetStream is
   available.
5. Click **Connect**.

Credentials go to your OS keychain or an encrypted file vault, never into the database. See
[Secrets](/reference/secrets).

<Video src="/media/connect.mp4" poster="/media/connect.jpg" caption="Creating and testing a cluster connection." />

## 2. Pick a stream

The sidebar lists **Streams**, **KV Stores** and **Object Store** for the connected server. Click a
stream to open it. Each stream has four tabs: **Messages**, **Config**, **Consumers** and **Publish**.

Press `Cmd+K` (`Ctrl+K` on Windows and Linux) to open the command palette and jump to any stream or
bucket by name.

## 3. Browse messages

The **Messages** tab opens in **History** mode with the most recent messages. Click a row to open it in
the side panel: NATS headers, timing, and the payload as decoded JSON, raw text, hex or base64.

Use **Filters** to narrow by subject (NATS wildcards `*` and `>` work), search the payload text, or jump
to a sequence or timestamp.

<Video src="/media/overview.mp4" poster="/media/overview.jpg" caption="Browsing a stream and opening a message." />

## Where to go next

- If binary payloads still show as hex, attach your schemas in [Protobuf](/guide/protobuf).
- To watch messages as they arrive, see [Live tail](/guide/live-tail).
- To send a message back, see [Publishing](/guide/publish).
