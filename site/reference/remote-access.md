---
title: Remote access
description: Natscope listens on loopback. Exposing it needs ALLOW_REMOTE and basic auth.
---

# Remote access

Natscope binds `127.0.0.1:4280` by default and refuses to start on a wider bind unless you opt in.

The reason is direct: the API has no authentication of its own. Stored credentials are never returned
over it, but anyone who can reach the listener can act through your saved connections: read and publish
messages, delete streams, purge buckets.

## Enable it

Two things are required, and a third is strongly recommended.

```bash
GRPC_WEB_ADDRESS=0.0.0.0:4280 ALLOW_REMOTE=true \
WEB_AUTH__USERNAME=admin WEB_AUTH__PASSWORD=change-me natscope
```

- `GRPC_WEB_ADDRESS` — the wider bind
- `ALLOW_REMOTE=true` — permits it. Without this the server exits at startup.
- `WEB_AUTH__USERNAME` and `WEB_AUTH__PASSWORD` — HTTP basic auth over the whole listener, UI and API
  alike. Browsers show their native credentials prompt; API clients send an `Authorization` header.

Both basic-auth values must be non-empty. Whitespace does not count.

## The insecure override

`ALLOW_INSECURE=true` lets the server start on a non-loopback bind with no basic auth. It logs a
prominent warning and leaves the API open to anyone who can route to it. The published Docker image sets
it, because the container binds all its interfaces for `-p` mapping to work; that is why the run command
publishes on `127.0.0.1`.

The same flag gates a non-loopback bind for the optional internal HTTP server (health, metrics, pprof).

## There is no TLS

The listener speaks cleartext h2c. Basic-auth credentials and API responses, including stored connection
secrets, travel unencrypted.

Put any remote exposure behind a TLS-terminating reverse proxy: nginx, Caddy or Traefik. Terminate TLS
there and forward to Natscope on loopback.

## A safer shape

For one person on one machine, skip all of this and use an SSH tunnel:

```bash
ssh -L 4280:127.0.0.1:4280 user@host
```

Natscope stays on loopback on the remote host, and you reach it at <http://127.0.0.1:4280> locally.
