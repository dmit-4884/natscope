// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/services/proto/registry"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	fwsvc "github.com/dmit-4884/natscope/internal/services/filewatcher"
	gitfetchersvc "github.com/dmit-4884/natscope/internal/services/gitfetcher"
	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	conflictsstorage "github.com/dmit-4884/natscope/internal/storages/proto/conflicts"
	descriptorsstorage "github.com/dmit-4884/natscope/internal/storages/proto/descriptors"
	selectionsstorage "github.com/dmit-4884/natscope/internal/storages/proto/selections"
	sourcesstorage "github.com/dmit-4884/natscope/internal/storages/proto/sources"
	versionsstorage "github.com/dmit-4884/natscope/internal/storages/proto/versions"
)

// ProtoReloadCallback runs synchronously inside notifyReload after a recompile
// — keep it cheap.
type ProtoReloadCallback func(messageCount int)

// Service implements proto.Service interface.
type Service struct {
	logger             *slog.Logger
	sourcesStorage     sourcesstorage.Storage
	versionsStorage    versionsstorage.Storage
	descriptorsStorage descriptorsstorage.Storage
	selectionsStorage  selectionsstorage.Storage
	conflictsStorage   conflictsstorage.Storage
	gitFetcher         gitfetchersvc.Service
	mappingsService    mappingssvc.Service
	fileWatcher        fwsvc.Service
	registryCache      *registry.Cache

	// compileLocks serializes compile->persist per source; zero-value usable
	// (lock() lazily inits) so intentionally NOT set in New().
	compileLocks sourceLocks

	reloadMu sync.RWMutex
	onReload ProtoReloadCallback
}

// New creates a new proto service.
func New(
	sourcesStorage sourcesstorage.Storage,
	versionsStorage versionsstorage.Storage,
	descriptorsStorage descriptorsstorage.Storage,
	selectionsStorage selectionsstorage.Storage,
	conflictsStorage conflictsstorage.Storage,
	gitFetcher gitfetchersvc.Service,
	mappingsService mappingssvc.Service,
	fileWatcher fwsvc.Service,
) *Service {
	s := &Service{
		logger:             slog.Default().With(slogx.Module("service:proto")),
		sourcesStorage:     sourcesStorage,
		versionsStorage:    versionsStorage,
		descriptorsStorage: descriptorsStorage,
		selectionsStorage:  selectionsStorage,
		conflictsStorage:   conflictsStorage,
		gitFetcher:         gitFetcher,
		mappingsService:    mappingsService,
		fileWatcher:        fileWatcher,
		registryCache:      registry.NewCache(descriptorsStorage),
	}

	if fileWatcher != nil {
		fileWatcher.SetCallback(s.onFileWatcherChange)
	}

	return s
}

// SetOnReloadCallback registers a callback invoked after proto recompilation.
func (s *Service) SetOnReloadCallback(cb ProtoReloadCallback) {
	s.reloadMu.Lock()
	s.onReload = cb
	s.reloadMu.Unlock()
}

// notifyReload refreshes the conflict report and fires the reload callback with
// the current message count; called after every recompile/filewatcher reload.
func (s *Service) notifyReload(ctx context.Context) {
	s.recomputeConflicts(ctx)

	s.reloadMu.RLock()
	cb := s.onReload
	s.reloadMu.RUnlock()
	if cb == nil {
		return
	}
	msgs := s.ListMessages(ctx)
	cb(len(msgs))
}

// recomputeConflicts runs MergeWithReport and overwrites conflicts storage;
// best-effort — errors are logged, not returned, so decode is never blocked.
func (s *Service) recomputeConflicts(ctx context.Context) {
	if s.conflictsStorage == nil {
		return
	}
	snaps := s.activeSnapshots(ctx)
	if len(snaps) < minActiveSnapshotsForConflictReport {
		// Single source can't conflict with itself at the cross-source level.
		//nolint:errcheck // best-effort cleanup of stale conflicts; persistence failure is non-fatal
		_ = s.conflictsStorage.ReplaceAll(ctx, nil)
		return
	}

	inputs := make([]protoutils.SchemaInput, 0, len(snaps))
	for _, snap := range snaps {
		if snap.Descriptor == nil || len(snap.Descriptor.DescriptorSet) == 0 {
			continue
		}
		inputs = append(inputs, protoutils.SchemaInput{
			SourceID: snap.SourceID,
			Tag:      snap.Tag,
			Bytes:    snap.Descriptor.DescriptorSet,
		})
	}
	report, err := protoutils.MergeWithReport(inputs)
	if err != nil {
		s.logger.WarnContext(ctx, "recompute conflicts: merge failed", slogx.Error(err))
		return
	}
	if err := s.conflictsStorage.ReplaceAll(ctx, report.Conflicts); err != nil {
		s.logger.WarnContext(ctx, "recompute conflicts: persist failed", slogx.Error(err))
	}
}

// maxStoredDiagnostics bounds the diagnostics head persisted on LastCompile so
// storage doesn't balloon (full lists stay in RPC responses).
const maxStoredDiagnostics = 50

// recordCompile persists compile telemetry onto the source's LastCompile,
// using Update directly to skip BeforeUpdate so UpdatedAt/etags don't move.
func (s *Service) recordCompile(
	ctx context.Context, sourceID string, ok bool, errMsg string,
	messageCount, fileCount int,
	diags []entities.CompileDiagnostic, roots []string, origin string,
) {
	source, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		// Source vanished mid-flight (race with delete) — nothing to write to.
		return
	}
	if len(diags) > maxStoredDiagnostics {
		diags = diags[:maxStoredDiagnostics]
	}
	result := &entities.ProtoCompileResult{
		At:           time.Now().UnixMilli(),
		Ok:           ok,
		MessageCount: int32(messageCount), //nolint:gosec // counts are bounded by descriptor size
		FileCount:    int32(fileCount),    //nolint:gosec
		Diagnostics:  diags,
		Roots:        roots,
		RootsOrigin:  origin,
	}
	if !ok && errMsg != "" {
		result.Error = &errMsg
	}
	source.LastCompile = result
	if err := s.sourcesStorage.Update(ctx, source); err != nil {
		s.logger.WarnContext(ctx, "failed to persist compile telemetry",
			slog.String("source_id", sourceID),
			slogx.Error(err))
	}
}

// Compile-time checks that Service satisfies every segregated proto role.
var (
	_ protosvc.Registry         = (*Service)(nil)
	_ protosvc.Codec            = (*Service)(nil)
	_ protosvc.SourceManager    = (*Service)(nil)
	_ protosvc.SelectionManager = (*Service)(nil)
)
