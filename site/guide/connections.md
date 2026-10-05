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
6. Optional: set a **JetStream domain** to manage JetStream in another domain than the server's own,
   such as the hub's streams from a leaf node (`--js-domain` in the NATS CLI), or a **JetStream API
   prefix** when another account exports its JetStream API to yours under a prefix such as
   `JS.orders.API` (`--js-api-prefix`). Set one or the other, not both.
7. Click **Test**. Natscope checks the connection step by step and shows what each step found:
   - **DNS**: the host name resolves
   - **TCP**: something listens on the port
   - **NATS protocol**: the server greets like a NATS server, not an HTTP or monitoring port, and
     whether it waits for a TLS handshake first
   - **TLS**: the handshake, whether the certificate is trusted, matches the host name and when it expires,
     and whether the server wants a client certificate
   - **Authentication**: the credentials are accepted, or a warning when the server checks none
   - **JetStream**: the account has JetStream, in the configured domain or API prefix

   A failed step says what to do next, such as adding the CA certificate or turning on **TLS
   handshake first**. When Natscope runs in Docker and the server is on your machine, it suggests
   `host.docker.internal` instead of `localhost`. The later steps show as skipped. With several
   server URLs, Natscope tries them in order: the test passes when one of them connects, and
   otherwise shows the server that got furthest. A test that runs out of time marks the steps it
   could not finish as skipped, not failed.
8. Click **Connect**.

## Import from the nats CLI

If you already use the `nats` command-line tool, click **From nats CLI** in **Settings → Connections**.
Natscope lists the contexts in `~/.config/nats/context` (or `$XDG_CONFIG_HOME/nats/context`) on the
machine it runs on and marks the one the CLI currently uses. Pick the ones you want and click
**Import**. Each context becomes a connection with its servers, description, inbox prefix, JetStream
domain or API prefix, and its credentials: the `.creds` file, NKey seed, token or user and password,
plus the TLS certificates and keys. They go to the secret vault like any other credential.

Natscope running in Docker or on another host cannot see your contexts. Click **Upload context files**
and pick the `.json` files from that folder instead. With [remote access](/reference/remote-access) allowed,
Natscope never reads its own host's contexts, so other people using it cannot pick up that machine's
credentials: upload the files there too. The credential and certificate files they point at
stay on your machine, so add them to each connection after the import.

Contexts whose name is already taken are skipped, so an import never overwrites a connection. Settings
Natscope cannot carry over are listed under each context: `nsc` references, SOCKS proxies and the
Windows certificate store.

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
