// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"
)

const describeProto = `syntax = "proto3";
package shop;

import "google/protobuf/timestamp.proto";

// An order placed by a customer.
// Spans two lines.
message Order {
  // Order id.
  string id = 1;
  repeated Item items = 2;
  map<string, Status> statuses = 3;
  optional int64 total = 4; // In cents.
  oneof payment {
    string card = 5;
    string cash = 6;
  }
  google.protobuf.Timestamp created_at = 7 [deprecated = true];

  message Item {
    string sku = 1;
    Order parent = 2;
  }
}

/* Lifecycle of an order. */
enum Status {
  STATUS_UNSPECIFIED = 0;
  // Shipped to the customer.
  STATUS_SHIPPED = 1;
}

message Empty {}

// Order API.
service Orders {
  // Streams orders.
  rpc Watch(Empty) returns (stream Order);
}
`

func describeSchema(t *testing.T) *protoutils.Schema {
	t.Helper()
	return prototest.Schema(t, map[string]string{"shop/order.proto": describeProto})
}

func TestSchema_Summaries(t *testing.T) {
	t.Parallel()
	byName := map[string]entities.SchemaType{}
	for _, s := range describeSchema(t).Summaries() {
		byName[s.FullName] = s
	}

	assert.Equal(t, entities.SchemaType{
		FullName: "shop.Order", Kind: entities.SchemaTypeMessage, File: "shop/order.proto", Package: "shop",
		Comment: "An order placed by a customer.", MemberCount: 7,
	}, byName["shop.Order"])
	assert.Equal(t, entities.SchemaTypeEnum, byName["shop.Status"].Kind)
	assert.Equal(t, "Lifecycle of an order.", byName["shop.Status"].Comment)
	assert.Equal(t, int32(2), byName["shop.Status"].MemberCount)
	assert.Equal(t, entities.SchemaTypeService, byName["shop.Orders"].Kind)
	assert.Equal(t, int32(1), byName["shop.Orders"].MemberCount)
	assert.Contains(t, byName, "shop.Order.Item")
	assert.Contains(t, byName, "google.protobuf.Timestamp")
	assert.NotContains(t, byName, "shop.Order.StatusesEntry", "map entries are not types of their own")
}

func TestSchema_Describe(t *testing.T) {
	t.Parallel()
	schema := describeSchema(t)

	t.Run("message fields", func(t *testing.T) {
		t.Parallel()
		desc, ok := schema.Describe("shop.Order", false)
		require.True(t, ok)
		require.Len(t, desc.Messages, 1)
		assert.Empty(t, desc.Enums)

		order := desc.Messages[0]
		assert.Equal(t, "An order placed by a customer.\nSpans two lines.", order.Comment)
		fields := map[string]*entities.SchemaField{}
		for _, f := range order.Fields {
			fields[f.Name] = f
		}
		assert.Equal(t, &entities.SchemaField{Name: "id", JSONName: "id", Number: 1, Kind: "string", Comment: "Order id."}, fields["id"])
		assert.Equal(t, &entities.SchemaField{
			Name: "items", JSONName: "items", Number: 2, Kind: "message", TypeName: "shop.Order.Item", Repeated: true,
		}, fields["items"])
		assert.Equal(t, &entities.SchemaField{
			Name: "statuses", JSONName: "statuses", Number: 3, Kind: "enum", TypeName: "shop.Status", MapKey: "string",
		}, fields["statuses"])
		assert.Equal(t, &entities.SchemaField{
			Name: "total", JSONName: "total", Number: 4, Kind: "int64", Optional: true, Comment: "In cents.",
		}, fields["total"])
		assert.Equal(t, "payment", fields["card"].Oneof)
		assert.Equal(t, "createdAt", fields["created_at"].JSONName)
		assert.True(t, fields["created_at"].Deprecated)
	})

	t.Run("reachable types", func(t *testing.T) {
		t.Parallel()
		desc, ok := schema.Describe("shop.Order", true)
		require.True(t, ok)
		var names []string
		for _, m := range desc.Messages {
			names = append(names, m.FullName)
		}
		assert.Equal(t, []string{"shop.Order", "shop.Order.Item", "google.protobuf.Timestamp"}, names)
		require.Len(t, desc.Enums, 1)
		assert.Equal(t, "Shipped to the customer.", desc.Enums[0].Values[1].Comment)
	})

	t.Run("service", func(t *testing.T) {
		t.Parallel()
		desc, ok := schema.Describe("shop.Orders", true)
		require.True(t, ok)
		require.Len(t, desc.Services, 1)
		assert.Equal(t, "Order API.", desc.Services[0].Comment)
		assert.Equal(t, &entities.SchemaMethod{
			Name: "Watch", InputType: "shop.Empty", OutputType: "shop.Order", ServerStreaming: true, Comment: "Streams orders.",
		}, desc.Services[0].Methods[0])
		assert.Equal(t, "shop.Empty", desc.Messages[0].FullName)
		assert.Equal(t, "shop.Order", desc.Messages[1].FullName)
	})

	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		desc, ok := schema.Describe("shop.Status", false)
		require.True(t, ok)
		require.Len(t, desc.Enums, 1)
		assert.Empty(t, desc.Messages)
	})

	t.Run("unknown", func(t *testing.T) {
		t.Parallel()
		_, ok := schema.Describe("shop.Missing", true)
		assert.False(t, ok)
	})
}
