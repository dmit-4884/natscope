---
title: Installation
description: Install Natscope with Homebrew, Docker, or a binary from GitHub Releases.
---

# Installation

Natscope runs on macOS, Linux and Windows. Pick one of the four routes below, then open
<http://127.0.0.1:4280>.

## Homebrew

macOS and Linux:

```bash
brew install dmit-4884/tap/natscope
natscope
```

`natscope` with no subcommand starts the server. `natscope server run` does the same thing.

## Docker

```bash
docker run -d -p 127.0.0.1:4280:4280 -v natscope-data:/data ghcr.io/dmit-4884/natscope:latest
```

Publishing on `127.0.0.1` keeps the UI host-local: the container itself binds all its interfaces so that
`-p` port mapping works. The image runs as a non-root user, stores everything in `/data`, and uses the
encrypted file vault for secrets because containers have no OS keychain.

## Binary

Download an archive for your OS and architecture from
[GitHub Releases](https://github.com/dmit-4884/natscope/releases/latest), extract it, and run the
`natscope` binary.

## From source

Requires Go 1.26+ and Node.js 22+.

```bash
make deps
make build
./build/bin/<version>/<os>-<arch>/natscope
```

A clean clone must run `make deps` before `make build`: the build regenerates proto code with tools that
`make deps` installs.

## Open the UI

Once the server is up, open <http://127.0.0.1:4280> and add a connection to your NATS server. Head to
the [Quick start](/guide/quick-start) from there.

To listen somewhere else, set `GRPC_WEB_ADDRESS`:

```bash
GRPC_WEB_ADDRESS=127.0.0.1:9090 natscope
```

Binding to anything other than loopback needs [remote access](/reference/remote-access) turned on
explicitly.
