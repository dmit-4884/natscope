// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// extResolver combines GlobalTypes with dynamically loaded extensions.
type extResolver struct {
	types *protoregistry.Types
}

// FindExtensionByName finds an extension by name.
func (r *extResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if r.types != nil {
		if ext, err := r.types.FindExtensionByName(field); err == nil {
			return ext, nil
		}
	}
	return protoregistry.GlobalTypes.FindExtensionByName(field)
}

// FindExtensionByNumber finds an extension by number.
func (r *extResolver) FindExtensionByNumber(
	message protoreflect.FullName,
	field protoreflect.FieldNumber,
) (protoreflect.ExtensionType, error) {
	if r.types != nil {
		if ext, err := r.types.FindExtensionByNumber(message, field); err == nil {
			return ext, nil
		}
	}
	return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
}

// newExtResolver creates an extension resolver that includes extensions from parsed proto files.
func newExtResolver(md protoreflect.MessageDescriptor) protoregistry.ExtensionTypeResolver {
	types := &protoregistry.Types{}

	visited := make(map[string]bool)
	collectExtensions(md.ParentFile(), types, visited)

	return &extResolver{types: types}
}

// collectExtensions recursively collects extensions from a file and its dependencies.
func collectExtensions(fd protoreflect.FileDescriptor, types *protoregistry.Types, visited map[string]bool) {
	if fd == nil {
		return
	}

	name := fd.Path()
	if visited[name] {
		return
	}
	visited[name] = true

	// Process dependencies
	imports := fd.Imports()
	for i := range imports.Len() {
		imp := imports.Get(i)
		collectExtensions(imp.FileDescriptor, types, visited)
	}

	// Register extensions from this file
	exts := fd.Extensions()
	for i := range exts.Len() {
		ext := exts.Get(i)
		extType := dynamicpb.NewExtensionType(ext)
		_ = types.RegisterExtension(extType) //nolint:errcheck // best effort
	}
}
