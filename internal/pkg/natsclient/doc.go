// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package natsclient is the domain-typed client surface for a single NATS /
// JetStream connection: entities in, entities out, no SDK types on the
// boundary. A [Dialer] establishes a [Client]; the concrete nats.go
// implementation lives in the [natsgo] subpackage.
//
// All returned errors are already translated to internal/errs domain
// sentinels (matchable with errors.Is / errors.As) or *errs.NATSAPIError; no
// nats.go or jetstream SDK error type ever escapes an implementation.
//
// [natsgo]: github.com/dmit-4884/natscope/internal/pkg/natsclient/natsgo
package natsclient
