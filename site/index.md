---
layout: home
title: Natscope
description: Web UI for NATS JetStream — browse, decode, publish and manage streams.

hero:
  name: Natscope
  text: Web UI for NATS JetStream
  tagline: Decodes binary Protobuf payloads into readable JSON. Browse, tail, publish and manage streams from a single binary.
  image:
    src: /owl.png
    alt: Natscope
  actions:
    - theme: brand
      text: Get Started
      link: /guide/what-is-natscope
    - theme: alt
      text: View on GitHub
      link: https://github.com/dmit-4884/natscope
---

<div class="ns-features">
  <a class="ns-card" href="guide/live-tail.html">
    <span class="ns-card-token">LIVE.TAIL.></span>
    <h3>Live tail</h3>
    <p>Switch the message list to Realtime and watch messages land as they arrive. Throttle the display rate down to 1 msg/s when the stream outruns your eyes.</p>
  </a>
  <a class="ns-card" href="guide/protobuf.html">
    <span class="ns-card-token">PROTO.DECODE.*</span>
    <h3>Protobuf decoding</h3>
    <p>Point Natscope at the Git repo or directory holding your .proto files, bind subject patterns to message types, and read binary payloads as JSON.</p>
  </a>
  <a class="ns-card" href="guide/publish.html">
    <span class="ns-card-token">PUBLISH.VALID.></span>
    <h3>Publish console</h3>
    <p>Write JSON with schema autocompletion, validate it against the Protobuf schema before sending, and add custom NATS headers.</p>
  </a>
  <a class="ns-card" href="guide/consumers.html">
    <span class="ns-card-token">CONSUMERS.PULL.*</span>
    <h3>Consumers</h3>
    <p>Create, edit, pause and resume durable and ephemeral consumers. Copy any config as a ready-to-run nats CLI command.</p>
  </a>
  <a class="ns-card" href="guide/kv.html">
    <span class="ns-card-token">KV.REVISIONS.*</span>
    <h3>Key/Value</h3>
    <p>Browse buckets and keys, write with compare-and-set on the revision, and walk the revision history of any key.</p>
  </a>
  <a class="ns-card" href="guide/object-store.html">
    <span class="ns-card-token">OBJ.BUCKETS.></span>
    <h3>Object Store</h3>
    <p>Browse buckets, upload and download objects, seal a bucket when it should stop accepting writes.</p>
  </a>
  <a class="ns-card" href="guide/installation.html">
    <span class="ns-card-token">BIN.SINGLE</span>
    <h3>Single binary</h3>
    <p>The React UI is embedded in the Go binary. No database to provision, no Docker Compose, no CGO.</p>
  </a>
  <a class="ns-card" href="reference/data-locations.html">
    <span class="ns-card-token">LOCAL.FIRST</span>
    <h3>Local-first</h3>
    <p>State lives in one bbolt file under ~/.natscope/data. Credentials go to your OS keychain or an encrypted file vault, never the database.</p>
  </a>
</div>

## Install

```bash
brew install dmit-4884/tap/natscope
natscope
```

Open <http://127.0.0.1:4280>. See [Installation](/guide/installation) for Docker and binary downloads.

<Video src="/media/overview.mp4" poster="/media/overview.jpg" caption="Browsing a stream and opening a message in Natscope." />
