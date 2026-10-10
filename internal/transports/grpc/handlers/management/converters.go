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

// protoStreamSourceToEntity converts proto StreamSourceConfig; OptStartTime is parsed manually
// (RFC3339 string). A malformed OptStartTime is an error.
func protoStreamSourceToEntity(src *grpc_nats_management.StreamSourceConfig) (*entities.StreamSource, error) {
	if src == nil {
		return nil, nil //nolint:nilnil // nil means not configured
	}

	result := converter.Convert(src, &entities.StreamSource{},
		converter.WithIgnoreFields("OptStartTime"),
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

	return result, nil
}

// protoStreamSourcesToEntity converts a repeated StreamSourceConfig, stopping at the first error.
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
