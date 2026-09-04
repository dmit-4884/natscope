---
title: What is Natscope
description: Natscope is a single-binary web UI for NATS JetStream with first-class Protobuf decoding.
---

# What is Natscope

Natscope is a web UI for [NATS](https://nats.io) JetStream. It ships as one Go binary with the React
frontend embedded, serves the UI and its API from a single listener on `127.0.0.1:4280`, and keeps all
state in one local [bbolt](https://github.com/etcd-io/bbolt) file.

The reason it exists: NATS payloads are often binary Protobuf, so answering "did the service publish the
right event?" used to mean writing a throwaway decoder or pasting base64 into a converter. Natscope
compiles your `.proto` files, maps subjects to message types, and renders those payloads as JSON in the
message browser, in live tail, and in the publish preview.

<Video src="/media/overview.mp4" poster="/media/overview.jpg" caption="Browsing a stream and opening a message." />

## What it does

- **Connections** to any NATS server or cluster: user/password, token, NKey and credentials-file auth, plus full TLS and mTLS.
- **Streams** created and edited across the full JetStream config surface, plus purge, seal and delete.
- **Messages** browsed with pagination, subject wildcards, payload search and jump-to-time, exported to
  JSON, NDJSON or CSV.
- **Live tail** of a stream or subject pattern with a display-rate throttle.
- **Consumers** created, edited, paused and resumed, push and pull.
- **Publishing** of JSON or Protobuf messages with schema validation and custom headers.
- **Protobuf sources** from Git repositories, local directories or uploaded files, with versioning and
  live reload.
- **Key/Value and Object Store** buckets browsed and managed from the same UI.

## What it is not

Natscope has no server component beyond the binary you run and no external dependencies. It does not
proxy or store your message data: every read hits your NATS server live. The only thing it persists is
your own configuration (connections, proto sources, mappings, templates, publish history, preferences).

The listener speaks cleartext h2c with no TLS. Keep it on loopback, or read
[Remote access](/reference/remote-access) before exposing it.

## Next steps

- [Installation](/guide/installation)
- [Quick start](/guide/quick-start)
