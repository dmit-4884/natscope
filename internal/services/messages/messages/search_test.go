// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func payloadMessage(payload string, headers map[string]string) *entities.Message {
	return &entities.Message{DataBase64: base64.StdEncoding.EncodeToString([]byte(payload)), Headers: headers}
}

func headerMessage(values map[string][]string) *entities.Message {
	joined := make(map[string]string, len(values))
	for name, v := range values {
		joined[name] = strings.Join(v, ", ")
	}
	return &entities.Message{Headers: joined, HeaderValues: values}
}

func TestSearchMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  entities.MessageSearchRequest
		msg  *entities.Message
		want bool
	}{
		{name: "text ignores case", req: entities.MessageSearchRequest{Text: "NEEDLE"}, msg: payloadMessage(`{"note":"a needle"}`, nil), want: true},
		{name: "text missing", req: entities.MessageSearchRequest{Text: "needle"}, msg: payloadMessage(`{"note":"hay"}`, nil)},
		{name: "regex matches as written", req: entities.MessageSearchRequest{Text: `order-\d{3}`, Regex: true}, msg: payloadMessage(`order-123`, nil), want: true},
		{name: "regex keeps case", req: entities.MessageSearchRequest{Text: `Order`, Regex: true}, msg: payloadMessage(`order`, nil)},
		{name: "header name ignores case", req: entities.MessageSearchRequest{HeaderName: "x-trace"}, msg: payloadMessage("", map[string]string{"X-Trace": "abc"}), want: true},
		{name: "header value must match", req: entities.MessageSearchRequest{HeaderName: "X-Trace", HeaderValue: "abc"}, msg: payloadMessage("", map[string]string{"X-Trace": "abd"})},
		{name: "header value matches one of several", req: entities.MessageSearchRequest{HeaderName: "X-Tag", HeaderValue: "b"}, msg: headerMessage(map[string][]string{"X-Tag": {"a", "b"}}), want: true},
		{name: "header value is not a part of one value", req: entities.MessageSearchRequest{HeaderName: "X-Tag", HeaderValue: "b"}, msg: headerMessage(map[string][]string{"X-Tag": {"a, b"}})},
		{name: "header value with a comma matches whole", req: entities.MessageSearchRequest{HeaderName: "X-Tag", HeaderValue: "a, b"}, msg: headerMessage(map[string][]string{"X-Tag": {"a, b"}}), want: true},
		{name: "missing header", req: entities.MessageSearchRequest{HeaderName: "X-Trace"}, msg: payloadMessage("", nil)},
		{name: "every condition must hold", req: entities.MessageSearchRequest{Text: "needle", HeaderName: "X-Trace"}, msg: payloadMessage("needle", nil)},
		{name: "no condition matches all", req: entities.MessageSearchRequest{}, msg: payloadMessage("anything", nil), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := newSearchMatcher(&tt.req)
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.matchHeaders(tt.msg) && m.matchPayload(tt.msg, func() string { return "" }))
		})
	}
}

func TestSearchMatcherLooksInsideDecodedPayloads(t *testing.T) {
	t.Parallel()
	m, err := newSearchMatcher(&entities.MessageSearchRequest{Text: `"amount": 42`})
	require.NoError(t, err)

	decoded := 0
	msg := payloadMessage("\x08\x2a", nil)
	assert.True(t, m.matchPayload(msg, func() string { decoded++; return `{"amount": 42}` }))
	assert.Equal(t, 1, decoded)

	assert.True(t, m.matchPayload(payloadMessage(`{"amount": 42}`, nil), func() string { decoded++; return "" }))
	assert.Equal(t, 1, decoded, "a payload that matches as stored is not decoded")
}

func TestSearchMatcherRejectsBadRegex(t *testing.T) {
	t.Parallel()
	_, err := newSearchMatcher(&entities.MessageSearchRequest{Text: "order-(", Regex: true})
	var validation *errs.NATSValidationError
	require.True(t, errors.As(err, &validation))
	assert.Contains(t, validation.Error(), "regular expression")
}

func TestSearchMatcherDecodesProtobufThatLooksLikeText(t *testing.T) {
	t.Parallel()
	m, err := newSearchMatcher(&entities.MessageSearchRequest{Text: `"id"`})
	require.NoError(t, err)

	msg := payloadMessage("\n$0b5f1b9e-7a64-4f4a-9a43-4d1b2a8c1f00", nil)
	msg.ContentType = entities.ContentTypeText
	assert.True(t, m.matchPayload(msg, func() string { return `{"id":"0b5f1b9e"}` }))
}

func TestSearchMatcherRejectsAValueWithoutAHeaderName(t *testing.T) {
	t.Parallel()
	_, err := newSearchMatcher(&entities.MessageSearchRequest{HeaderName: "   ", HeaderValue: "abc"})
	var validation *errs.NATSValidationError
	require.True(t, errors.As(err, &validation))
}

func TestSearchMatcherRejectsAnInvalidSubjectFilter(t *testing.T) {
	t.Parallel()
	for _, filter := range []string{"edge..hit", "edge.>.hit", "orders. created"} {
		_, err := newSearchMatcher(&entities.MessageSearchRequest{SubjectFilter: filter})
		var validation *errs.NATSValidationError
		require.True(t, errors.As(err, &validation), "filter %q", filter)
	}
	_, err := newSearchMatcher(&entities.MessageSearchRequest{SubjectFilter: "orders.*.created"})
	require.NoError(t, err)
}
