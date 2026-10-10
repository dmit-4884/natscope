# Security Policy

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

Report vulnerabilities through
[GitHub private vulnerability reporting](https://github.com/dmit-4884/natscope/security/advisories/new),
not in a public issue.

## Scope notes

- Natscope stores connection credentials, TLS keys and git tokens in the OS keychain or an
  encrypted file vault. The bbolt database never holds them.
- The server binds to loopback by default, and the API has no authentication until you configure
  `webAuth`. Natscope does not support a listener exposed beyond localhost without basic auth and a
  TLS-terminating reverse proxy; reports that assume such a setup are out of scope.
