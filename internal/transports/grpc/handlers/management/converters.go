// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"fmt"
	"time"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	grpchelpers "github.com/dmit-4884/natscope/internal/transports/grpc/helpers"
	grpc_nats_management "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// protoCodecs bridges time.Time↔Timestamp and time.Duration↔Duration (converter
// type-gap).
var protoCodecs = grpchelpers.ProtoCodecs

// replicasMapping maps entity Replicas onto proto NumReplicas (KV/Object
// BucketInfo).
var replicasMapping = converter.WithFieldMappings(map[string]string{"Replicas": "NumReplicas"})

// replicasMappingToEntity maps proto NumReplicas onto entity Replicas (KV/Object
// BucketConfig).
var replicasMappingToEntity = converter.WithFieldMappings(map[string]string{"NumReplicas": "Replicas"})

// protoStreamSourceToEntity converts proto StreamSourceConfig;
// OptStartTime/SubjectTransforms/External handled manually (converter gaps).
// A malformed OptStartTime returns an error instead of being silently dropped
// — the source would otherwise replicate its entire history instead of the
// requested cutoff.
func protoStreamSourceToEntity(src *grpc_nats_management.StreamSourceConfig) (*entities.StreamSource, error) {
	if src == nil {
		return nil, nil //nolint:nilnil // nil input means "not configured", not an error
	}

	result := converter.Convert(src, &entities.StreamSource{},
		converter.WithIgnoreFields("OptStartTime", "SubjectTransforms", "External"),
	)

	if src.OptStartTime != nil {
		t, err := time.Parse(time.RFC3339, src.GetOptStartTime())
		if err != nil {
			return nil, &errs.NATSValidationError{
				Description: fmt.Sprintf("invalid opt_start_time %q: must be RFC3339", src.GetOptStartTime()),
				Cause:       err,
			}
		}
		result.OptStartTime = &t
	}

	for _, st := range src.GetSubjectTransforms() {
		result.SubjectTransforms = append(result.SubjectTransforms, converter.Convert(st, &entities.SubjectTransformConfig{}))
	}

	if src.GetExternal() != nil {
		result.External = converter.Convert(src.GetExternal(), &entities.ExternalStream{})
	}

	return result, nil
}

// protoStreamSourcesToEntity converts a repeated StreamSourceConfig, stopping
// at the first conversion error (see protoStreamSourceToEntity).
func protoStreamSourcesToEntity(sources []*grpc_nats_management.StreamSourceConfig) ([]*entities.StreamSource, error) {
	if len(sources) == 0 {
		return nil, nil
	}

	result := make([]*entities.StreamSource, 0, len(sources))
	for _, src := range sources {
		converted, err := protoStreamSourceToEntity(src)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

// streamSourceRefToEntity converts proto StreamSourceRef (KV mirror/sources);
// reused on create so KV shares Stream Mirror/Sources building blocks.
func streamSourceRefToEntity(src *natspb.StreamSourceRef) *entities.StreamSource {
	if src == nil {
		return nil
	}
	result := converter.Convert(src, &entities.StreamSource{},
		converter.WithIgnoreFields("External"),
	)
	if src.GetExternal() != nil {
		result.External = converter.Convert(src.GetExternal(), &entities.ExternalStream{})
	}
	return result
}
