// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package natsgo is the nats.go implementation of the natsclient interfaces.
// It owns every SDK-facing concern for a single connection: dialing, the
// wire-model converters, fetch strategies, and the single error-translation
// point ([wrapErr]) that maps nats.go / jetstream sentinels to internal/errs
// domain sentinels. No SDK error type escapes a [Client] or [Dialer] method.
package natsgo
