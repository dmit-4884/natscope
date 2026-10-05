---
title: Connections
description: Save NATS connections with any auth method, TLS material and cluster failover.
---

# Connections

A connection is a saved NATS endpoint: one or more server URLs, an auth method, and optional TLS
material. Natscope keeps as many as you want and switches between them from the header dropdown.

The backend pools connections lazily. It opens a NATS connection on first use and reuses it, so saving a
connection does not dial anything until you select it.

## Add a connection

Go to **Settings → Connections** (or click **New Connection** on the start screen).

1. **Name** the connection, add an optional **Description**.
   Under **Environment**, give it a **Label** such as `PROD` in one of five colors and switch on
   **Read-only** if Natscope must never change anything there (see below).
2. Add URLs under **Servers**. Use **Add server** for each extra node; multiple URLs give cluster
   failover. Supported schemes: `nats://`, `tls://`, `ws://`, `wss://`.
3. Choose an auth method:
   - **None**
   - **User/Pass** — username and password
   - **Token**
   - **NKey** — paste a seed or upload a `.nk` file
   - **Credentials** — paste or upload a `.creds` file (JWT + seed)
4. Optional: click **Add TLS configuration** for a **CA certificate (PEM)**, a **Client certificate**
   and **Client key** (mutual TLS), **Skip certificate verification** for self-signed dev certs, and
   **TLS handshake first** for NATS 2.10+ servers running `tls_handshake_first`.
5. Optional: set an **Inbox prefix** when your account may only receive replies on a private inbox,
   such as `_INBOX_alice` (the same as `--inbox-prefix` in the NATS CLI). Request / Reply and Services
   use it for their replies.
6. Click **Test** to check round-trip time, server version and JetStream availability.
7. Click **Connect**.

## Manage saved connections

Each connection row carries its own actions:

- **Connect** — make it the active connection
- **Ping** — test it without switching
- **Duplicate** — copy it as a starting point for a similar endpoint
- **Edit**
- **Delete**

Rows show a **Connected** chip when active, a node count for multi-URL connections, and TLS badges
(**TLS**, **mTLS**, **TLS skip-verify**).

## Labels and read-only connections

A label shows next to the connection name in the header, in the connection lists, and as a colored
stripe across the top of the window, so a red `PROD` is hard to miss.

A **Read-only** connection lets you browse everything but change nothing. The server refuses every
write through it: publishing, requests, creating, editing, purging, sealing or deleting streams and
consumers, pausing or resetting consumers, deleting messages, and writing KV keys or objects. That
holds for AI agents over [MCP](/guide/mcp) too. The UI hides those actions, and pages that only write,
such as **Publish** and **Request / Reply**, explain why and link to the connection settings.

Reading still works the way it does elsewhere: browsing and searching messages may create short-lived
ordered consumers that the server removes on its own.

## Where credentials live

Passwords, tokens, NKey seeds, credentials files and TLS private keys go to the secret vault: your OS
keychain, or an AES-256-GCM file vault on hosts without one. The bbolt database stores the connection
metadata and a reference to the secret, never the secret itself. See [Secrets](/reference/secrets).

Workspace exports skip secrets entirely. Imported connections are flagged so you know to re-enter
credentials. See [Workspace](/guide/workspace).
