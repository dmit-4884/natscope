// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package grpchelpers

import (
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/converter/codec/durpb"
	"github.com/altessa-s/go-atlas/domain/converter/codec/tspb"
)

// ProtoCodecs bridges converter gaps: time.Time<->tspb, time.Duration<->durpb.
// WithIgnoreZero keeps zero values nil per proto3 presence semantics.
var ProtoCodecs = converter.WithCodecs(
	tspb.New(tspb.WithIgnoreZero()),
	durpb.New(durpb.WithIgnoreZero()),
)
