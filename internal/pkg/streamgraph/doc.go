// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package streamgraph builds the relations graph of an account's streams from their infos: which
// streams source, mirror or republish into which, with placeholders for upstreams in other accounts
// or domains, upstreams that do not exist, and republish subjects no stream captures.
package streamgraph
