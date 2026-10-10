// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package proto

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func loadBenchSchema(tb testing.TB) (*protoutils.Schema, protoreflect.MessageDescriptor) {
	tb.Helper()
	schema := prototest.Schema(tb, map[string]string{"bench.proto": `
syntax = "proto3";
package bench;
message Payload {
  bytes  body         = 1;
  string subject      = 2;
  int64  timestamp_ms = 3;
  map<string, string> headers = 4;
}`})
	md, ok := schema.Message("bench.Payload")
	if !ok {
		tb.Fatalf("Payload not found")
	}
	return schema, md
}

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

func benchmarkDecode(b *testing.B, bodySize int) {
	schema, md := loadBenchSchema(b)
	data := buildPayloadBytes(b, md, bodySize)

	b.SetBytes(int64(len(data)))
	for b.Loop() {
		_ = decodeWithDescriptor(schema, md, data, "bench.Payload", true)
	}
}

func BenchmarkProtoDecode_4KB(b *testing.B) { benchmarkDecode(b, 4*1024) }

func BenchmarkProtoDecode_1_6MB(b *testing.B) { benchmarkDecode(b, 1600*1024) }

func BenchmarkProtoDecode_5MB(b *testing.B) { benchmarkDecode(b, 5*1024*1024) }

func BenchmarkProtoUnmarshalOnly_1_6MB(b *testing.B) {
	schema, md := loadBenchSchema(b)
	data := buildPayloadBytes(b, md, 1600*1024)

	b.SetBytes(int64(len(data)))
	for b.Loop() {
		_ = schema.ParseBinary(data, dynamicpb.NewMessage(md))
	}
}

func BenchmarkProtoJSONMarshalOnly_1_6MB(b *testing.B) {
	schema, md := loadBenchSchema(b)
	data := buildPayloadBytes(b, md, 1600*1024)
	msg := dynamicpb.NewMessage(md)
	if err := schema.ParseBinary(data, msg); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	for b.Loop() {
		_, _ = schema.RenderJSON(msg)
	}
}
