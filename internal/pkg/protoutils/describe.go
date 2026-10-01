// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"cmp"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Summaries lists the messages, enums and services of the schema.
func (s *Schema) Summaries() []entities.SchemaType {
	out := make([]entities.SchemaType, 0, len(s.Messages)+len(s.Enums)+len(s.Services))
	for _, md := range s.Messages {
		out = append(out, summary(md, entities.SchemaTypeMessage, md.Fields().Len()))
	}
	for _, ed := range s.Enums {
		out = append(out, summary(ed, entities.SchemaTypeEnum, ed.Values().Len()))
	}
	for _, sd := range s.Services {
		out = append(out, summary(sd, entities.SchemaTypeService, sd.Methods().Len()))
	}
	return out
}

func summary(d protoreflect.Descriptor, kind entities.SchemaTypeKind, members int) entities.SchemaType {
	first, _, _ := strings.Cut(Comment(d), "\n")
	return entities.SchemaType{
		FullName:    string(d.FullName()),
		Kind:        kind,
		File:        d.ParentFile().Path(),
		Package:     string(d.ParentFile().Package()),
		Comment:     first,
		MemberCount: int32(members), //nolint:gosec // bounded by descriptor size
	}
}

// Describe returns a type and, with reachable, every message and enum its fields or methods lead to.
func (s *Schema) Describe(fullName string, reachable bool) (*entities.TypeDescription, bool) {
	w := &typeWalker{schema: s, seen: map[protoreflect.FullName]bool{}, reachable: reachable, out: &entities.TypeDescription{}}
	switch {
	case s.Messages[fullName] != nil:
		w.message(s.Messages[fullName])
	case s.Enums[fullName] != nil:
		w.enum(s.Enums[fullName])
	case s.Services[fullName] != nil:
		w.service(s.Services[fullName])
	default:
		return nil, false
	}
	return w.out, true
}

type typeWalker struct {
	schema    *Schema
	seen      map[protoreflect.FullName]bool
	reachable bool
	out       *entities.TypeDescription
}

func (w *typeWalker) visit(d protoreflect.Descriptor) bool {
	if w.seen[d.FullName()] {
		return false
	}
	w.seen[d.FullName()] = true
	return true
}

func (w *typeWalker) message(md protoreflect.MessageDescriptor) {
	if !w.visit(md) {
		return
	}
	w.out.Messages = append(w.out.Messages, DescribeMessage(md))
	if !w.reachable {
		return
	}
	fields := md.Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		if fd.IsMap() {
			fd = fd.MapValue()
		}
		w.follow(fd)
	}
}

func (w *typeWalker) follow(fd protoreflect.FieldDescriptor) {
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		w.message(fd.Message())
	case protoreflect.EnumKind:
		w.enum(fd.Enum())
	default:
	}
}

func (w *typeWalker) enum(ed protoreflect.EnumDescriptor) {
	if w.visit(ed) {
		w.out.Enums = append(w.out.Enums, DescribeEnum(ed))
	}
}

func (w *typeWalker) service(sd protoreflect.ServiceDescriptor) {
	if !w.visit(sd) {
		return
	}
	w.out.Services = append(w.out.Services, DescribeService(sd))
	if !w.reachable {
		return
	}
	methods := sd.Methods()
	for i := range methods.Len() {
		w.message(methods.Get(i).Input())
		w.message(methods.Get(i).Output())
	}
}

// DescribeMessage builds the schema view of a message.
func DescribeMessage(md protoreflect.MessageDescriptor) *entities.SchemaMessage {
	fields := md.Fields()
	out := &entities.SchemaMessage{
		FullName:   string(md.FullName()),
		File:       md.ParentFile().Path(),
		Comment:    Comment(md),
		Deprecated: deprecated(md),
		Fields:     make([]*entities.SchemaField, 0, fields.Len()),
	}
	for i := range fields.Len() {
		out.Fields = append(out.Fields, describeField(fields.Get(i)))
	}
	return out
}

func describeField(fd protoreflect.FieldDescriptor) *entities.SchemaField {
	f := &entities.SchemaField{
		Name:       string(fd.Name()),
		JSONName:   fd.JSONName(),
		Number:     int32(fd.Number()),
		Repeated:   fd.IsList(),
		Optional:   fd.HasOptionalKeyword(),
		Required:   fd.Cardinality() == protoreflect.Required,
		Deprecated: deprecated(fd),
		Comment:    Comment(fd),
	}
	if od := fd.ContainingOneof(); od != nil && !od.IsSynthetic() {
		f.Oneof = string(od.Name())
	}
	elem := fd
	if fd.IsMap() {
		f.MapKey = fd.MapKey().Kind().String()
		elem = fd.MapValue()
	}
	f.Kind = elem.Kind().String()
	switch elem.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		f.TypeName = string(elem.Message().FullName())
	case protoreflect.EnumKind:
		f.TypeName = string(elem.Enum().FullName())
	default:
	}
	return f
}

// DescribeEnum builds the schema view of an enum.
func DescribeEnum(ed protoreflect.EnumDescriptor) *entities.SchemaEnum {
	values := ed.Values()
	out := &entities.SchemaEnum{
		FullName:   string(ed.FullName()),
		File:       ed.ParentFile().Path(),
		Comment:    Comment(ed),
		Deprecated: deprecated(ed),
		Values:     make([]*entities.SchemaEnumValue, 0, values.Len()),
	}
	for i := range values.Len() {
		v := values.Get(i)
		out.Values = append(out.Values, &entities.SchemaEnumValue{
			Name:       string(v.Name()),
			Number:     int32(v.Number()),
			Comment:    Comment(v),
			Deprecated: deprecated(v),
		})
	}
	return out
}

// DescribeService builds the schema view of a service.
func DescribeService(sd protoreflect.ServiceDescriptor) *entities.SchemaService {
	methods := sd.Methods()
	out := &entities.SchemaService{
		FullName:   string(sd.FullName()),
		File:       sd.ParentFile().Path(),
		Comment:    Comment(sd),
		Deprecated: deprecated(sd),
		Methods:    make([]*entities.SchemaMethod, 0, methods.Len()),
	}
	for i := range methods.Len() {
		m := methods.Get(i)
		out.Methods = append(out.Methods, &entities.SchemaMethod{
			Name:            string(m.Name()),
			InputType:       string(m.Input().FullName()),
			OutputType:      string(m.Output().FullName()),
			ClientStreaming: m.IsStreamingClient(),
			ServerStreaming: m.IsStreamingServer(),
			Comment:         Comment(m),
			Deprecated:      deprecated(m),
		})
	}
	return out
}

// Comment returns the leading comment of a declaration, else its trailing one.
func Comment(d protoreflect.Descriptor) string {
	loc := d.ParentFile().SourceLocations().ByDescriptor(d)
	text := cmp.Or(strings.TrimSpace(loc.LeadingComments), strings.TrimSpace(loc.TrailingComments))
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(strings.TrimPrefix(line, " "), " \t")
	}
	return strings.Join(lines, "\n")
}

// KeepComments drops every source location without a leading or trailing comment.
func KeepComments(info *descriptorpb.SourceCodeInfo) *descriptorpb.SourceCodeInfo {
	var kept []*descriptorpb.SourceCodeInfo_Location
	for _, loc := range info.GetLocation() {
		if loc.GetLeadingComments() != "" || loc.GetTrailingComments() != "" {
			loc.LeadingDetachedComments = nil
			kept = append(kept, loc)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return &descriptorpb.SourceCodeInfo{Location: kept}
}

type deprecatable interface {
	GetDeprecated() bool
}

func deprecated(d protoreflect.Descriptor) bool {
	o, ok := d.Options().(deprecatable)
	return ok && o.GetDeprecated()
}
