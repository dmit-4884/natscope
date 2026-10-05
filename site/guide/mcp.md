---
title: AI agents (MCP)
description: Let Claude Code, Cursor or any MCP client read your streams with Protobuf payloads already decoded.
---

# AI agents (MCP)

Natscope serves a [Model Context Protocol](https://modelcontextprotocol.io) endpoint at `/mcp` on the same
listener as the UI. An agent connected to it works through your saved connections and sees payloads the way
the UI shows them: Protobuf decoded to JSON through your subject mappings, credentials never exposed.

Ask things like "did the order service publish `OrderCreated` for order 123 in the last 10 minutes?",
"which consumers are lagging?" or "why doesn't `payments.eu.captured` decode?".

## Connect a client

Start Natscope, then point the client at the endpoint.

Claude Code:

```bash
claude mcp add --transport http natscope http://127.0.0.1:4280/mcp
```

Clients that only launch a command (stdio) use `natscope mcp`, which relays every call to the running server:

```json
{ "mcpServers": { "natscope": { "command": "natscope", "args": ["mcp"] } } }
```

`natscope mcp` reads the listen address and `webAuth` credentials from the same config as the server. Pass
`--url` to target another address. It exits with an error when Natscope is not running, and it lists the tools
once at startup: restart the client after changing `mcp.allowWrites`.

## Tools

| Tool | What it does |
|------|--------------|
| `list_connections` | Saved connections with URLs, label, read-only flag and the last test result, no credentials |
| `get_server_info` | Server version, cluster, max payload, JetStream account usage |
| `list_streams`, `get_stream` | Streams with subjects, limits and state |
| `get_stream_relations` | Sources, mirrors and republish targets with filters, transforms, lag, last activity and errors; one stream's links when `stream` is set |
| `list_consumers` | Consumer progress, most pending first: pending, ack pending, redeliveries; all streams when `stream` is omitted, naming the streams whose consumers the user may not list |
| `find_messages` | A page of stored messages, newest first, by subject and time; `contains` (optionally a regex) and `header` search the whole stream on the server, 100,000 messages per call |
| `get_message` | One message by sequence with headers and the full payload |
| `tail_subject` | Messages published on a subject during the next few seconds |
| `list_message_types`, `describe_message_type` | Compiled Protobuf types, their fields and an example payload |
| `list_mappings`, `resolve_subject` | Subject mappings and why a subject decodes, or does not |
| `decode_payload`, `validate_payload` | Decode base64 bytes, check JSON against a type and its `buf.validate` rules |
| `detect_message_type` | Rank every loaded type by how well base64 bytes decode as it, for payloads no mapping covers |
| `get_schema_status` | Loaded types and conflicts between proto sources |
| `list_kv_buckets`, `list_kv_keys`, `get_kv_entry`, `get_kv_history` | Key/Value buckets, keys and revisions, Protobuf values decoded |
| `publish_message` | Publish JSON (encoded to Protobuf when the subject is mapped) or text. Only with `mcp.allowWrites` |
| `request_message` | Send a core NATS request and return the first reply. Only with `mcp.allowWrites` |

Tools that talk to NATS take a `connection` argument, a saved connection's name or id. With a single saved
connection it can be omitted. A payload or value decoded by [type detection](/guide/protobuf#type-detection)
rather than a mapping carries `decodedAuto`, so the agent knows the type is a guess. Payloads are clipped to a byte budget (a page or tail carries at most 256 KiB);
a clipped message is marked `truncated` and `get_message` fetches more of it.

## Writes

The endpoint is read-only by default. Set `mcp.allowWrites: true` (env `MCP__ALLOW_WRITES=true`) to add
`publish_message` and `request_message`. Every publish lands in the [publish history](/guide/history), so you can
see what the agent sent. A request is not recorded, but the service that answers it may act on it, so it counts as
a write. Deleting, purging and sealing are not exposed over MCP at all.
A [read-only connection](/guide/connections#labels-and-read-only-connections) refuses both tools even with writes on.

Message payloads reach the agent as data. A payload that contains instructions can still influence a model,
which is one more reason to keep writes off unless you need them.

## Security

`/mcp` sits behind the same guards as the UI and API: the loopback-only default, the `Host` check and
`webAuth`. See [Remote access](/reference/remote-access) before exposing the listener beyond localhost.
Turn the endpoint off with `mcp.enabled: false`.
