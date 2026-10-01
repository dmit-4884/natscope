// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"

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
	// Decode decodes a payload by its subject; nil when nothing applies.
	Decode(ctx context.Context, data []byte, subject string) *entities.DecodeResult
}

// Registry exposes read-only inspection of loaded proto message types,
// examples, stats, and schema conflicts.
type Registry interface {
	// ListTypes returns the messages, enums and services of one source, or of every enabled source when sourceID is empty.
	ListTypes(ctx context.Context, sourceID string) ([]entities.SchemaType, error)

	// DescribeType describes a type of a source, with every type it reaches when reachable is set;
	// ErrProtoTypeNotFound if absent.
	DescribeType(ctx context.Context, sourceID, fingerprint, fullName string, reachable bool) (*entities.TypeDescription, error)

	// GenerateExample builds an example JSON object for a message type;
	// ErrProtoMessageNotFound if absent.
	GenerateExample(ctx context.Context, sourceID, fingerprint, messageType string) (any, error)

	// MappingHealth returns resolvability state for the given mapping ids, in
	// input order.
	MappingHealth(ctx context.Context, ids []string) ([]entities.SubjectMappingHealth, error)

	// SchemaStatus counts the loaded message types and lists clashes between enabled sources.
	SchemaStatus(ctx context.Context) (*entities.SchemaStatus, error)
}

// Codec encodes, decodes, and validates protobuf payloads against pinned
// snapshots.
type Codec interface {
	// Decode decodes via the schema picked by req.SourceID and req.Fingerprint.
	Decode(ctx context.Context, req entities.CodecRequest) (*entities.DecodeResult, error)

	// DecodeWire reads a payload without a schema.
	DecodeWire(data []byte) *entities.WireDump

	// DecodeForMapping decodes against the mapping's bound source — safest path,
	// no manual source pick.
	DecodeForMapping(ctx context.Context, data []byte, m *entities.SubjectMapping) (*entities.DecodeResult, error)

	// DecodeSubject decodes a payload published on subject through its mapping or, with detect, a detected type;
	// nil when neither applies.
	DecodeSubject(ctx context.Context, subject string, data []byte, detect bool) *entities.DecodeResult

	// DecodeMessages batch-decodes messages, grouping by resolved snapshot; detect auto-detects unmapped types.
	DecodeMessages(ctx context.Context, messages []*entities.Message, detect bool)

	// DetectTypes ranks the message types of one source, or of every enabled source, by how well data decodes as each.
	DetectTypes(ctx context.Context, data []byte, sourceID string, limit int) ([]entities.TypeCandidate, error)

	// NewLiveDecoder creates a stateful decoder for live streams with lazy-init
	// and reload; detect auto-detects unmapped types.
	NewLiveDecoder(detect bool) LiveDecoder

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
	ValidateRepository(
		ctx context.Context,
		sourceType entities.SourceType,
		repository string,
		token *string,
	) (*entities.RepositoryValidation, error)

	// ValidateLocalPath probes a filesystem root; same contract as
	// ValidateRepository.
	ValidateLocalPath(ctx context.Context, dirPath string) (*entities.LocalPathValidation, error)

	// ListRefs returns the tags and branches of a Git source, or the labels of a BSR source.
	ListRefs(ctx context.Context, sourceID string) ([]entities.ProtoRef, error)

	// SelectRef points a Git or BSR source at a ref; compile errors come back in the outcome.
	SelectRef(ctx context.Context, sourceID, ref string) (*entities.ProtoSource, *entities.CompileOutcome, error)

	// RefreshSource rebuilds the active schema: re-resolves the selected ref or recompiles the files.
	RefreshSource(ctx context.Context, sourceID string) (*entities.ProtoSource, *entities.CompileOutcome, error)

	// ListRevisions returns the stored schemas of a source, newest first.
	ListRevisions(ctx context.Context, sourceID string) ([]entities.SchemaRevision, error)

	// SetEnabled enables or disables a source.
	SetEnabled(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error)

	// SetWatcher enables or disables file watcher for a local directory source.
	SetWatcher(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error)

	// UploadSchema makes uploaded .proto files, or a compiled descriptor set, the active schema of an upload source;
	// compile errors come back in the outcome.
	UploadSchema(ctx context.Context, sourceID string, upload entities.SchemaUpload) (*entities.ProtoSource, *entities.CompileOutcome, error)

	// UploadedSchema returns what was uploaded for the active schema of an upload source.
	UploadedSchema(ctx context.Context, sourceID string) (*entities.SchemaUpload, error)
}
