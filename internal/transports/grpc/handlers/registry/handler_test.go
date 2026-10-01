// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// --- Mocks ---

// mockProtoSvc embeds proto.Registry; only registry methods are overridden.
type mockProtoSvc struct {
	protosvc.Registry
	types         []entities.SchemaType
	description   *entities.TypeDescription
	describeErr   error
	gotReachable  bool
	exampleResult map[string]interface{}
	exampleErr    error
	status        *entities.SchemaStatus
	statusErr     error
}

func (m *mockProtoSvc) ListTypes(_ context.Context, sourceID string) ([]entities.SchemaType, error) {
	if sourceID == "missing" {
		return nil, errs.ErrMappingSourceNotFound
	}
	return m.types, nil
}

func (m *mockProtoSvc) DescribeType(_ context.Context, _, _ string, reachable bool) (*entities.TypeDescription, error) {
	m.gotReachable = reachable
	return m.description, m.describeErr
}

func (m *mockProtoSvc) GenerateExample(_ context.Context, _, _ string) (any, error) {
	return m.exampleResult, m.exampleErr
}

func (m *mockProtoSvc) SchemaStatus(_ context.Context) (*entities.SchemaStatus, error) {
	return m.status, m.statusErr
}

// --- Tests ---

func TestHandler_ListTypes(t *testing.T) {
	t.Parallel()

	t.Run("maps kinds", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{types: []entities.SchemaType{
			{FullName: "test.Msg", Kind: entities.SchemaTypeMessage, SourceID: "src-1", MemberCount: 2, Comment: "A message."},
			{FullName: "test.Status", Kind: entities.SchemaTypeEnum, SourceID: "src-1", Dependency: true},
			{FullName: "test.Api", Kind: entities.SchemaTypeService, SourceID: "src-1"},
		}}

		resp, err := New(svc).ListTypes(t.Context(), connect.NewRequest(&registrypb.ListTypesRequest{}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Types, 3)
		assert.Equal(t, "test.Msg", resp.Msg.Types[0].GetFullName())
		assert.Equal(t, "src-1", resp.Msg.Types[0].GetSourceId())
		assert.Equal(t, int32(2), resp.Msg.Types[0].GetMemberCount())
		assert.Equal(t, "A message.", resp.Msg.Types[0].GetComment())
		assert.Equal(t, protopb.SchemaTypeKind_SCHEMA_TYPE_KIND_MESSAGE, resp.Msg.Types[0].GetKind())
		assert.Equal(t, protopb.SchemaTypeKind_SCHEMA_TYPE_KIND_ENUM, resp.Msg.Types[1].GetKind())
		assert.True(t, resp.Msg.Types[1].GetDependency())
		assert.Equal(t, protopb.SchemaTypeKind_SCHEMA_TYPE_KIND_SERVICE, resp.Msg.Types[2].GetKind())
	})

	t.Run("unknown source", func(t *testing.T) {
		t.Parallel()
		_, err := New(&mockProtoSvc{}).ListTypes(t.Context(), connect.NewRequest(&registrypb.ListTypesRequest{SourceId: new("missing")}))
		assert.ErrorIs(t, err, errs.ErrMappingSourceNotFound)
	})
}

func TestHandler_DescribeType(t *testing.T) {
	t.Parallel()

	t.Run("converts nested members", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{description: &entities.TypeDescription{
			Messages: []*entities.SchemaMessage{{
				FullName: "test.Msg",
				Comment:  "A message.",
				Fields: []*entities.SchemaField{
					{Name: "tags", JSONName: "tags", Number: 1, Kind: "string", MapKey: "string", Comment: "Labels."},
					{Name: "status", JSONName: "status", Number: 2, Kind: "enum", TypeName: "test.Status", Oneof: "state"},
				},
			}},
			Enums: []*entities.SchemaEnum{{FullName: "test.Status", Values: []*entities.SchemaEnumValue{{Name: "OK", Number: 0}}}},
			Services: []*entities.SchemaService{{FullName: "test.Api", Methods: []*entities.SchemaMethod{
				{Name: "Get", InputType: "test.Msg", OutputType: "test.Msg", ServerStreaming: true},
			}}},
		}}

		resp, err := New(svc).DescribeType(t.Context(), connect.NewRequest(&registrypb.DescribeTypeRequest{
			SourceId: "src-1", FullName: "test.Msg", IncludeReachable: true,
		}))
		require.NoError(t, err)
		assert.True(t, svc.gotReachable)
		require.Len(t, resp.Msg.Messages, 1)
		fields := resp.Msg.Messages[0].GetFields()
		require.Len(t, fields, 2)
		assert.Equal(t, "string", fields[0].GetMapKey())
		assert.Equal(t, "Labels.", fields[0].GetComment())
		assert.Equal(t, "test.Status", fields[1].GetTypeName())
		assert.Equal(t, "state", fields[1].GetOneof())
		assert.Equal(t, "OK", resp.Msg.Enums[0].GetValues()[0].GetName())
		assert.True(t, resp.Msg.Services[0].GetMethods()[0].GetServerStreaming())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		_, err := New(&mockProtoSvc{describeErr: errs.ErrProtoTypeNotFound}).DescribeType(t.Context(),
			connect.NewRequest(&registrypb.DescribeTypeRequest{SourceId: "src-1", FullName: "missing.Msg"}))
		assert.ErrorIs(t, err, errs.ErrProtoTypeNotFound)
	})
}

func TestHandler_GenerateExample(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{exampleResult: map[string]interface{}{"field": "value"}}
		handler := New(svc)

		resp, err := handler.GenerateExample(t.Context(), connect.NewRequest(&registrypb.GenerateExampleRequest{
			SourceId: "src-1",
			FullName: "test.Msg",
		}))
		require.NoError(t, err)
		assert.Contains(t, resp.Msg.Json, "field")
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{exampleErr: errs.ErrProtoMessageNotFound}
		handler := New(svc)

		_, err := handler.GenerateExample(t.Context(), connect.NewRequest(&registrypb.GenerateExampleRequest{
			SourceId: "src-1",
			FullName: "missing.Msg",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoMessageNotFound)
	})
}

func TestHandler_GetSchemaStatus(t *testing.T) {
	t.Parallel()

	t.Run("conflicts", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{status: &entities.SchemaStatus{MessageTypes: 3, Conflicts: entities.SchemaConflicts{{
			Kind: entities.ConflictDifferentShape, Severity: entities.SeverityError, Symbol: "shop.Order",
			First:  entities.SchemaRef{SourceID: "a", Revision: "v1", File: "order.proto"},
			Second: entities.SchemaRef{SourceID: "b", File: "legacy/order.proto"},
			Reason: "two sources define the type differently",
		}}}}

		resp, err := New(svc).GetSchemaStatus(t.Context(), connect.NewRequest(&registrypb.GetSchemaStatusRequest{}))
		require.NoError(t, err)
		assert.Equal(t, int32(3), resp.Msg.MessageTypes)
		require.Len(t, resp.Msg.Conflicts, 1)
		c := resp.Msg.Conflicts[0]
		assert.Equal(t, protopb.ConflictKind_CONFLICT_KIND_DIFFERENT_SHAPE, c.Kind)
		assert.Equal(t, protopb.ConflictSeverity_CONFLICT_SEVERITY_ERROR, c.Severity)
		assert.Equal(t, "shop.Order", c.Symbol)
		assert.Equal(t, "v1", c.First.Revision)
		assert.Equal(t, "legacy/order.proto", c.Second.File)
		assert.Equal(t, "two sources define the type differently", c.Reason)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		_, err := New(&mockProtoSvc{statusErr: errs.ErrMappingSourceNotFound}).
			GetSchemaStatus(t.Context(), connect.NewRequest(&registrypb.GetSchemaStatusRequest{}))
		require.ErrorIs(t, err, errs.ErrMappingSourceNotFound)
	})
}
