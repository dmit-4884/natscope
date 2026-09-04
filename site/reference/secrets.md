---
title: Secrets
description: Credentials live in the OS keychain or an encrypted file vault, never in the database.
---

# Secrets

NATS credentials, TLS private keys and Git tokens never touch the bbolt database. They go to a separate
vault, and the database holds only a reference to them.

That split is what makes a [workspace export](/guide/workspace) safe to hand around: it carries the
connection metadata and leaves the secrets behind.

## Backends

Set `SECRETS__BACKEND` (or `secrets.backend` in the config file):

| Value | Behavior |
|-------|----------|
| `auto` | Default. Probe the OS keychain, fall back to the file vault when no keychain is usable. |
| `keyring` | OS keychain only. macOS Keychain, Windows Credential Manager, or Linux Secret Service over D-Bus. |
| `file` | AES-256-GCM encrypted file vault under the data directory. |

`auto` is the right answer on a desktop. Headless Linux boxes and containers have no keychain, so they
land on the file vault, which is why the published Docker image sets `SECRETS__BACKEND=file`.

## The file vault

The file backend writes two files into the data directory:

- `secrets.vault` — the encrypted secrets
- `vault.key` — the encryption key, generated on first use

Both are user-only. Whoever can read both can read your secrets, so a key sitting next to the vault it
opens is obfuscation, not encryption at rest.

For real at-rest protection, supply the key out of band:

```bash
SECRETS__FILE_KEY=<64 hex characters> natscope
```

That is a 256-bit AES key as 64 hex characters. With it set, Natscope writes no `vault.key` at all. Keep
the key somewhere the machine cannot read on its own: a password manager, a CI secret, an operator's
head.

## What counts as a secret

- Connection passwords and tokens
- NKey seeds
- Credentials files (JWT plus seed)
- TLS client keys and CA material
- Git tokens for proto sources

## What Natscope never does

- Store any of the above in `natscope.bolt`
- Write a secret value into the log
- Include a secret in a workspace export
