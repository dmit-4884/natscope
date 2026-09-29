<div align="center">

# Natscope

**A single-binary web GUI for NATS JetStream with first-class Protobuf support.**

[![Release](https://img.shields.io/github/v/release/dmit-4884/natscope)](https://github.com/dmit-4884/natscope/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/dmit-4884/natscope/ci.yml?branch=main&label=CI)](https://github.com/dmit-4884/natscope/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/dmit-4884/natscope)](LICENSE)

![Natscope walkthrough](docs/demo/natscope-demo.gif)

**[Documentation](https://natscope.app)** · [Demo in 1080p](docs/demo/natscope-demo-1080p.mp4)

</div>

- **Single binary.** The UI is embedded and all state lives in one local file — no database, nothing to provision.
- **Protobuf-native.** Point it at your `.proto` files and binary payloads read as JSON in history, live tail and publish.
- **All of JetStream.** Streams, consumers, Key/Value and Object Store — browse, edit, purge, pause.
- **Secrets stay secret.** Credentials live in the OS keychain, never in the database.
- **AI agents.** Claude Code or Cursor can read your streams, decoded, over MCP.

## Install

```bash
brew install dmit-4884/tap/natscope
natscope
```

```bash
docker run -d -p 127.0.0.1:4280:4280 -v natscope-data:/data ghcr.io/dmit-4884/natscope:latest
```

Open <http://localhost:4280> and add a connection. Natscope listens on loopback only — read
[remote access](https://natscope.app/reference/remote-access) before exposing it.

[Configuration](https://natscope.app/reference/configuration) · [Building from source](CONTRIBUTING.md) ·
[Apache 2.0](LICENSE)
