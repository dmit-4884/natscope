// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package grpchelpers

import (
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/converter/codec/durpb"
	"github.com/altessa-s/go-atlas/domain/converter/codec/tspb"

	"github.com/dmit-4884/natscope/internal/pkg/convcodecs"
)

// ProtoCodecs bridges converter gaps: time.Time<->tspb, time.Duration<->durpb.
// WithIgnoreZero keeps zero values nil per proto3 presence semantics.
// convcodecs.DurationSaturating runs before durpb so Duration overflow saturates.
var ProtoCodecs = converter.WithCodecs(
	tspb.New(tspb.WithIgnoreZero()),
	convcodecs.DurationSaturating,
	durpb.New(durpb.WithIgnoreZero()),
)
