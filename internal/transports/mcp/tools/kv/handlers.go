// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package kv

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func (t *Toolset) listBuckets(ctx context.Context, _ *mcp.CallToolRequest, in mcptransport.ConnectionArg) (*mcp.CallToolResult, listBucketsOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, listBucketsOutput{}, err
	}
	buckets, err := t.kv.ListKVBuckets(ctx, connID)
	if err != nil {
		return nil, listBucketsOutput{}, err
	}
	return nil, listBucketsOutput{Buckets: mcptransport.Items(slices.To(buckets, func(b entities.KVBucketInfo) bucketView {
		return *converter.Convert(&b, &bucketView{}, mcptransport.ViewCodecs)
	}))}, nil
}

func (t *Toolset) listKeys(ctx context.Context, _ *mcp.CallToolRequest, in listKeysInput) (*mcp.CallToolResult, listKeysOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, listKeysOutput{}, err
	}
	keys, err := t.kv.ListKVKeys(ctx, connID, in.Bucket)
	if err != nil {
		return nil, listKeysOutput{}, err
	}
	if pattern := strings.TrimSpace(in.Pattern); pattern != "" {
		matched := keys[:0:0]
		for _, k := range keys {
			if natsutil.MatchSubject(pattern, k) {
				matched = append(matched, k)
			}
		}
		keys = matched
	}
	return nil, listKeysOutput{
		Keys:  mcptransport.Items(keys[:min(len(keys), mcptransport.Limit(in.Limit, defaultKeysLimit, maxKeysLimit))]),
		Total: len(keys),
	}, nil
}

func newEntryView(e *entities.KVEntry, limit int) entryView {
	v := converter.Convert(e, &entryView{}, converter.WithIgnoreFields("Value"))
	v.Value, v.Truncated = mcptransport.NewBodyFromBase64(e.Value, limit, false)
	return *v
}

func (t *Toolset) getEntry(ctx context.Context, _ *mcp.CallToolRequest, in keyInput) (*mcp.CallToolResult, entryView, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, entryView{}, err
	}
	entry, err := t.kv.GetKVKey(ctx, connID, in.Bucket, in.Key)
	if err != nil {
		return nil, entryView{}, err
	}
	return nil, newEntryView(entry, mcptransport.Limit(in.MaxValueBytes, defaultValueBytes, maxValueBytes)), nil
}

func (t *Toolset) getHistory(ctx context.Context, _ *mcp.CallToolRequest, in historyInput) (*mcp.CallToolResult, historyOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, historyOutput{}, err
	}
	entries, err := t.kv.GetKVKeyHistory(ctx, connID, in.Bucket, in.Key)
	if err != nil {
		return nil, historyOutput{}, err
	}
	limit := mcptransport.Limit(in.MaxValueBytes, defaultValueBytes, maxValueBytes)
	count := min(len(entries), mcptransport.Limit(in.Limit, defaultHistoryLimit, maxHistoryLimit))
	out := historyOutput{Entries: make([]entryView, 0, count), Total: len(entries)}
	for i := len(entries) - 1; i >= len(entries)-count; i-- {
		out.Entries = append(out.Entries, newEntryView(&entries[i], limit))
	}
	return nil, out, nil
}
