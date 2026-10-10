# Contributing to Natscope

Open an issue for a bug or a feature request, or send a pull request.

## Prerequisites

- Go 1.27+
- Node.js 22+
- make

## Getting started

```bash
git clone https://github.com/dmit-4884/natscope.git
cd natscope
make deps     # Go modules, npm dependencies, codegen tools
make build    # frontend + backend; binary lands in ./build/bin/
```

For day-to-day development run the backend and the Vite dev server side by side:

```bash
make dev-backend    # backend on :4280 (stub UI)
make dev-frontend   # Vite dev server on :5173, proxies the API
```

## Before opening a PR

```bash
make lint        # golangci-lint
make test        # backend + frontend tests
make web-verify  # full frontend verification (typecheck, eslint, vitest, build)
```

- Proto contracts live under `proto/services/` and `proto/types/`; after editing them run
  `make proto-generate`. Never edit `proto/gen/` or `web/src/gen/` by hand: `make proto-generate` writes both.
- Commit messages follow Conventional Commits: `fix(scope): ...`, `feat(scope): ...`, `docs: ...`.
- Keep each PR to one concern.

## Reporting issues

Use the issue templates. Report security vulnerabilities the way [SECURITY.md](SECURITY.md) describes,
not in a public issue.
