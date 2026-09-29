// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// StreamRelationKind is how messages move from one stream to another.
type StreamRelationKind int

const (
	StreamRelationUnspecified StreamRelationKind = iota // unspecified
	StreamRelationSource                                // source
	StreamRelationMirror                                // mirror
	StreamRelationRepublish                             // republish
)

// StreamNodeKind is what a node of the relations graph stands for.
type StreamNodeKind int

const (
	StreamNodeUnspecified StreamNodeKind = iota // unspecified
	StreamNodeStream                            // stream
	StreamNodeKV                                // kv
	StreamNodeObjectStore                       // object store
	StreamNodeExternal                          // external
	StreamNodeMissing                           // missing
	StreamNodeSubject                           // subject
)

// StreamRelations is the replication graph of an account: every stream that sources, mirrors or
// republishes into another one, with placeholders for upstreams that cannot be listed.
type StreamRelations struct {
	// Nodes are the streams and placeholders, sorted by ID.
	Nodes []StreamRelationNode

	// Edges point from the upstream node to the downstream one.
	Edges []StreamRelationEdge
}

// StreamRelationNode is one node of the relations graph.
type StreamRelationNode struct {
	// ID is unique within the graph: the stream name for local streams, prefixed for placeholders.
	ID string

	// Name is the stream name, or the subject for a republish destination no stream captures.
	Name string

	// Kind is what the node stands for.
	Kind StreamNodeKind

	// Info is the stream info without its raw JSON; nil for placeholders.
	Info *StreamInfo

	// External is the API of an upstream in another account or domain; nil for local nodes.
	External *ExternalStreamRef
}

// StreamRelationEdge is one link of the relations graph.
type StreamRelationEdge struct {
	// Kind is how messages move along the link.
	Kind StreamRelationKind

	// From is the ID of the upstream node.
	From string

	// To is the ID of the downstream node.
	To string

	// Source is the source or mirror config of the downstream stream; nil for republish links.
	Source *StreamSourceRef

	// State is the live state of a source or mirror link; nil when the server reports none.
	State *StreamSourceInfo

	// Republish is the republish config of the upstream stream; nil for source and mirror links.
	Republish *StreamRePublish
}
