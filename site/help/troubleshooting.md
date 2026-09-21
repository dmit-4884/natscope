---
title: Troubleshooting
description: Common Natscope problems and how to fix them.
---

# Troubleshooting

## The server exits at startup on a non-loopback bind

Natscope refuses to start when `GRPC_WEB_ADDRESS` points at anything but a loopback address and you have
not opted in. Set `ALLOW_REMOTE=true` plus `WEB_AUTH__USERNAME` and `WEB_AUTH__PASSWORD`:

```bash
GRPC_WEB_ADDRESS=0.0.0.0:4280 ALLOW_REMOTE=true \
WEB_AUTH__USERNAME=admin WEB_AUTH__PASSWORD=change-me natscope
```

Empty or whitespace-only credentials do not satisfy the gate. `ALLOW_INSECURE=true` skips the auth
requirement and logs a warning. Read [Remote access](/reference/remote-access) before you use it.

## The UI returns 404

This one is specific to builds from source. The React app is embedded from `web/dist` at compile time,
so a binary built without a frontend serves 404 on every UI route while the API keeps working.

Run `make build-frontend` and rebuild, or `make stub-dist` if you only need the backend. Released
binaries, Homebrew installs and the Docker image all ship the UI already embedded.

## A connection fails

Use **Test** on the connection form. It reports round-trip time, server version and JetStream
availability, which separates "cannot reach the server" from "reached it, wrong credentials".

Check, in order:

- The URL scheme matches the server: `nats://`, `tls://`, `ws://` or `wss://`
- The port is right (`4222` for plain NATS by default)
- The auth method matches what the server expects
- For self-signed certs in development, tick **Skip certificate verification**
- For NATS 2.10+ servers running `tls_handshake_first`, tick **TLS handshake first**

Remember that the NATS connection is made by the Natscope process, not your browser. A server reachable
from your laptop is not automatically reachable from a container.

## Payloads show as hex instead of JSON

Natscope decodes a payload only when a subject mapping resolves a message type for that subject. Check
in this order:

1. **Settings → Proto Files** — is a source present and compiled without errors? Compile diagnostics
   name the file, line and any missing import.
2. **Settings → Mappings** — does a mapping cover this subject? Wildcards count: `*` matches one token,
   a trailing `>` matches one or more.
3. The mapping's health badge. A missing source, a stale version selection or a type that no longer
   exists all break decoding, and the badge says which.

See [Protobuf](/guide/protobuf).

## Where the logs go

Natscope logs to stderr. In a terminal the default is the compact `console` format at the `warning`
level, so a healthy start prints the banner on stdout and nothing else. Under Docker or a service
manager the default is `text` at `info`. Turn up the detail with the `--log-level` flag, which overrides
both `LOGGER__LEVEL` and the config file:

```bash
natscope --log-level debug
```

For machine-readable output, pass `--log-format json` or set `LOGGER__OUTPUT_FORMAT=json`.
`LOGGER__OUTPUT_SOURCE=true` adds the file and line of each entry. Under Docker,
`docker logs <container>` shows the same stream.

## Health and metrics endpoints return nothing

The internal HTTP server is off by default. With no `http` section in the config, `/internal/healthz`,
`/internal/readyz`, `/internal/metrics` and pprof do not exist. Enable it explicitly:

```yaml
http:
  listenAddress: "127.0.0.1:9080"
  pprof:
    enabled: true
```

Keep it on loopback. Those endpoints have no authentication.

## Credentials disappear after a restart

Check which vault backend is active. On a headless Linux host or in a container there is no OS keychain,
so `auto` falls back to the encrypted file vault, which needs a writable data directory. If the data
directory is not persisted (a container with no volume mounted at `/data`), everything goes away with
the container.

Mount a volume, and set `SECRETS__BACKEND=file` when you want the file vault regardless. See
[Secrets](/reference/secrets).

## Still stuck

Open an issue at [github.com/dmit-4884/natscope/issues](https://github.com/dmit-4884/natscope/issues)
with the output of `natscope version --full` and the relevant log lines at `LOGGER__LEVEL=debug`.
