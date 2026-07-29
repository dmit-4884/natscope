// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package grpchelpers

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestVTCodec_Name pins the codec identifier to "proto" so it shadows the default binary codec.
func TestVTCodec_Name(t *testing.T) {
	t.Parallel()
	if got := (VTCodec{}).Name(); got != "proto" {
		t.Fatalf("Name() = %q, want %q", got, "proto")
	}
}

// TestVTCodec_RoundTripVTPath round-trips a message with MarshalVT codegen — the fast path should fire.
func TestVTCodec_RoundTripVTPath(t *testing.T) {
	t.Parallel()

	src := &natspb.NatsMessage{
		Sequence:    42,
		Subject:     "orders.created",
		Timestamp:   timestamppb.New(time.UnixMilli(1700000000000).UTC()),
		DataBase64:  "aGVsbG8=",
		DataSize:    5,
		ContentType: "text",
		Truncated:   true,
	}
	codec := VTCodec{}
	wire, err := codec.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if codec.Size(src) != len(wire) {
		t.Fatalf("Size()=%d != len(wire)=%d — vt SizeVT must agree with Marshal output",
			codec.Size(src), len(wire))
	}
	got := &natspb.NatsMessage{}
	if err := codec.Unmarshal(wire, got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Sequence != 42 || got.Subject != "orders.created" || !got.Truncated {
		t.Fatalf("round-trip lost fields: %+v", got)
	}
}

// TestVTCodec_FallbackPath round-trips emptypb.Empty (no MarshalVT codegen) to verify the proto.Marshal/Unmarshal fallback, guarding Connect-internal types like google.rpc.Status.
func TestVTCodec_FallbackPath(t *testing.T) {
	t.Parallel()

	codec := VTCodec{}
	wire, err := codec.Marshal(&emptypb.Empty{})
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	if len(wire) != 0 {
		t.Fatalf("empty.Marshal must yield 0 bytes, got %d", len(wire))
	}
	if err := codec.Unmarshal(wire, &emptypb.Empty{}); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
}

// TestVTCodec_RejectsNonProto verifies a clear error when handed a non-proto.Message.
func TestVTCodec_RejectsNonProto(t *testing.T) {
	t.Parallel()

	type notAProto struct{ X int }
	if _, err := (VTCodec{}).Marshal(&notAProto{}); err == nil {
		t.Fatal("Marshal(non-proto) must error")
	}
	if err := (VTCodec{}).Unmarshal([]byte("zzz"), &notAProto{}); err == nil {
		t.Fatal("Unmarshal(non-proto) must error")
	}
}
