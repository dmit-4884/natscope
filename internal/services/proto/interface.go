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
	// ListTypes returns the messages, enums and services of one source, or of every enabled source when sourceID is empty.
	ListTypes(ctx context.Context, sourceID string) ([]entities.SchemaType, error)

	// DescribeType describes a type of a source, with every type it reaches when reachable is set;
	// ErrProtoTypeNotFound if absent.
	DescribeType(ctx context.Context, sourceID, fullName string, reachable bool) (*entities.TypeDescription, error)

	// GenerateExample builds an example JSON object for a message type;
	// ErrProtoMessageNotFound if absent.
	GenerateExample(ctx context.Context, sourceID, messageType string) (any, error)

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
	// Decode decodes via the schema picked by req.SourceID and req.Fingerprint.
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

	// ListRefs returns the tags and branches of a git source.
	ListRefs(ctx context.Context, sourceID string) ([]entities.ProtoRef, error)

	// SelectRef points a git source at a tag, branch or commit; compile errors come back in the outcome.
	SelectRef(ctx context.Context, sourceID, ref string) (*entities.ProtoSource, *entities.CompileOutcome, error)

	// RefreshSource rebuilds the active schema: re-resolves the git ref or recompiles local files.
	RefreshSource(ctx context.Context, sourceID string) (*entities.ProtoSource, *entities.CompileOutcome, error)

	// ListRevisions returns the stored schemas of a source, newest first.
	ListRevisions(ctx context.Context, sourceID string) ([]entities.SchemaRevision, error)

	// SetEnabled enables or disables a source.
	SetEnabled(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error)

	// SetWatcher enables or disables file watcher for a local directory source.
	SetWatcher(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error)

	// ValidateFiles compiles a Files-type source (or inline files/includeDirs when
	// sourceID empty) without persisting.
	ValidateFiles(ctx context.Context, sourceID *string, files []string, includeDirs []string) (*entities.CompileOutcome, error)
}
