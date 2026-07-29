// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"time"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

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

// protoStreamSourceToEntity converts proto StreamSourceConfig;
// OptStartTime/SubjectTransforms/External handled manually (converter gaps).
func protoStreamSourceToEntity(src *grpc_nats_management.StreamSourceConfig) *entities.StreamSource {
	if src == nil {
		return nil
	}

	result := converter.Convert(src, &entities.StreamSource{},
		converter.WithIgnoreFields("OptStartTime", "SubjectTransforms", "External"),
	)

	if src.OptStartTime != nil {
		if t, err := time.Parse(time.RFC3339, src.GetOptStartTime()); err == nil {
			result.OptStartTime = &t
		}
	}

	for _, st := range src.GetSubjectTransforms() {
		result.SubjectTransforms = append(result.SubjectTransforms, converter.Convert(st, &entities.SubjectTransformConfig{}))
	}

	if src.GetExternal() != nil {
		result.External = converter.Convert(src.GetExternal(), &entities.ExternalStream{})
	}

	return result
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
