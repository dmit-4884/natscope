// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func mustMarshalSet(t *testing.T, file *descriptorpb.FileDescriptorProto) []byte {
	t.Helper()
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}}
	data, err := proto.Marshal(set)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func TestMergeWithReport_SameSymbolDifferentFile_Error(t *testing.T) {
	// Same FQN declared in two different files in two different sources.
	pkg := "ru.one2work.types.agents"
	msgName := "Statistics"

	file1 := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("agent_statistics.proto"),
		Package: proto.String(pkg),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String(msgName)},
		},
		Syntax: proto.String("proto3"),
	}
	file2 := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("agents_statistics.proto"),
		Package: proto.String(pkg),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String(msgName)},
		},
		Syntax: proto.String("proto3"),
	}

	report, err := MergeWithReport([]SchemaInput{
		{SourceID: "src-A", Tag: "v1", Bytes: mustMarshalSet(t, file1)},
		{SourceID: "src-B", Tag: "v3", Bytes: mustMarshalSet(t, file2)},
	})
	if err != nil {
		t.Fatalf("MergeWithReport: %v", err)
	}

	merged := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(report.Result, merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.File) != 1 {
		t.Errorf("expected 1 file after merge, got %d", len(merged.File))
	}
	if merged.File[0].GetName() != "agent_statistics.proto" {
		t.Errorf("first-wins broken: winner = %s", merged.File[0].GetName())
	}

	if len(report.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(report.Conflicts))
	}
	c := report.Conflicts[0]
	if c.Kind != entities.SameSymbolDifferentShape {
		t.Errorf("kind = %s, want %s", c.Kind, entities.SameSymbolDifferentShape)
	}
	if c.Severity != entities.SeverityError {
		t.Errorf("severity = %s, want %s", c.Severity, entities.SeverityError)
	}
	if c.Symbol != pkg+"."+msgName {
		t.Errorf("symbol = %s, want %s", c.Symbol, pkg+"."+msgName)
	}
	if c.Winner.SourceID != "src-A" || c.Loser.SourceID != "src-B" {
		t.Errorf("winner/loser ordering wrong: %+v", c)
	}
}

func TestMergeWithReport_DifferentTypesNoConflict(t *testing.T) {
	pkg := "com.example"

	file1 := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("foo.proto"),
		Package:     proto.String(pkg),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Foo")}},
		Syntax:      proto.String("proto3"),
	}
	file2 := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("bar.proto"),
		Package:     proto.String(pkg),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Bar")}},
		Syntax:      proto.String("proto3"),
	}

	report, err := MergeWithReport([]SchemaInput{
		{SourceID: "src-A", Bytes: mustMarshalSet(t, file1)},
		{SourceID: "src-B", Bytes: mustMarshalSet(t, file2)},
	})
	if err != nil {
		t.Fatalf("MergeWithReport: %v", err)
	}

	merged := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(report.Result, merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.File) != 2 {
		t.Errorf("expected 2 files after merge, got %d", len(merged.File))
	}
	if len(report.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(report.Conflicts))
	}
}

func TestMergeWithReport_SameFileNameDifferentContent_Error(t *testing.T) {
	file1 := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("common.proto"),
		Package:     proto.String("com.example"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Common")}},
		Syntax:      proto.String("proto3"),
	}
	file2 := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("common.proto"),
		Package:     proto.String("com.example"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("CommonV2")}},
		Syntax:      proto.String("proto3"),
	}

	report, err := MergeWithReport([]SchemaInput{
		{SourceID: "src-A", Bytes: mustMarshalSet(t, file1)},
		{SourceID: "src-B", Bytes: mustMarshalSet(t, file2)},
	})
	if err != nil {
		t.Fatalf("MergeWithReport: %v", err)
	}

	merged := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(report.Result, merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.File) != 1 {
		t.Errorf("expected 1 file after merge, got %d", len(merged.File))
	}
	if len(report.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(report.Conflicts))
	}
	c := report.Conflicts[0]
	if c.Kind != entities.DuplicateFileDifferentContent {
		t.Errorf("kind = %s, want duplicate_file_different_content", c.Kind)
	}
	if c.Severity != entities.SeverityError {
		t.Errorf("severity = %s, want error", c.Severity)
	}
}

func TestMergeWithReport_SameFileNameIdenticalContent_NoConflict(t *testing.T) {
	file := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("common.proto"),
		Package:     proto.String("com.example"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Common")}},
		Syntax:      proto.String("proto3"),
	}
	bytes := mustMarshalSet(t, file)

	report, err := MergeWithReport([]SchemaInput{
		{SourceID: "src-A", Bytes: bytes},
		{SourceID: "src-B", Bytes: bytes},
	})
	if err != nil {
		t.Fatalf("MergeWithReport: %v", err)
	}

	merged := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(report.Result, merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.File) != 1 {
		t.Errorf("expected 1 file after merge, got %d", len(merged.File))
	}
	if len(report.Conflicts) != 0 {
		t.Errorf("identical-content duplicates must not produce conflicts, got %d", len(report.Conflicts))
	}
}

func TestMergeWithReport_EnumConflict_Error(t *testing.T) {
	pkg := "com.example"

	file1 := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("status_v1.proto"),
		Package: proto.String(pkg),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Status"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("UNKNOWN"), Number: proto.Int32(0)},
				},
			},
		},
		Syntax: proto.String("proto3"),
	}
	file2 := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("status_v2.proto"),
		Package: proto.String(pkg),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Status"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("UNKNOWN"), Number: proto.Int32(0)},
					{Name: proto.String("ACTIVE"), Number: proto.Int32(1)},
				},
			},
		},
		Syntax: proto.String("proto3"),
	}

	report, err := MergeWithReport([]SchemaInput{
		{SourceID: "src-A", Bytes: mustMarshalSet(t, file1)},
		{SourceID: "src-B", Bytes: mustMarshalSet(t, file2)},
	})
	if err != nil {
		t.Fatalf("MergeWithReport: %v", err)
	}

	merged := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(report.Result, merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.File) != 1 {
		t.Errorf("expected 1 file after merge, got %d", len(merged.File))
	}
	if len(report.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(report.Conflicts))
	}
	c := report.Conflicts[0]
	if c.Symbol != pkg+".Status" {
		t.Errorf("symbol = %s, want %s.Status", c.Symbol, pkg)
	}
	if c.Severity != entities.SeverityError {
		t.Errorf("severity = %s, want error", c.Severity)
	}
}
