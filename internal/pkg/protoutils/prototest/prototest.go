// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package prototest compiles inline .proto sources for tests.
package prototest

import (
	"slices"
	"testing"

	"github.com/bufbuild/protocompile"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// DescriptorSet compiles sources into FileDescriptorSet bytes that include all imports.
func DescriptorSet(tb testing.TB, sources map[string]string) []byte {
	tb.Helper()
	targets := make([]string, 0, len(sources))
	for name := range sources {
		targets = append(targets, name)
	}
	slices.Sort(targets)

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(sources),
		}),
	}
	files, err := compiler.Compile(tb.Context(), targets...)
	if err != nil {
		tb.Fatalf("compile protos: %v", err)
	}

	set := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}
	var add func(fd protoreflect.FileDescriptor)
	add = func(fd protoreflect.FileDescriptor) {
		if seen[fd.Path()] {
			return
		}
		seen[fd.Path()] = true
		imports := fd.Imports()
		for i := range imports.Len() {
			add(imports.Get(i).FileDescriptor)
		}
		set.File = append(set.File, protodesc.ToFileDescriptorProto(fd))
	}
	for _, f := range files {
		add(f)
	}
	data, err := proto.Marshal(set)
	if err != nil {
		tb.Fatalf("marshal descriptor set: %v", err)
	}
	return data
}

// Schema compiles sources into a parsed schema.
func Schema(tb testing.TB, sources map[string]string) *protoutils.Schema {
	tb.Helper()
	schema, err := protoutils.ParseSchema(DescriptorSet(tb, sources))
	if err != nil {
		tb.Fatalf("parse schema: %v", err)
	}
	return schema
}
