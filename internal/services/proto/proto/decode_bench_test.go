// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// loadBenchMessageDescriptor compiles a synthetic .proto and returns the
// "bench.Payload" descriptor via the production parse path.
func loadBenchMessageDescriptor(tb testing.TB) protoreflect.MessageDescriptor {
	tb.Helper()
	src := `
syntax = "proto3";
package bench;
message Payload {
  bytes  body         = 1;
  string subject      = 2;
  int64  timestamp_ms = 3;
  map<string, string> headers = 4;
}`
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				"bench.proto": src,
			}),
		},
	}
	files, err := compiler.Compile(tb.Context(), "bench.proto")
	if err != nil {
		tb.Fatalf("compile: %v", err)
	}
	fd, err := files.AsResolver().FindFileByPath("bench.proto")
	if err != nil {
		tb.Fatalf("find file: %v", err)
	}
	md := fd.Messages().ByName("Payload")
	if md == nil {
		tb.Fatalf("Payload not found")
	}
	return md
}

// buildPayloadBytes creates wire-format bytes for a Payload message of approx
// `bodySize` payload size. Returns the serialized bytes.
func buildPayloadBytes(tb testing.TB, md protoreflect.MessageDescriptor, bodySize int) []byte {
	tb.Helper()
	msg := dynamicpb.NewMessage(md)

	body := make([]byte, bodySize)
	_, _ = rand.Read(body)

	msg.Set(md.Fields().ByName("body"), protoreflect.ValueOfBytes(body))
	msg.Set(md.Fields().ByName("subject"), protoreflect.ValueOfString("bench.subject.long.name"))
	msg.Set(md.Fields().ByName("timestamp_ms"), protoreflect.ValueOfInt64(1700000000000))

	headersFD := md.Fields().ByName("headers")
	headers := msg.Mutable(headersFD).Map()
	for i := range 4 {
		k := protoreflect.ValueOfString("header-" + string('a'+rune(i))).MapKey()
		v := protoreflect.ValueOfString(strings.Repeat("v", 32))
		headers.Set(k, v)
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		tb.Fatalf("marshal: %v", err)
	}
	return data
}

// BenchmarkProtoDecode_1_6MB measures the production decode path on a 1.6 MB
// protobuf: dynamicpb.New + proto.Unmarshal + protojson.Marshal.
func BenchmarkProtoDecode_1_6MB(b *testing.B) {
	md := loadBenchMessageDescriptor(b)
	data := buildPayloadBytes(b, md, 1600*1024)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		_ = decodeWithDescriptor(md, data, "bench.Payload")
	}
}

// BenchmarkProtoDecode_5MB measures the top of the user's payload range.
func BenchmarkProtoDecode_5MB(b *testing.B) {
	md := loadBenchMessageDescriptor(b)
	data := buildPayloadBytes(b, md, 5*1024*1024)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		_ = decodeWithDescriptor(md, data, "bench.Payload")
	}
}

// BenchmarkProtoDecode_4KB measures the small-message baseline.
func BenchmarkProtoDecode_4KB(b *testing.B) {
	md := loadBenchMessageDescriptor(b)
	data := buildPayloadBytes(b, md, 4*1024)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		_ = decodeWithDescriptor(md, data, "bench.Payload")
	}
}

// BenchmarkProtoUnmarshalOnly_1_6MB isolates proto.Unmarshal so we can separate
// it from the protojson.Marshal half.
func BenchmarkProtoUnmarshalOnly_1_6MB(b *testing.B) {
	md := loadBenchMessageDescriptor(b)
	data := buildPayloadBytes(b, md, 1600*1024)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		msg := dynamicpb.NewMessage(md)
		_ = proto.Unmarshal(data, msg)
	}
}

// BenchmarkProtoJSONMarshalOnly_1_6MB isolates the protojson.Marshal half.
func BenchmarkProtoJSONMarshalOnly_1_6MB(b *testing.B) {
	md := loadBenchMessageDescriptor(b)
	data := buildPayloadBytes(b, md, 1600*1024)
	msg := dynamicpb.NewMessage(md)
	if err := proto.Unmarshal(data, msg); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		_, _ = protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: true,
		}.Marshal(msg)
	}
}
