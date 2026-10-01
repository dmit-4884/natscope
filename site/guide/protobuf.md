---
title: Protobuf
description: Compile .proto sources from Git, the Buf Schema Registry, disk or an upload, and decode binary payloads by subject.
---

# Protobuf

Natscope decodes binary Protobuf payloads into JSON everywhere it shows a message: the message
browser, live tail, Key/Value values, request replies and the publish preview. Two pieces make that work.

1. A **proto source** gives Natscope your `.proto` files and compiles them into a schema.
2. A **subject mapping** binds a NATS subject pattern to a fully-qualified message type.

Payloads on subjects without a mapping still decode when one message type fits them better than any other. See
[Type detection](#type-detection).

## Add a proto source

Go to **Settings → Proto Files** and click **Add source**. Pick one of four kinds.

- **Git Repository**: Natscope clones the repository over HTTPS, with a token for private ones, and
  compiles the branch, tag or commit you pick.
- **Buf Schema Registry**: enter a module such as `buf.build/acme/payments`. Natscope loads the label or
  commit you pick, the default label first. Private modules need a BSR token.
- **Local Directory**: a path on the machine running Natscope. Natscope can watch it for changes.
- **Upload**: drop `.proto` files, a folder, or a compiled descriptor set (`buf build -o set.binpb` or
  `protoc --descriptor_set_out --include_imports`). Uploading again stores a new version.

Compilation resolves `buf.lock` dependencies from the buf module cache and ships the well-known types.
When it fails, the diagnostics name the file, the line and the missing import, and the previous schema
keeps decoding.

### Versions

Git and BSR sources store a revision for every branch, tag, label or commit they compile. Switch the
active one on the source card. An upload stores each version you upload and makes the latest active. A
local directory keeps only its latest compile. Natscope keeps the 20 newest revisions of a source, plus
the active one and any revision a mapping pins.

Mappings decode with the active revision unless you pin them. See [Pin a schema version](#pin-a-schema-version).

### Live reload

Edit a `.proto` in a watched local directory and every connected browser gets the recompiled schema in
under half a second. Live tails re-decode with it without reconnecting.

## Map subjects to types

Go to **Settings → Mappings** and click **Add mapping**. The form has five steps.

1. **Source**: the proto source to take the type from.
2. **Proto message type**: the fully-qualified type.
3. **Subject pattern**: the NATS pattern that decodes with it, wildcards included. Key/Value values use
   the subject `$KV.<bucket>.<key>`, so `$KV.config.>` maps a whole bucket.
4. **Framing**: what wraps the message on the wire. See [Framing](#framing).
5. **Schema version**: the active schema, or a pinned revision.

**Import**, **Export** and **Copy** in the toolbar move mappings between workspaces in bulk.

### Framing

Some producers wrap the Protobuf message in a few extra bytes. Set the framing on the mapping and
Natscope strips it before decoding and adds it back when you publish.

| Framing | Bytes around the message |
|---------|--------------------------|
| **gRPC frame** | A compression flag and a 4-byte length. Natscope inflates gzip-compressed frames. |
| **Confluent Schema Registry** | A magic byte, the 4-byte schema id and the message indexes. Set the schema id the producer uses. |
| **Varint length-delimited** | A varint length, as written by `writeDelimitedTo`. |
| **Custom prefix and suffix bytes** | Fixed bytes before and after the message, entered as hex. |

When a payload fails to decode and looks framed, the error tells you which framing to set.

### Pin a schema version

A pinned mapping keeps decoding with one revision after its source moves on, so messages written under
an older schema still read correctly. Pick the revision under **Schema version**. The mapping table marks
pinned rows. Publishing, requests, Key/Value saves and editor completion on a pinned subject use the same
revision. A local directory keeps a single revision, so its mappings cannot be pinned.

### Health checks

Each mapping reports its own health: a missing source, a pinned revision that is gone, or a type the
schema no longer has. A broken mapping shows up in the list instead of failing quietly at decode time.

## Type detection

A binary payload on a subject without a mapping still decodes when exactly one message type of your
enabled sources decodes every byte and scores higher than every other type. The message viewer marks it
**Auto-detected**. Click **Save as mapping** to keep the type for that subject, with numeric and UUID
tokens turned into `*`.

For a payload that matches nothing on its own, open **Detect type** in the viewer. It ranks every
message type by how well the payload decodes as it, with a fit score and a decoded preview. Pick one and
click **Decode as this type**, or save it as a mapping under a pattern you choose.

Natscope remembers the detected type per subject and forgets it when a schema reloads. Turn detection off
under **Settings → Preferences → Messages → Detect message types**.

## Read a payload

The message viewer shows a decoded payload under **Decoded**, next to **Raw**, **Hex** and **Wire**.
**Wire** reads the bytes without a schema: field numbers, wire types and every plausible reading of
each value, with nested messages expanded.

Two notices explain an imperfect decode.

- **Fields not in the schema**: the payload carries field numbers the type does not declare. The
  producer probably uses a newer schema. Natscope lists them by field number and size.
- **Decoded the first N bytes**: the payload breaks partway through. Natscope shows what decoded before
  the break and the error that stopped it.

## Edit JSON for a type

The publish and request editors know the type they encode to. They complete field names at any depth, inside nested messages, repeated fields and map values, and suggest enum values and
`true`/`false`. Suggestions keep the schema's field order and show each field's comment.

## Schema browser

**Settings → Proto Files → Schema browser** lists the messages, enums and services of every active
schema, grouped by package, with their comments. Open a type to see its fields, the types it uses, the
mappings that decode with it and an example JSON payload, which doubles as a starting point for
[publishing](/guide/publish). Types pulled in from imports stay hidden until you turn on **Show imported types**.

## Conflicts

When two enabled sources define the same file path with different content, or the same type name,
**Settings → Proto Files** lists them under **Schema conflicts**. A **Clash** means the two definitions
differ; a **Duplicate** means they match, usually a shared file vendored twice. Each mapping names its
source, so decoding stays correct either way, but type pickers list a clashing type twice.
