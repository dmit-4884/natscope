---
title: Protobuf
description: Compile .proto sources from Git or disk and map subjects to message types.
---

# Protobuf

Natscope decodes binary Protobuf payloads into JSON everywhere it shows a message: the browser, live
tail and the publish preview. Two pieces make that work.

1. A **proto source** gives Natscope your `.proto` files and compiles them into descriptors.
2. A **subject mapping** binds a NATS subject pattern to a fully-qualified message type.

<Video src="/media/proto.mp4" poster="/media/proto.jpg" caption="Attaching a proto source and mapping, then decoding a binary payload." />

## Add a proto source

Go to **Settings → Proto Files** and click **Add source**. Three kinds:

- **Git repository** — clone over HTTPS with a token, pick a tag. Tags sort by semver.
- **Local directory** — a path on the machine running Natscope.
- **Uploaded files** — an ad-hoc set of `.proto` files.

Compilation resolves `buf.lock` dependencies from the buf module cache and ships the well-known types.
When it fails, the diagnostics name the file, the line and the missing import.

<Shot src="/media/proto-sources.png" alt="Proto sources" />

### Versions

A Git source compiles a specific tag. Switch the active version per source when your schema moves on.
Mappings can pin to a tag or to a descriptor fingerprint, so messages written under an older schema keep
decoding after the schema changes.

### Live reload

Local sources can be watched. Edit a `.proto` on disk and the recompiled schema reaches every connected
browser in under half a second. Active live tails re-decode with it without reconnecting.

## Map subjects to types

Go to **Settings → Mappings** and add one. The form is three steps:

1. **Source** — which proto source to pick a type from
2. **Proto message type** — the fully-qualified type
3. **Subject pattern** — the NATS pattern that should decode with it, wildcards included

Click **Add mapping**. Use **Import**, **Export** and **Copy** in the toolbar to move mappings in bulk
between workspaces.

<Shot src="/media/proto-mappings.png" alt="Subject mappings" />

### Health checks

Each mapping reports its own health: a missing source, a stale version selection, or a type that no
longer exists in the compiled descriptors. Broken mappings show up in the list instead of failing
silently at decode time.

### Conflicts

When two sources define the same symbol, Natscope reports the conflict rather than picking one at
random. Decoding keeps working while you sort it out.

## Browse the registry

**Settings → Proto Files** has an **Available messages** section listing every message type compiled
from the active source versions, grouped by package. Open a type to see its fields and a generated
example JSON payload, which doubles as a starting point for [publishing](/guide/publish).
