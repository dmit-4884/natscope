// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"encoding/json"

	"github.com/dmit-4884/natscope/internal/entities"
)

// LiveDecoder is a stateful decoder for live message streams with lazy-init and
// reload.
type LiveDecoder interface {
	// Reset marks the decoder as needing re-initialization (e.g. after proto
	// reload).
	Reset()
	Init(ctx context.Context)
	Ready() bool
	Decode(
		ctx context.Context,
		data []byte,
		subject string,
	) (decoded json.RawMessage, decodedType string, decodeError string)
}

// Registry exposes read-only inspection of loaded proto message types,
// examples, stats, and schema conflicts.
type Registry interface {
	// ListMessages lists all messages from active snapshots; each entry carries
	// SourceID + SourceTag.
	ListMessages(ctx context.Context) []entities.ProtoMessageInfo

	// GetMessage returns message-type detail within a source;
	// ErrProtoMessageNotFound if absent.
	GetMessage(ctx context.Context, sourceID, messageType string) (*entities.ProtoMessageInfo, error)

	// GenerateExample builds an example JSON object for a message type;
	// ErrProtoMessageNotFound if absent.
	GenerateExample(ctx context.Context, sourceID, messageType string) (map[string]interface{}, error)

	// Stats returns aggregated statistics about loaded proto descriptors across
	// active selections.
	Stats(ctx context.Context) *entities.ProtoStats

	// MappingHealth returns resolvability state for the given mapping ids, in
	// input order.
	MappingHealth(ctx context.Context, ids []string) ([]entities.SubjectMappingHealth, error)

	// ListSchemaConflicts returns cross-source conflicts from the last
	// compile/reload; empty overwrites prior state.
	ListSchemaConflicts(ctx context.Context) (entities.SchemaConflicts, error)
}

// Codec encodes, decodes, and validates protobuf payloads against pinned
// snapshots.
type Codec interface {
	// Decode decodes via the snapshot from req.SourceID+req.Tag; SourceID
	// required, empty Tag uses active selection.
	Decode(ctx context.Context, req entities.CodecRequest) (*entities.DecodeResult, error)

	// DecodeForMapping decodes against the mapping's bound source — safest path,
	// no manual source pick.
	DecodeForMapping(ctx context.Context, data []byte, m *entities.SubjectMapping) (*entities.DecodeResult, error)

	// DecodeMessages batch-decodes messages, grouping by resolved snapshot.
	DecodeMessages(ctx context.Context, messages []*entities.Message)

	// NewLiveDecoder creates a stateful decoder for live streams with lazy-init
	// and reload.
	NewLiveDecoder() LiveDecoder

	// Encode converts JSON data to protobuf binary format using the resolved
	// snapshot.
	Encode(ctx context.Context, req entities.CodecRequest) (*entities.EncodeResult, error)

	// EncodeRaw converts JSON data to raw protobuf bytes using the resolved
	// snapshot.
	EncodeRaw(ctx context.Context, req entities.CodecRequest) ([]byte, error)

	// Validate validates base64-encoded protobuf data against buf.validate rules
	// within a snapshot.
	Validate(ctx context.Context, dataBase64 string, req entities.CodecRequest) (*entities.ValidationResult, error)

	// EncodeWithValidation encodes JSON to protobuf and validates the result.
	EncodeWithValidation(
		ctx context.Context,
		req entities.CodecRequest,
	) (*entities.EncodeResult, []*entities.ValidationViolation, error)

	// ValidateJSON is the one-shot encode+validate pipeline;
	// encode/snapshot/message-type errors surface in the result, not the error.
	ValidateJSON(ctx context.Context, req entities.CodecRequest) (*entities.ValidationResult, error)
}

// SourceManager handles proto-source CRUD, validation, fetching, and
// compilation.
type SourceManager interface {
	// CreateSource creates a proto source; ErrProtoSourceNameAlreadyInUse if name
	// is taken.
	CreateSource(ctx context.Context, in *entities.ProtoSourceCreate) (*entities.ProtoSource, error)

	// GetSource retrieves a source by Id; ErrProtoSourceNotFound if absent.
	GetSource(ctx context.Context, id string) (*entities.ProtoSource, error)

	// ListSources returns sources for a user with pagination.
	ListSources(ctx context.Context, in *entities.ProtoSourcesList) (*entities.List[entities.ProtoSources], error)

	// UpdateSource updates a proto source; ErrProtoSourceNotFound if absent.
	UpdateSource(ctx context.Context, in *entities.ProtoSourceUpdate) (*entities.ProtoSource, error)

	// DeleteSource soft-deletes a proto source; ErrProtoSourceNotFound if absent.
	DeleteSource(ctx context.Context, id string) error

	// ValidateRepository probes a Git repo; outcome is always in the result, error
	// only on unexpected internal failure.
	ValidateRepository(ctx context.Context, repository string, token *string) (*entities.RepositoryValidation, error)

	// ValidateLocalPath probes a filesystem root; same contract as
	// ValidateRepository.
	ValidateLocalPath(ctx context.Context, dirPath string) (*entities.LocalPathValidation, error)

	// ListTags returns available tags from a source's Git repository.
	ListTags(ctx context.Context, sourceID string) ([]string, error)

	// FetchVersion fetches proto files for a specific tag and stores them.
	FetchVersion(ctx context.Context, sourceID, tag string) (*entities.ProtoVersion, error)

	// GetVersion retrieves a stored version by source Id and tag.
	GetVersion(ctx context.Context, sourceID, tag string) (*entities.ProtoVersion, error)

	// FetchAndCompile fetches proto files, compiles them to descriptors, and
	// stores them.
	FetchAndCompile(ctx context.Context, sourceID, tag string) (*entities.ProtoDescriptor, error)

	// SetEnabled enables or disables a source.
	SetEnabled(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error)

	// SetWatcher enables or disables file watcher for a local directory source.
	SetWatcher(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error)

	// CompileLocal compiles a local dir source; compile errors come back as
	// diagnostics with nil error.
	CompileLocal(ctx context.Context, sourceID string) (*entities.CompileResult, []entities.CompileDiagnostic, error)

	// ValidateFiles compiles a Files-type source (or inline files/includeDirs when
	// sourceID empty) without persisting.
	ValidateFiles(
		ctx context.Context,
		sourceID *string,
		files []string,
		includeDirs []string,
	) (*entities.CompileResult, []entities.CompileDiagnostic, error)

	// CompileFiles compiles a Files-type source and persists; on error nothing is
	// persisted, prior descriptor stays.
	CompileFiles(ctx context.Context, sourceID string) (*entities.CompileResult, []entities.CompileDiagnostic, error)
}

// SelectionManager manages the per-user selection of proto versions.
type SelectionManager interface {
	// Select selects a proto version for a user (creates or updates).
	Select(ctx context.Context, in *entities.ProtoSelectionCreate) (*entities.ProtoSelection, error)

	// ListSelections returns all selections.
	ListSelections(ctx context.Context) (entities.ProtoSelections, error)

	// LoadAllSelections fetch+compiles every stored selection; per-selection
	// failures are logged, never abort.
	LoadAllSelections(ctx context.Context) (*entities.ProtoLoadResult, error)

	// DeleteSelection removes a selection.
	DeleteSelection(ctx context.Context, selectionID string) error
}
