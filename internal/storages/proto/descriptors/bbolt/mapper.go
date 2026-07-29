// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import (
	"github.com/altessa-s/go-atlas/core/encoding/hash"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
)

// toDoc converts the entity to its document and stamps the content fingerprint
// (sha256 of the descriptor set) used for pinned-snapshot lookup.
func toDoc(in *entities.ProtoDescriptor) *descriptorDoc {
	d := converter.Convert(in, &descriptorDoc{}, bbstore.Opts()...)
	d.Fingerprint = fingerprint(in.DescriptorSet)
	return d
}

// fingerprint is the sha256 hex of the descriptor set, or "" when empty.
func fingerprint(set []byte) string {
	if len(set) == 0 {
		return ""
	}
	return hash.SHA256HexBytes(set)
}
