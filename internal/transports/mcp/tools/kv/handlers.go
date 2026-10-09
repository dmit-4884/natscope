// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package kv

import (
	"context"
	"encoding/base64"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

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
	list, err := t.kv.ListKVKeys(ctx, connID, in.Bucket, entities.KVKeysQuery{
		Filter: in.Pattern,
		Limit:  mcptransport.Limit(in.Limit, defaultKeysLimit, maxKeysLimit),
	})
	if err != nil {
		return nil, listKeysOutput{}, err
	}
	return nil, listKeysOutput{Keys: mcptransport.Items(list.Keys), Truncated: list.Truncated}, nil
}

func newEntryView(e *entities.KVEntry, limit int, r *entities.DecodeResult) entryView {
	v := converter.Convert(e, &entryView{}, converter.WithIgnoreFields("Value"))
	if r != nil {
		v.DecodedType, v.DecodedAuto, v.DecodeError = r.MessageType, r.Auto, r.Error
		if r.Success && len(r.Decoded) <= limit {
			v.Decoded = r.Decoded
			return *v
		}
	}
	v.Value, v.Truncated = mcptransport.NewBodyFromBase64(e.Value, limit, false)
	return *v
}

func (t *Toolset) decode(ctx context.Context, e *entities.KVEntry, detect bool) *entities.DecodeResult {
	data, err := base64.StdEncoding.DecodeString(e.Value)
	if err != nil || len(data) == 0 {
		return nil
	}
	return t.codec.DecodeSubject(ctx, e.Subject(), data, detect)
}

func (t *Toolset) detectsTypes(ctx context.Context) bool {
	cfg, err := t.settings.Get(ctx)
	return err != nil || cfg.DetectsTypes()
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
	limit := mcptransport.Limit(in.MaxValueBytes, defaultValueBytes, maxValueBytes)
	return nil, newEntryView(entry, limit, t.decode(ctx, entry, t.detectsTypes(ctx))), nil
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
	detect := t.detectsTypes(ctx)
	for i := len(entries) - 1; i >= len(entries)-count; i-- {
		out.Entries = append(out.Entries, newEntryView(&entries[i], limit, t.decode(ctx, &entries[i], detect)))
	}
	return nil, out, nil
}
