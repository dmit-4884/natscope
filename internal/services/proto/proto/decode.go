// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/altessa-s/go-atlas/core/runtime/concurrency"
	"github.com/altessa-s/go-atlas/core/runtime/panics"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/services/proto/registry"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Tunables for the per-snapshot batch decode path: sequential below
// decodeParallelFrom, else worker pool sized by optimalDecodeConcurrency.
const (
	decodeConcurrencyMin = 8
	decodeConcurrencyMax = 32
	decodeParallelFrom   = 4 // parallelize sooner; decode is CPU-heavy even for small batches
)

// optimalDecodeConcurrency returns the parallel-decode worker count: NumCPU
// clamped (decode is CPU-bound).
func optimalDecodeConcurrency() int {
	n := runtime.NumCPU()
	if n < decodeConcurrencyMin {
		return decodeConcurrencyMin
	}
	if n > decodeConcurrencyMax {
		return decodeConcurrencyMax
	}
	return n
}

// Decode decodes protobuf data using the snapshot resolved from req.SourceID +
// req.Tag.
func (s *Service) Decode(ctx context.Context, req entities.CodecRequest) (*entities.DecodeResult, error) {
	snap, err := s.snapshotForRequest(ctx, req)
	if err != nil {
		return &entities.DecodeResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to resolve snapshot: %v", err),
		}, nil
	}

	result := decodeWithSnapshot(snap, req.Data, req.MessageType)

	// Add formatted JSON for single-message decode (used by codec UI).
	if result.Success && len(result.Decoded) > 0 {
		var pretty interface{}
		if jsonErr := json.Unmarshal(result.Decoded, &pretty); jsonErr == nil {
			if formatted, formatErr := json.MarshalIndent(pretty, "", "  "); formatErr == nil {
				result.FormattedJSON = string(formatted)
			}
		}
	}

	return result, nil
}

// DecodeForMapping decodes a payload for the source the mapping is bound to.
func (s *Service) DecodeForMapping(
	ctx context.Context,
	data []byte,
	m *entities.SubjectMapping,
) (*entities.DecodeResult, error) {
	snap, err := s.resolveDescriptorForMapping(ctx, m)
	if err != nil {
		return nil, err
	}
	return decodeWithSnapshot(snap, data, m.MessageType), nil
}

// decodeWithSnapshot decodes a single protobuf payload against a parsed
// snapshot.
func decodeWithSnapshot(snap *registry.Snapshot, data []byte, messageType string) *entities.DecodeResult {
	if snap == nil {
		return &entities.DecodeResult{Success: false, Error: "snapshot unavailable"}
	}
	md, ok := snap.Messages[messageType]
	if !ok {
		return &entities.DecodeResult{
			Success: false,
			Error:   fmt.Sprintf("Proto type '%s' not found in source '%s' (tag '%s').", messageType, snap.SourceID, snap.Tag),
		}
	}
	return decodeWithDescriptor(md, data, messageType)
}

// decodeWithDescriptor decodes a single payload against a MessageDescriptor;
// the underlying protobuf error is surfaced verbatim for diagnosis.
func decodeWithDescriptor(md protoreflect.MessageDescriptor, data []byte, messageType string) *entities.DecodeResult {
	msg := dynamicpb.NewMessage(md)
	if unmarshalErr := proto.Unmarshal(data, msg); unmarshalErr != nil {
		return &entities.DecodeResult{
			Success: false,
			Error: fmt.Sprintf(
				"Cannot decode message as %q: %s%s",
				messageType, unmarshalErr.Error(), protoutils.FramingHint(data),
			),
		}
	}

	rawJSON, err := protojson.MarshalOptions{
		UseProtoNames:   true, // snake_case as in proto file
		EmitUnpopulated: true, // include zero-valued fields so UI debugger shows full structure
	}.Marshal(msg)
	if err != nil {
		return &entities.DecodeResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to convert to JSON: %v", err),
		}
	}

	return &entities.DecodeResult{
		Success: true,
		Decoded: rawJSON,
	}
}

// DecodeMessages decodes a slice of messages: resolve each mapping →
// source/tag, group by snapshot, decode per-group.
func (s *Service) DecodeMessages(ctx context.Context, messages []*entities.Message) {
	if len(messages) == 0 {
		return
	}

	t0 := time.Now()

	resolver := s.mappingsService.Resolver(ctx)
	if resolver == nil {
		return
	}

	// Per-snapshot grouping. snapKey = sourceID + tag.
	type group struct {
		snap *registry.Snapshot
		idxs []int
		mts  []string
	}
	groups := make(map[string]*group)

	for i, msg := range messages {
		m := resolver.Resolve(msg.Subject)
		if m == nil {
			continue
		}
		snap, err := s.resolveDescriptorForMapping(ctx, m)
		if err != nil {
			messages[i].DecodeError = err.Error()
			continue
		}
		key := snap.SourceID + "\x00" + snap.Tag
		g := groups[key]
		if g == nil {
			g = &group{snap: snap}
			groups[key] = g
		}
		g.idxs = append(g.idxs, i)
		g.mts = append(g.mts, m.MessageType)
	}

	t1 := time.Now()

	for _, g := range groups {
		items, decErrs := messagesToBatch(messages, g.idxs, g.mts)
		results := decodeBatchWithSnapshot(ctx, g.snap, items)
		for j, idx := range g.idxs {
			if decErrs[j] != "" {
				messages[idx].DecodeError = decErrs[j]
				continue
			}
			r := results[j]
			if r == nil {
				continue
			}
			if r.Success {
				messages[idx].Decoded = r.Decoded
				messages[idx].DecodedType = g.mts[j]
			} else if r.Error != "" {
				messages[idx].DecodeError = r.Error
			}
		}
	}

	s.logger.DebugContext(ctx, "DecodeMessages completed",
		slog.Duration("resolve", t1.Sub(t0)),
		slog.Duration("decode", time.Since(t1)),
		slog.Duration("total", time.Since(t0)),
		slog.Int("groups", len(groups)),
		slog.Int("messages", len(messages)))
}

// messagesToBatch builds a BatchDecodeItem slice from selected message indices,
// plus a per-index decode error (empty when none) carrying the base64 failure
// for payloads that fail to decode.
func messagesToBatch(messages []*entities.Message, idxs []int, mts []string) ([]entities.BatchDecodeItem, []string) {
	items := make([]entities.BatchDecodeItem, len(idxs))
	decErrs := make([]string, len(idxs))
	for j, idx := range idxs {
		data, decErr := base64.StdEncoding.DecodeString(messages[idx].DataBase64)
		if decErr != nil {
			decErrs[j] = fmt.Sprintf("Invalid base64 payload: %v", decErr)
			continue
		}
		items[j] = entities.BatchDecodeItem{
			Data:        data,
			MessageType: mts[j],
		}
	}
	return items, decErrs
}

// decodeBatchWithSnapshot decodes a batch against one snapshot; sequential for
// small batches, bounded-concurrency parallel for larger.
func decodeBatchWithSnapshot(
	ctx context.Context,
	snap *registry.Snapshot,
	items []entities.BatchDecodeItem,
) []*entities.DecodeResult {
	results := make([]*entities.DecodeResult, len(items))
	if len(items) == 0 {
		return results
	}

	workers := 1
	if len(items) >= decodeParallelFrom {
		workers = optimalDecodeConcurrency()
	}

	indices := make([]int, len(items))
	for i := range items {
		indices[i] = i
	}

	// Best-effort: fn never errors; a canceled ctx just leaves later results
	// nil, which the caller skips.
	//nolint:errcheck // returned error carries no actionable info here
	_ = concurrency.Process(ctx, indices, func(ctx context.Context, i int) error {
		defer panics.Handle(ctx)
		item := items[i]
		if item.MessageType == "" {
			results[i] = &entities.DecodeResult{Success: false, Error: "Missing message type"}
			return nil
		}
		results[i] = decodeWithSnapshot(snap, item.Data, item.MessageType)
		return nil
	}, concurrency.WithConcurrency[int](workers))

	return results
}
