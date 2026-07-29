# Security Policy

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

Please report vulnerabilities via
[GitHub private vulnerability reporting](https://github.com/dmit-4884/natscope/security/advisories/new) —
do not open a public issue.

## Scope notes

- Natscope stores connection credentials, TLS keys and git tokens in the OS keychain or an
  encrypted file vault — never in the bbolt database.
- The server binds to loopback by default and the API is unauthenticated unless `webAuth` is
  configured. Exposing the listener beyond localhost without basic auth and a TLS-terminating
  reverse proxy is unsupported; reports assuming such a setup are out of scope.
