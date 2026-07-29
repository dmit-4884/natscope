// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

import (
	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/converter/codec/unixtime"
)

// Opts are the converter options mapping entity <-> persistence doc: embedded
// structs flatten (BaseEntity to Base), time.Time maps to unix milliseconds.
func Opts() []converter.Option {
	return []converter.Option{
		converter.WithHandleEmbeddedStructs(true),
		converter.WithCodecs(unixtime.New(unixtime.WithMilliseconds())),
	}
}

// ToEntity converts one persistence doc to a domain entity of type E using Opts.
func ToEntity[E, D any](doc D) *E {
	return converter.Convert(doc, new(E), Opts()...)
}

// ToEntities converts a slice of persistence docs to domain entities of type E
// using Opts.
func ToEntities[E, D any](docs []D) []*E {
	return slices.To(docs, ToEntity[E, D])
}
