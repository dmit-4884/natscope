// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ServerInfo is a snapshot of NATS server state and cluster topology.
type ServerInfo struct {
	ServerId     string
	ServerName   string
	Version      string
	Host         string
	Port         int32
	ClusterName  string
	MaxPayload   int64
	ConnectedUrl string
	Jetstream    bool
	AuthRequired bool
	TlsRequired  bool
	ConnectUrls  []string

	// ClientStats holds client connection statistics.
	ClientStats *ClientStatistics

	// JsAccount holds JetStream account stats (nil if JetStream unavailable).
	JsAccount *JetStreamAccountInfo

	// Capabilities holds version-gated feature support (nil if not computed).
	Capabilities *ServerCapabilities
}

// ClientStatistics is connection-level message and byte counters.
type ClientStatistics struct {
	InMsgs     uint64
	OutMsgs    uint64
	InBytes    uint64
	OutBytes   uint64
	Reconnects uint64
}

// JetStreamAccountInfo is the account-level JetStream resource usage and limits.
type JetStreamAccountInfo struct {
	Memory        int64
	Storage       int64
	Streams       int64
	Consumers     int64
	MemoryLimit   int64
	StorageLimit  int64
	StreamLimit   int64
	ConsumerLimit int64
	ApiTotal      uint64
	ApiErrors     uint64
	Domain        string
}
