// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

var dependencyPrefixes = []string{
	"google/protobuf/", "google/api/", "google/rpc/", "google/type/", "buf/validate/", "validate/", "gogoproto/",
}

// NormalizeDescriptorSet keeps only comments in an uploaded set's source info and checks that its files link.
func NormalizeDescriptorSet(data []byte) ([]byte, *Schema, error) {
	set := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, set); err != nil {
		return nil, nil, coreerrs.WrapOperation(err, "unmarshal descriptor set")
	}
	for _, f := range set.GetFile() {
		f.SourceCodeInfo = KeepComments(f.GetSourceCodeInfo())
	}
	out, err := proto.MarshalOptions{Deterministic: true}.Marshal(set)
	if err != nil {
		return nil, nil, coreerrs.WrapOperation(err, "marshal descriptor set")
	}
	schema, err := ParseSchema(out)
	if err != nil {
		return nil, nil, err
	}
	return out, schema, nil
}

// OwnFiles lists the files of a schema outside well-known dependency trees, or every file when all of them are.
func OwnFiles(schema *Schema) []string {
	var own, all []string
	schema.Files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		all = append(all, fd.Path())
		if !slices.ContainsFunc(dependencyPrefixes, func(p string) bool { return strings.HasPrefix(fd.Path(), p) }) {
			own = append(own, fd.Path())
		}
		return true
	})
	if len(own) == 0 {
		own = all
	}
	slices.Sort(own)
	return own
}
