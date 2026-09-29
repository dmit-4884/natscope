// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package kv holds the read-only MCP tools over JetStream Key/Value buckets.
package kv

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

const (
	defaultKeysLimit    = 200
	maxKeysLimit        = 5000
	defaultValueBytes   = 16 << 10
	maxValueBytes       = 1 << 20
	defaultHistoryLimit = 20
	maxHistoryLimit     = 100
)

// Toolset serves list_kv_buckets, list_kv_keys, get_kv_entry and get_kv_history.
type Toolset struct {
	conns *mcptransport.Connections
	kv    natssvc.KVStore
}

// New creates the Key/Value toolset.
func New(conns *mcptransport.Connections, kv natssvc.KVStore) *Toolset {
	return &Toolset{conns: conns, kv: kv}
}

// Register adds the toolset's tools to s.
func (t *Toolset) Register(s *mcp.Server) {
	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "list_kv_buckets",
		Description: "List the Key/Value buckets of a connection with their size, history depth and TTL.",
		Annotations: mcptransport.ReadOnly("List KV buckets"),
	}, t.listBuckets)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "list_kv_keys",
		Description: "List the keys of a Key/Value bucket, optionally matching a pattern with NATS wildcards (e.g. users.*.profile).",
		Annotations: mcptransport.ReadOnly("List KV keys"),
	}, t.listKeys)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "get_kv_entry",
		Description: "Read the current value of a key with its revision and timestamp.",
		Annotations: mcptransport.ReadOnly("Get KV entry"),
	}, t.getEntry)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "get_kv_history",
		Description: "Read the stored revisions of a key, newest first, including delete and purge markers.",
		Annotations: mcptransport.ReadOnly("Get KV history"),
	}, t.getHistory)
}
