// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

type fakeMappings struct {
	mappingssvc.Service
	resolver *natsutil.MappingResolver
}

func (f fakeMappings) Resolver(context.Context) *natsutil.MappingResolver { return f.resolver }

type fakeRegistry struct {
	protosvc.Registry
	types []entities.SchemaType
}

func (f fakeRegistry) ListTypes(context.Context, string) ([]entities.SchemaType, error) {
	return f.types, nil
}

func newToolset() *Toolset {
	pinned := "fp-2"
	return New(
		&appconfig.Config{MCP: &appconfig.MCPConfig{AllowWrites: true}},
		nil, nil,
		fakeMappings{resolver: natsutil.NewMappingResolver(entities.SubjectMappings{
			{
				BaseEntity: entities.BaseEntity{Id: "m1"}, Pattern: "orders.*", MessageType: "o.v1.Order", SourceID: "src", PinnedFingerprint: &pinned,
				Framing: entities.Framing{Kind: entities.FramingGRPC},
			},
		})},
		fakeRegistry{types: []entities.SchemaType{{FullName: "o.v1.Refund", Kind: entities.SchemaTypeMessage, SourceID: "src"}}},
	)
}

func TestRequest(t *testing.T) {
	t.Parallel()
	ts := newToolset()
	payload := map[string]any{"id": "42"}

	req, out, err := ts.request(t.Context(), publishInput{Subject: "orders.created", JSON: payload})
	require.NoError(t, err)
	assert.Equal(t, `{"id":"42"}`, req.Data)
	assert.Equal(t, "o.v1.Order", *req.MessageType)
	assert.Equal(t, "src", *req.SourceID)
	assert.Equal(t, "fp-2", *req.SchemaFingerprint)
	assert.Equal(t, entities.FramingGRPC, req.Framing.Kind)
	assert.Equal(t, publishOutput{Encoding: encodingProtobuf, MessageType: "o.v1.Order"}, out)

	req, out, err = ts.request(t.Context(), publishInput{Subject: "orders.created", JSON: payload, Raw: true})
	require.NoError(t, err)
	assert.Nil(t, req.MessageType)
	assert.Equal(t, encodingRaw, out.Encoding)

	req, out, err = ts.request(t.Context(), publishInput{Subject: "refunds.new", JSON: payload, Type: "o.v1.Refund"})
	require.NoError(t, err)
	assert.Equal(t, "o.v1.Refund", *req.MessageType)
	assert.Nil(t, req.SchemaFingerprint)
	assert.Equal(t, encodingProtobuf, out.Encoding)

	req, out, err = ts.request(t.Context(), publishInput{Subject: "logs.app", Text: "plain line", Headers: map[string]string{"k": "v"}})
	require.NoError(t, err)
	assert.Equal(t, "plain line", req.Data)
	assert.Equal(t, map[string]string{"k": "v"}, req.Headers)
	assert.Equal(t, encodingRaw, out.Encoding)
}

func TestRequestRejectsAmbiguousPayloads(t *testing.T) {
	t.Parallel()
	ts := newToolset()

	tests := []struct {
		name string
		in   publishInput
		want string
	}{
		{name: "no payload", in: publishInput{Subject: "a"}, want: "pass the payload as json or text"},
		{name: "both payloads", in: publishInput{Subject: "a", JSON: map[string]any{}, Text: "x"}, want: "pass either json or text, not both"},
		{name: "raw with type", in: publishInput{Subject: "a", JSON: map[string]any{}, Type: "o.v1.Refund", Raw: true},
			want: "raw and type contradict each other; drop one"},
		{name: "type with text", in: publishInput{Subject: "a", Text: "x", Type: "o.v1.Refund"}, want: "type applies to a json payload; pass the message as json"},
		{name: "unknown type", in: publishInput{Subject: "a", JSON: map[string]any{}, Type: "o.v1.Missing"},
			want: `message type "o.v1.Missing" is not loaded; list_message_types shows the loaded ones`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := ts.request(t.Context(), tt.in)
			require.EqualError(t, err, tt.want)
		})
	}
}

func TestRequestMessageRejectsTimeoutOutOfRange(t *testing.T) {
	t.Parallel()
	ts := newToolset()

	for _, timeoutMs := range []int{-1, maxRequestTimeoutMs + 1} {
		_, _, err := ts.requestMessage(t.Context(), nil, requestInput{Subject: "svc.ping", TimeoutMs: timeoutMs})
		require.EqualError(t, err, "timeoutMs must be between 1 and 60000")
	}
}

func TestRegisterHonorsAllowWrites(t *testing.T) {
	t.Parallel()
	assert.True(t, newToolset().enabled)
	assert.False(t, New(&appconfig.Config{}, nil, nil, nil, nil).enabled)
}
