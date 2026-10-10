// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package protoutils

import (
	"cmp"
	"math"
	"slices"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	knownBytesWeight  = 80
	coverageWeight    = 10
	maxSetFieldsBonus = 10
)

// Candidate is a message type a payload decodes as; Score runs from 0 to 100.
type Candidate struct {
	MessageType  string
	Score        int
	UnknownBytes int
	Message      *dynamicpb.Message
}

// DetectTypes ranks the given message types by how well the payload decodes as each, best first.
func (s *Schema) DetectTypes(data []byte, types []string, limit int) []Candidate {
	top, _, err := DecodeWire(data)
	if err != nil || len(top) == 0 {
		return nil
	}
	var out []Candidate
	for _, name := range types {
		md, ok := s.Messages[name]
		if !ok || md.IsMapEntry() || !fitsWire(md, top) {
			continue
		}
		msg := dynamicpb.NewMessage(md)
		if s.ParseBinary(data, msg) != nil {
			continue
		}
		if c, ok := score(msg, len(data)); ok {
			c.MessageType = name
			out = append(out, c)
		}
	}
	slices.SortStableFunc(out, func(a, b Candidate) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.MessageType, b.MessageType))
	})
	return out[:min(len(out), limit)]
}

func fitsWire(md protoreflect.MessageDescriptor, fields []*entities.WireField) bool {
	for _, f := range fields {
		fd := md.Fields().ByNumber(protoreflect.FieldNumber(f.Number))
		if fd != nil && !wireTypeFits(fd, f.WireType) {
			return false
		}
	}
	return true
}

func wireTypeFits(fd protoreflect.FieldDescriptor, wt entities.WireType) bool {
	var want entities.WireType
	switch fd.Kind() {
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.MessageKind:
		return wt == entities.WireBytes
	case protoreflect.GroupKind:
		return wt == entities.WireGroup
	case protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind, protoreflect.FloatKind:
		want = entities.WireFixed32
	case protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind, protoreflect.DoubleKind:
		want = entities.WireFixed64
	default:
		want = entities.WireVarint
	}
	return wt == want || (fd.IsList() && wt == entities.WireBytes)
}

func score(msg *dynamicpb.Message, size int) (Candidate, bool) {
	top := 0
	msg.Range(func(protoreflect.FieldDescriptor, protoreflect.Value) bool {
		top++
		return true
	})
	if top == 0 {
		return Candidate{}, false
	}
	unknown := 0
	for _, f := range UnknownFields(msg) {
		unknown += f.Size
	}
	known := float64(size-unknown) / float64(size)
	coverage := float64(top) / float64(msg.Descriptor().Fields().Len())
	points := known*knownBytesWeight + coverage*coverageWeight + float64(min(countSet(msg), maxSetFieldsBonus))
	return Candidate{Score: int(math.Round(points)), UnknownBytes: unknown, Message: msg}, true
}

func countSet(msg protoreflect.Message) int {
	n := 0
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		n++
		if fd.Message() != nil && !fd.IsList() && !fd.IsMap() {
			n += countSet(v.Message())
		}
		return true
	})
	return n
}
