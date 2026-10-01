// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"bytes"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// SchemaInput is the descriptor set of one proto source revision.
type SchemaInput struct {
	SourceID string
	Revision string
	Bytes    []byte
}

type owner struct {
	ref   entities.SchemaRef
	shape []byte
}

type namedShape struct {
	name  string
	shape []byte
}

var conflictReasons = map[entities.ConflictKind]string{
	entities.ConflictFileContent:    "the file has different content in two sources",
	entities.ConflictSameShape:      "two sources define the type identically",
	entities.ConflictDifferentShape: "two sources define the type differently",
}

// FindConflicts reports clashes between the inputs: one file path with different content, or one type name
// defined twice. Comments never count as a difference.
func FindConflicts(inputs []SchemaInput) entities.SchemaConflicts {
	var out entities.SchemaConflicts
	files := make(map[string]owner)
	types := make(map[string]owner)

	for _, in := range inputs {
		fds := &descriptorpb.FileDescriptorSet{}
		if len(in.Bytes) == 0 || proto.Unmarshal(in.Bytes, fds) != nil {
			continue
		}
		for _, file := range fds.File {
			file.SourceCodeInfo = nil
			name := file.GetName()
			ref := entities.SchemaRef{SourceID: in.SourceID, Revision: in.Revision, File: name}
			content := shapeOf(file)

			if first, ok := files[name]; ok {
				if !bytes.Equal(first.shape, content) && !strings.HasPrefix(name, "google/") {
					out = append(out, newConflict(entities.ConflictFileContent, name, first.ref, ref))
				}
				continue
			}
			files[name] = owner{ref: ref, shape: content}

			for _, t := range topLevelTypes(file) {
				first, ok := types[t.name]
				if !ok {
					types[t.name] = owner{ref: ref, shape: t.shape}
					continue
				}
				kind := entities.ConflictDifferentShape
				if bytes.Equal(first.shape, t.shape) {
					kind = entities.ConflictSameShape
				}
				out = append(out, newConflict(kind, t.name, first.ref, ref))
			}
		}
	}
	return out
}

func newConflict(kind entities.ConflictKind, symbol string, first, second entities.SchemaRef) *entities.SchemaConflict {
	return entities.SchemaConflictNew(func(c *entities.SchemaConflict) {
		c.Kind = kind
		c.Severity = entities.SeverityError
		if kind == entities.ConflictSameShape {
			c.Severity = entities.SeverityInfo
		}
		c.Symbol = symbol
		c.First, c.Second = first, second
		c.Reason = conflictReasons[kind]
	})
}

func topLevelTypes(file *descriptorpb.FileDescriptorProto) []namedShape {
	pkg := file.GetPackage()
	out := make([]namedShape, 0, len(file.GetMessageType())+len(file.GetEnumType())+len(file.GetService()))
	for _, m := range file.GetMessageType() {
		out = append(out, namedShape{qualifiedName(pkg, m.GetName()), shapeOf(m)})
	}
	for _, e := range file.GetEnumType() {
		out = append(out, namedShape{qualifiedName(pkg, e.GetName()), shapeOf(e)})
	}
	for _, s := range file.GetService() {
		out = append(out, namedShape{qualifiedName(pkg, s.GetName()), shapeOf(s)})
	}
	return out
}

func shapeOf(m proto.Message) []byte {
	b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(m) //nolint:errcheck // descriptors always marshal
	return b
}

func qualifiedName(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}
