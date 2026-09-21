---
title: Configuration
description: Config file, environment variables and defaults for the Natscope server.
---

# Configuration

Natscope runs with zero configuration. Every key below has a working default. Override only what you need.

## How settings resolve

Two sources, environment wins:

1. **A YAML config file**, passed with `--config` (or `-c`) or the `CONFIG_FILE` environment variable.
   `CONFIG_FILE` takes precedence over the flag. Natscope fails to start if the path does not exist.
2. **Environment variables**, which override anything in the file.

Nested YAML keys map to environment variables with double underscores. `logger.level` becomes
`LOGGER__LEVEL`, `storage.local.dataDir` becomes `STORAGE__LOCAL__DATA_DIR`.

```bash
natscope --config /etc/natscope/config.yaml
CONFIG_FILE=/etc/natscope/config.yaml natscope
```

Commented templates for every section live in [`configs/`](https://github.com/dmit-4884/natscope/tree/main/configs)
in the repository.

::: tip Env prefix
Environment keys carry a prefix that is set at build time. Released binaries ship with no prefix, so the
names below are used as written. Run `natscope version --full` to print the active prefix.
:::

## Keys

### Listener

| Key | Env | Default | What it does |
|-----|-----|---------|--------------|
| `grpcWebAddress` | `GRPC_WEB_ADDRESS` | `127.0.0.1:4280` | Address serving the API and the embedded UI |
| `allowRemote` | `ALLOW_REMOTE` | `false` | Permit a non-loopback bind |
| `allowInsecure` | `ALLOW_INSECURE` | `false` | Permit a non-loopback bind with no basic auth |
| `webAuth.username` | `WEB_AUTH__USERNAME` | unset | HTTP basic auth username for the whole listener |
| `webAuth.password` | `WEB_AUTH__PASSWORD` | unset | HTTP basic auth password |

One listener speaks Connect, gRPC-Web and gRPC, and serves the SPA. There is no separate gRPC port. See
[Remote access](/reference/remote-access) before binding beyond loopback.

### Storage

| Key | Env | Default | What it does |
|-----|-----|---------|--------------|
| `storage.local.dataDir` | `STORAGE__LOCAL__DATA_DIR` | `~/.natscope/data` | Directory holding `natscope.bolt` and the file vault |

### Secrets

| Key | Env | Default | What it does |
|-----|-----|---------|--------------|
| `secrets.backend` | `SECRETS__BACKEND` | `auto` | Vault backend: `auto`, `keyring` or `file` |
| | `SECRETS__FILE_KEY` | unset | AES-256 key for the file vault, 64 hex characters |

See [Secrets](/reference/secrets).

### Logging

| Key | Env | Default | Values |
|-----|-----|---------|--------|
| `logger.level` | `LOGGER__LEVEL` | `warning` in a terminal, `info` otherwise | `error`, `warning`, `info`, `debug`, `none` |
| `logger.output` | `LOGGER__OUTPUT` | `stderr` | `stdout`, `stderr` |
| `logger.outputFormat` | `LOGGER__OUTPUT_FORMAT` | `console` in a terminal, `text` otherwise | `console`, `text`, `json` |
| `logger.colorized` | `LOGGER__COLORIZED` | `true` | Color for `console` and `text`, ignored for `json` |
| `logger.outputSource` | `LOGGER__OUTPUT_SOURCE` | `false` | Add file and line to each entry |

"In a terminal" means stderr is a TTY. The `--log-level` and `--log-format` flags override both the
environment and the config file for those two keys.

### Service directories

| Env | Default | What it does |
|-----|---------|--------------|
| `VAR_DIR` | `/var` | Base directory for variable runtime data |
| `LIB_DIR` | `<VAR_DIR>/lib/natscope`, so `/var/lib/natscope` | Service instance id file and the TLS certificate cache (`<LIB_DIR>/certs`) |

When neither variable is set and the system paths are not writable, Natscope falls back to
`~/.natscope/var` and `~/.natscope/lib` on its own. Set either variable and that fallback stops: if
Natscope cannot create the directory you named, it fails to start.

### Internal HTTP server

Off by default. With no `http` section in the config, it never starts. Enable it for health, readiness
and metrics endpoints under `/internal`, plus opt-in pprof.

```yaml
http:
  listenAddress: "127.0.0.1:9080"
  pprof:
    enabled: true
```

Keep it on loopback. These endpoints expose runtime internals and carry no authentication. A
non-loopback bind here needs `ALLOW_INSECURE=true`.

## Example

```yaml
logger:
  level: debug
  outputFormat: json

grpcWebAddress: "127.0.0.1:4280"

storage:
  local:
    dataDir: /var/lib/natscope/data

secrets:
  backend: file
```

## Docker defaults

The published image sets its own environment so port mapping and containers work out of the box:

```
GRPC_WEB_ADDRESS=0.0.0.0:4280
ALLOW_REMOTE=true
ALLOW_INSECURE=true
SECRETS__BACKEND=file
STORAGE__LOCAL__DATA_DIR=/data
LIB_DIR=/data/lib
VAR_DIR=/data/var
```

The container binds all its interfaces so `-p` works. Publish on `127.0.0.1` to keep it host-local, or
set `WEB_AUTH__USERNAME` and `WEB_AUTH__PASSWORD` when it is reachable from elsewhere.
