<div align="center">

# Natscope

**A single-binary web UI for NATS JetStream — browse, decode, publish and manage your streams, with first-class Protobuf support.**

[![Release](https://img.shields.io/github/v/release/dmit-4884/natscope)](https://github.com/dmit-4884/natscope/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/dmit-4884/natscope/ci.yml?branch=main&label=CI)](https://github.com/dmit-4884/natscope/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmit-4884/natscope)](https://goreportcard.com/report/github.com/dmit-4884/natscope)
[![License](https://img.shields.io/github/license/dmit-4884/natscope)](LICENSE)

[Quick start](#quick-start) · [Features](#features) · [Configuration](#configuration) · [Remote access](#remote-access)

![Natscope walkthrough](docs/demo/natscope-demo.gif)

*Connect to a cluster → tail a work queue live → inspect consumers → attach a `.proto` source and a subject
mapping → watch binary payloads turn into readable JSON → hand-write and publish a schema-validated
Protobuf message → browse KV revisions.*
**[▶ Watch the full walkthrough in 1080p](docs/demo/natscope-demo-1080p.mp4)**

</div>

---

## What it is

Natscope is a developer tool for everyday work with [NATS](https://nats.io) — a fast web interface
for seeing what actually flows through your streams, including binary Protobuf payloads.

- **Single binary, zero infrastructure.** The React frontend is embedded into the Go binary; all state
  lives in one local [bbolt](https://github.com/etcd-io/bbolt) file under `~/.natscope/data/` (pure Go,
  no CGO). No external database, no Docker Compose, nothing to provision.
- **Protobuf-native.** Point Natscope at the Git repo or local directory that holds your `.proto` files,
  map subjects to message types, and binary payloads become readable JSON everywhere — in the message
  browser, in live tailing, and when publishing.
- **Full JetStream coverage.** Not just read-only browsing: create and edit streams and consumers,
  manage KV buckets and Object Stores, purge, seal, pause — the whole lifecycle from one UI.
- **Secure by default.** Connection credentials, TLS private keys and git tokens live in your OS
  keychain (or an AES-256-GCM file vault on headless hosts) — never in the database. The server listens
  on loopback unless you explicitly opt into remote access.

## Quick start

### Homebrew (macOS / Linux)

```bash
brew install dmit-4884/tap/natscope
natscope
```

### Docker

```bash
# Publish to loopback only — the container binds all its interfaces.
# The image runs as a non-root user and stores data in /data.
docker run -d -p 127.0.0.1:4280:4280 -v natscope-data:/data ghcr.io/dmit-4884/natscope:latest
```

### From source

```bash
make deps   # Go modules, npm deps and codegen tools (first build only)
make build
./build/bin/<version>/<os>-<arch>/natscope
```

Requires Go 1.26+ and Node.js 22+. A clean clone must run `make deps` before `make build`, because the
build regenerates proto code with tools installed by `make deps`.

Then open <http://localhost:4280>, add a connection to your NATS server and start browsing.

## Features

<details>
<summary><b>Connections</b> — saved connections, every NATS auth method, full TLS, cluster failover</summary>

- Multiple saved connections with quick switching, one-click test (RTT, server version, JetStream
  availability), duplication and cluster failover via multiple URLs (`nats://`, `tls://`, `ws://`, `wss://`)
- Every auth method NATS supports: username/password, token, NKey, credentials files (JWT + seed)
- Full TLS configuration: custom CA, mutual TLS (client cert + key), TLS-first handshake
- Fine-grained tuning: timeouts, reconnect policy, ping intervals, inbox prefix, no-echo
- Lazy connection pooling on the backend — connections are established on first use and deduplicated

</details>

<details>
<summary><b>Message browsing</b> — pagination, jump to time, wildcard filters, search, export, edit &amp; resend</summary>

- Paginated browsing forward or backward from any sequence
- **Jump to time** — pick a date/time and open the stream at the first message published at or after it
- Subject filtering with NATS wildcards (`*`, `>`) and full-text search across decoded payloads
- Multiple payload views: decoded JSON, raw text, hex, base64 — with NATS headers alongside
- Export to JSON, NDJSON or CSV; the **full-range** scope walks the whole stream server-side (full
  payloads, no preview truncation) with a progress bar and cancel
- **Edit & resend** — load any message back into the publish form (subject, payload, headers) in one click
- **Delete a message** by sequence (optional secure erase), and bookmark interesting messages with notes

</details>

<details>
<summary><b>Live tailing</b> — real-time subscriptions with on-the-fly Protobuf decoding</summary>

- Real-time subscription to any subject pattern via server streaming — core NATS or bound to a stream
- Messages are Protobuf-decoded on the fly, delivered in batches with live throughput stats
  (msgs/sec, per-subject counts, drop tracking)
- When proto definitions change, active live sessions re-decode with the new schema automatically

</details>

<details>
<summary><b>Publishing</b> — schema-aware editor, reusable templates, automatic publish history</summary>

- Publish raw JSON or Protobuf-encoded messages (JSON in, binary out) with custom NATS headers
- Schema-aware editor: field autocompletion, pre-publish validation against the Protobuf schema and
  example payload generation for any message type
- Reusable publish templates: subject + message type + payload + headers saved as a preset
- Automatic publish history — every publish is logged with subject, encoding, payload and result

</details>

<details>
<summary><b>Protobuf support</b> — Git/local sources, versioning, live reload, subject mappings</summary>

The proto pipeline is the heart of Natscope:

- **Sources** — `.proto` files from Git repositories (token auth, semver-sorted tags), local directories
  or ad-hoc file sets
- **Versioning** — compile a specific Git tag and switch the active version per source; mappings can pin
  to a tag or descriptor fingerprint, so old messages still decode after a schema change
- **Live reload** — local sources can be watched: edit a `.proto` and the new schema reaches every
  connected client within half a second
- **Real-world compilation** — resolves `buf.lock` dependencies from the buf module cache and ships the
  well-known types; compile errors come back as precise diagnostics (file, line, missing-import hints)
- **Subject mappings** — bind subject patterns (wildcards supported) to fully-qualified message types,
  individually or in bulk, with per-mapping health checks
- **Conflict diagnostics** — when several sources define the same symbol, Natscope reports the conflict
  instead of silently picking one, and never blocks decoding over it
- **Registry browser** — explore every loaded message type, its fields and generated example JSON

</details>

<details>
<summary><b>JetStream management</b> — streams, consumers, Key/Value, Object Store, monitoring</summary>

- **Streams** — create and edit with the full config surface (retention, storage, limits, replicas,
  mirrors/sources, subject transforms, republish, compression…), purge, seal, delete
- **Consumers** — create, edit, pause/resume, delete; push and pull, with delivery policies, backoff,
  filtering and flow control
- **Copy as `nats` CLI** — turn any stream or consumer config into a ready-to-paste
  `nats stream add` / `nats consumer add` command
- **Key/Value** — browse buckets and keys, put with optimistic concurrency (compare-and-swap on
  revision), browse per-key revision history, delete and purge history
- **Object Store** — browse buckets, upload/download objects, seal buckets
- **Monitoring** — per-stream and per-consumer stats, server info, cluster topology, connection health

</details>

<details>
<summary><b>Settings &amp; workspace</b> — destructive-action confirmations, workspace export/import</summary>

- **Behavior settings** — per-operation confirmation toggles for destructive actions with an in-dialog
  "Don't ask again"; irreversible-scale operations always keep type-to-confirm
- **Workspace export/import** — back up connections, proto sources, mappings, templates and settings
  into one JSON file, with a dry-run preview and merge or replace strategies on import.
  **Secrets are never exported**; imported entries are flagged to have credentials re-entered

</details>

<details>
<summary><b>UI</b> — command palette, density modes, persisted layout</summary>

- Command palette (`Cmd+K`), comfortable/compact density modes, resizable persisted panels
- JSON viewer with syntax highlighting, config diff preview before applying stream changes
- All preferences persist across sessions

</details>

## Configuration

Natscope works with zero configuration. When you need to tweak it:

| What | How |
|------|-----|
| Listen address | `GRPC_WEB_ADDRESS` (default `127.0.0.1:4280`, loopback only) |
| Config file | `--config` flag or `CONFIG_FILE` env var (templates in `configs/`) |
| Env overrides | any config field; nested keys use double underscores (`LOGGER__LEVEL`) |
| Data directory | `STORAGE__LOCAL__DATA_DIR` (default `~/.natscope/data/`) |
| Secret vault | `SECRETS__BACKEND=auto\|keyring\|file`; for `file`, supply `SECRETS__FILE_KEY` (64 hex chars, AES-256) out-of-band |

## Remote access

The API is unauthenticated by default and returns your saved connection secrets, so Natscope listens on
loopback and **refuses to start on a wider bind** unless you opt in:

```bash
GRPC_WEB_ADDRESS=0.0.0.0:4280 ALLOW_REMOTE=true \
WEB_AUTH__USERNAME=admin WEB_AUTH__PASSWORD=change-me natscope
```

`webAuth` protects the whole listener (UI and API) with HTTP basic auth — required for anything beyond
loopback. Without it Natscope refuses to start on a non-loopback bind; `ALLOW_INSECURE=true` overrides
that and accepts the risk (with a prominent warning in the log).

The listener itself has **no TLS** — it speaks cleartext h2c, so credentials and API responses travel
unencrypted. Any remote exposure must sit behind a TLS-terminating reverse proxy (nginx, caddy, traefik).

## Where it came from

Natscope grew out of a concrete, boring pain: testers and developers could not work with Protobuf
messages in NATS. Payloads are binary, so every check — "did the service publish the right event?" —
meant writing a throwaway decoder, or pasting base64 into some converter and guessing which schema
version produced it.

The first answer was a small console prototype: point it at a subject, give it a `.proto`, get
readable JSON. It solved the decoding problem well enough that people started asking for everything
around it — browsing history instead of tailing, publishing a message back, checking consumers,
looking into KV. The web UI, the JetStream management surface, the proto source/version/mapping
pipeline and the workspace features all grew from that prototype.

## Roadmap

Natscope is under active development and the TODO list is considerably longer than what has already
shipped — the feature set above is the part that is done, not the ceiling. Expect the tool to keep
growing in the same direction: deeper Protobuf tooling, more of the JetStream surface, and better
day-to-day ergonomics for people debugging real systems.

Ideas, requests and bug reports are welcome — open an issue.

## License

Released under the [Apache License 2.0](LICENSE).
