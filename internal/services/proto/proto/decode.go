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

// Decode decodes protobuf data with the schema picked by req.SourceID and req.Fingerprint.
func (s *Service) Decode(ctx context.Context, req entities.CodecRequest) (*entities.DecodeResult, error) {
	snap, err := s.snapshotForRequest(ctx, req)
	if err != nil {
		return &entities.DecodeResult{Success: false, Error: snapshotError(req.SourceID, err).Error()}, nil
	}

	result := decodeWithSnapshot(snap, req.Data, req.MessageType, req.Framing)

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

// DecodeWire reads a payload without a schema.
func (s *Service) DecodeWire(data []byte) *entities.WireDump {
	fields, valid, err := protoutils.DecodeWire(data)
	dump := &entities.WireDump{Fields: fields, ValidBytes: valid}
	if err != nil {
		dump.Error = err.Error()
	}
	return dump
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
	return decodeWithSnapshot(snap, data, m.MessageType, m.Framing), nil
}

// decodeWithSnapshot decodes a single protobuf payload against a parsed
// snapshot.
func decodeWithSnapshot(snap *registry.Snapshot, data []byte, messageType string, framing entities.Framing) *entities.DecodeResult {
	if snap == nil {
		return &entities.DecodeResult{Success: false, Error: "snapshot unavailable"}
	}
	md, ok := snap.Schema.Message(messageType)
	if !ok {
		return &entities.DecodeResult{
			Success: false,
			Error:   fmt.Sprintf("Proto type '%s' not found in source '%s' (revision '%s').", messageType, snap.SourceID, snap.Revision),
		}
	}
	msg, offset, err := protoutils.Unframe(data, framing)
	if err != nil {
		return &entities.DecodeResult{Success: false, Error: fmt.Sprintf("Cannot unwrap the %s framing: %v", framing.Kind, err)}
	}
	result := decodeWithDescriptor(snap.Schema, md, msg, messageType, framing.Kind == entities.FramingNone)
	if result.ValidBytes > 0 {
		result.ValidBytes += offset
	}
	return result
}

// decodeWithDescriptor decodes a single payload against a MessageDescriptor;
// the underlying protobuf error is surfaced verbatim for diagnosis.
func decodeWithDescriptor(
	schema *protoutils.Schema,
	md protoreflect.MessageDescriptor,
	data []byte,
	messageType string,
	hintFraming bool,
) *entities.DecodeResult {
	msg := dynamicpb.NewMessage(md)
	if unmarshalErr := schema.ParseBinary(data, msg); unmarshalErr != nil {
		hint := ""
		if hintFraming {
			hint = protoutils.FramingHint(data)
		}
		result := &entities.DecodeResult{
			Success: false,
			Error:   fmt.Sprintf("Cannot decode message as %q: %s%s", messageType, unmarshalErr.Error(), hint),
		}
		result.Decoded, result.ValidBytes = decodePrefix(schema, md, data)
		return result
	}

	rawJSON, err := schema.RenderJSON(msg)
	if err != nil {
		return &entities.DecodeResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to convert to JSON: %v", err),
		}
	}

	return &entities.DecodeResult{
		Success:       true,
		Decoded:       rawJSON,
		UnknownFields: protoutils.UnknownFields(msg),
	}
}

func decodePrefix(schema *protoutils.Schema, md protoreflect.MessageDescriptor, data []byte) (json.RawMessage, int) {
	_, valid, _ := protoutils.DecodeWire(data) //nolint:errcheck // the error marks where the valid prefix ends
	if valid == 0 || valid == len(data) {
		return nil, 0
	}
	msg := dynamicpb.NewMessage(md)
	if schema.ParseBinary(data[:valid], msg) != nil {
		return nil, 0
	}
	rawJSON, err := schema.RenderJSON(msg)
	if err != nil {
		return nil, 0
	}
	return rawJSON, valid
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
		snap     *registry.Snapshot
		idxs     []int
		mts      []string
		framings []entities.Framing
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
		key := snap.SourceID + "\x00" + snap.Revision
		g := groups[key]
		if g == nil {
			g = &group{snap: snap}
			groups[key] = g
		}
		g.idxs = append(g.idxs, i)
		g.mts = append(g.mts, m.MessageType)
		g.framings = append(g.framings, m.Framing)
	}

	t1 := time.Now()

	for _, g := range groups {
		items, decErrs := messagesToBatch(messages, g.idxs, g.mts, g.framings)
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
			messages[idx].DecodeError = r.Error
			if r.Success || r.ValidBytes > 0 {
				messages[idx].Decoded = r.Decoded
				messages[idx].DecodedType = g.mts[j]
				messages[idx].DecodedUnknownFields = len(r.UnknownFields)
				messages[idx].DecodedValidBytes = r.ValidBytes
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
func messagesToBatch(
	messages []*entities.Message,
	idxs []int,
	mts []string,
	framings []entities.Framing,
) ([]entities.BatchDecodeItem, []string) {
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
			Framing:     framings[j],
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
		results[i] = decodeWithSnapshot(snap, item.Data, item.MessageType, item.Framing)
		return nil
	}, concurrency.WithConcurrency[int](workers))

	return results
}
