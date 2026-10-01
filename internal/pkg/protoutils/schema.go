// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// Schema is a parsed descriptor set with a type resolver and lookups by full name.
type Schema struct {
	Files    *protoregistry.Files
	Types    *dynamicpb.Types
	Messages map[string]protoreflect.MessageDescriptor
	Enums    map[string]protoreflect.EnumDescriptor
	Services map[string]protoreflect.ServiceDescriptor
}

// ParseSchema parses FileDescriptorSet bytes; map entry types are left out of Messages.
func ParseSchema(data []byte) (*Schema, error) {
	if len(data) == 0 {
		return nil, errors.New("empty descriptor set")
	}

	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fds); err != nil {
		return nil, coreerrs.WrapOperation(err, "unmarshal descriptor set")
	}

	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "create file registry")
	}

	s := &Schema{
		Files:    files,
		Types:    dynamicpb.NewTypes(files),
		Messages: make(map[string]protoreflect.MessageDescriptor),
		Enums:    make(map[string]protoreflect.EnumDescriptor),
		Services: make(map[string]protoreflect.ServiceDescriptor),
	}
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		s.collect(fd.Messages(), fd.Enums())
		services := fd.Services()
		for i := range services.Len() {
			s.Services[string(services.Get(i).FullName())] = services.Get(i)
		}
		return true
	})
	return s, nil
}

func (s *Schema) collect(msgs protoreflect.MessageDescriptors, enums protoreflect.EnumDescriptors) {
	for i := range enums.Len() {
		ed := enums.Get(i)
		s.Enums[string(ed.FullName())] = ed
	}
	for i := range msgs.Len() {
		md := msgs.Get(i)
		if !md.IsMapEntry() {
			s.Messages[string(md.FullName())] = md
		}
		s.collect(md.Messages(), md.Enums())
	}
}

// Message looks up a message by full name.
func (s *Schema) Message(fullName string) (protoreflect.MessageDescriptor, bool) {
	md, ok := s.Messages[fullName]
	return md, ok
}

// Resolver resolves Any payload types and extensions.
type Resolver interface {
	protoregistry.ExtensionTypeResolver
	protoregistry.MessageTypeResolver
}

type decodeResolver struct {
	*dynamicpb.Types
}

var opaqueAnyType = func() protoreflect.MessageType {
	fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:        proto.String("natscope/opaque.proto"),
		Package:     proto.String("natscope.opaque"),
		Syntax:      proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Unresolved")}},
	}, nil)
	if err != nil {
		panic(fmt.Sprintf("build opaque any type: %v", err))
	}
	return dynamicpb.NewMessageType(fd.Messages().Get(0))
}()

func (r decodeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	mt, err := r.Types.FindMessageByURL(url)
	if errors.Is(err, protoregistry.NotFound) {
		return opaqueAnyType, nil
	}
	return mt, err
}

// DecodeResolver renders an Any of a type missing from the schema as just its "@type".
func (s *Schema) DecodeResolver() Resolver {
	return decodeResolver{s.Types}
}

type extensionResolver struct {
	*dynamicpb.Types
}

func (r extensionResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if xt, err := r.Types.FindExtensionByName(field); err == nil {
		return xt, nil
	}
	return protoregistry.GlobalTypes.FindExtensionByName(field)
}

func (r extensionResolver) FindExtensionByNumber(
	message protoreflect.FullName,
	field protoreflect.FieldNumber,
) (protoreflect.ExtensionType, error) {
	if xt, err := r.Types.FindExtensionByNumber(message, field); err == nil {
		return xt, nil
	}
	return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
}
