// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package reasoncodes maps this service's protovalidate rule IDs to canonical,
// client-facing validation reason codes.
//
// A reason code is a public contract and must never be the raw rule ID (a
// dotted, implementation-specific identifier). go-atlas no longer derives one
// itself, so the service owns the mapping: this package feeds the protovalidate
// interceptor via bufhelpers.WithResolver / NewResolver. Standard protovalidate
// rules (gte/lte/min_len/uuid/...) are mapped by the local Resolver
// (resolver.go); the domain CEL rules below extend it.
package reasoncodes

// Prefix is the rule-ID namespace this service's custom CEL rules live under.
// Passing it to the resolver makes any uncataloged rule in this namespace
// resolve to "" (no code) instead of falling through to a standard rule.
const Prefix = "natscope."

// Catalog maps domain (custom CEL) rule IDs to canonical reason codes.
var Catalog = map[string]string{
	// Message-level (cross-field) rules.
	"natscope.nats.messages.start_seq_xor_start_time": "MUTUALLY_EXCLUSIVE_FIELDS",
}
