// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"log/slog"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
	"github.com/dmit-4884/natscope/internal/services/proto/registry"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

// liveDecoder is a stateful, per-client decoder for live streams; Reset()
// clears caches so the next Decode() picks up fresh mappings after a reload.
type liveDecoder struct {
	service  *Service
	resolver *natsutil.MappingResolver
	detect   bool

	// Per-client snapshot cache keyed by mappingSnapshotKey; resolved lazily.
	snapshotsByKey map[string]*registry.Snapshot

	ready bool
}

var _ protosvc.LiveDecoder = (*liveDecoder)(nil)

func (d *liveDecoder) Reset() {
	d.ready = false
	d.snapshotsByKey = nil
	d.resolver = nil
}

func (d *liveDecoder) Init(ctx context.Context) {
	if d.service == nil || d.service.mappingsService == nil {
		return
	}
	d.resolver = d.service.mappingsService.Resolver(ctx)
	d.snapshotsByKey = make(map[string]*registry.Snapshot)
	d.ready = true
}

func (d *liveDecoder) Ready() bool {
	return d.ready
}

func (d *liveDecoder) Decode(ctx context.Context, data []byte, subject string) *entities.DecodeResult {
	if !d.ready || d.resolver == nil {
		return nil
	}

	m := d.resolver.Resolve(subject)
	if m == nil {
		if d.detect && entities.DetectContentType(data) == entities.ContentTypeBinary {
			return d.service.autoDecode(ctx, subject, data)
		}
		return nil
	}

	snap, err := d.snapshotFor(ctx, m)
	if err != nil {
		return &entities.DecodeResult{Error: err.Error()}
	}
	result := decodeWithSnapshot(snap, data, m.MessageType, m.Framing)
	result.MessageType = m.MessageType
	return result
}

// snapshotFor returns the snapshot the mapping resolves to (pin-aware),
// populating the per-client cache on first miss.
func (d *liveDecoder) snapshotFor(ctx context.Context, m *entities.SubjectMapping) (*registry.Snapshot, error) {
	key := mappingSnapshotKey(m)
	if snap, ok := d.snapshotsByKey[key]; ok {
		return snap, nil
	}
	snap, err := d.service.resolveDescriptorForMapping(ctx, m)
	if err != nil {
		slog.Debug("live: snapshot unavailable",
			slog.String("source_id", m.SourceID),
			slogx.Error(err))
		return nil, err
	}
	d.snapshotsByKey[key] = snap
	return snap, nil
}

// mappingSnapshotKey keys the snapshot cache by source plus pin, so differently
// pinned mappings on one source don't collide.
func mappingSnapshotKey(m *entities.SubjectMapping) string {
	var pinFP string
	if m.PinnedFingerprint != nil {
		pinFP = *m.PinnedFingerprint
	}
	return m.SourceID + "\x00" + pinFP
}
