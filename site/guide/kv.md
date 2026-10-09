---
title: Key/Value
description: Browse and watch JetStream KV buckets, find keys with NATS patterns, give keys a TTL, and edit or clear buckets.
---

# Key/Value

**KV Stores** in the sidebar lists the JetStream Key/Value buckets on the connected server. Open one to
browse its keys and edit values.

Writes are revision-checked. Natscope sends the revision you loaded, so a stale edit gets rejected
instead of silently overwriting someone else's change.

## Create a bucket

Click the **+** next to **KV Stores**, or **New KV bucket** on the overview page. The form groups the
bucket config into **Basic Configuration**, **Limits**, **Storage Options** (storage type, replicas,
compression), **Placement**, **Mirror**, **Sources**, **Republish** and **Metadata**. Fields marked
immutable can only be set at creation time.

**Key TTL marker** under **Limits** lets single keys carry their own TTL (NATS 2.11+). It also sets how long a
marker stays after a key expires, so watchers see the expiry. Once on, it can't be turned off.

## Edit or clear a bucket

The **⋯** menu of an open bucket has:

- **Edit bucket…** — change the description, history, TTL, size limits, replicas, compression, key TTL
  marker and metadata. You review the changes before they apply. Storage type, mirror, sources, republish
  and placement stay as they are.
- **Clear bucket…** — remove every key and every revision in one purge and keep the bucket. Apps watching
  the bucket aren't told: no delete markers are left, so the values they cached stay until they reload.
- **Delete bucket…** — remove the bucket and its data.

Clearing and deleting ask you to type the bucket name.

## Find keys

The search box above the key list takes plain text or a NATS pattern:

- Plain text filters the loaded keys, ignoring case.
- A pattern with `*` or `>`, such as `orders.*` or `users.>`, goes to the server, which returns only the
  matching keys.

The list loads up to 1,000 keys. When more match, a note says so; narrow the pattern to see the rest.

## Work with keys

Select a bucket, then a key. The editor gives you:

- **Save Value** — write a new revision (compare-and-set on the revision you loaded)
- **Reset** — discard your edits
- **History** — open the revision history
- **Purge** — drop the key's history
- **Delete** — remove the key

**New key** or **Create New Key** adds one. In create mode the save button reads **Create Key**.

On a bucket with a key TTL marker, the new key form also takes a **TTL** such as `30s`, `5m` or `1h`.
The key is removed once it expires, and its header shows the TTL and when it expires. A TTL can only be set
when a key is created.

## Watch changes

Turn on **Live updates** above the key list to watch the bucket. Natscope then:

- adds keys to the list and drops them as they are written, deleted or purged;
- reloads the open key when it changes;
- lists the latest changes under **Changes**, newest first, with the operation, revision, time and the
  start of the value. Pick one to open the key.

The watch follows the pattern in the search box, and it reports changes made after you turned it on.
If you're editing a value and someone else changes the key, the editor says so. Saving then fails the revision
check instead of overwriting their change.

## Protobuf values

A value decodes when a [subject mapping](/guide/protobuf) matches `$KV.<bucket>.<key>`, for example
`$KV.config.>` for a whole bucket, or when [type detection](/guide/protobuf#type-detection) recognizes it.
A bar above the editor names the type, and the editor holds the value as JSON.
**Save Value** encodes the JSON back to Protobuf, with the mapping's framing. **Raw bytes** shows the
stored bytes field by field instead.

A detected type carries an **Auto-detected** badge. **Save as mapping** maps the whole bucket to it.
When you create a key that a mapping covers, the form says which type the JSON becomes.

## Revision history

**History** lists every revision of a key with its operation badge: **put**, **delete** or **purge**.
Walk back through them to see what a value looked like and when it changed. Protobuf revisions show as
decoded JSON.

## Confirmations

Deleting a key and purging its history both prompt by default. Both prompts have their own toggle under
**Settings → Preferences → Behavior**.
