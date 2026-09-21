---
title: Data locations
description: Where Natscope keeps its database, vault and service state, and what to back up.
---

# Data locations

Everything Natscope persists lives in a handful of local files. There is no external database and
nothing in the cloud.

## Data directory

Default `~/.natscope/data`, overridden with `STORAGE__LOCAL__DATA_DIR`.

| File | What it holds |
|------|---------------|
| `natscope.bolt` | The bbolt database: connections, proto sources and versions, subject mappings, templates, publish history, bookmarks, settings. One bucket per domain, one JSON document per row. |
| `secrets.vault` | The encrypted secret vault, present only when the `file` backend is in use. |
| `vault.key` | The vault encryption key, written only when `SECRETS__FILE_KEY` is unset. |

On a keychain backend, the secrets sit in the OS keychain instead and neither vault file exists. See
[Secrets](/reference/secrets).

## Service directories

Separate from the data directory, and rarely interesting:

| Path | Env | What it holds |
|------|-----|---------------|
| `/var/lib/natscope` | `LIB_DIR` | The service instance id file and a TLS certificate cache under `certs/` |
| `/var` | `VAR_DIR` | Base for the above |

When the system paths are not writable and neither variable is set, Natscope falls back to
`~/.natscope/lib` and `~/.natscope/var`.

## Docker

The published image pins everything under the `/data` volume:

```
STORAGE__LOCAL__DATA_DIR=/data
LIB_DIR=/data/lib
VAR_DIR=/data/var
```

Mounting a named volume at `/data` preserves your workspace across container restarts and image
upgrades.

## What to back up

**Back up `natscope.bolt`.** It carries your whole workspace. Stop Natscope first so bbolt is not
mid-write.

**Back up `vault.key` separately, or not at all.** Copying it next to `secrets.vault` defeats the point
of encrypting the vault. If you set `SECRETS__FILE_KEY`, store that key in a password manager instead.

**Prefer a workspace export for anything you share.** **Settings → Workspace** writes one JSON file with
connections, proto sources, mappings, templates and settings, and drops every secret on the way out. It
is portable across machines and safe to commit. See [Workspace](/guide/workspace).

## Moving to a new machine

1. Export your workspace on the old machine.
2. Install Natscope on the new one.
3. Import the file.
4. Re-enter credentials for each connection: the export carried none.
