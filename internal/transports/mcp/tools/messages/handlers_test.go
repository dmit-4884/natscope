// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func TestNewMessageView(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		msg           entities.Message
		limit         int
		wantDecoded   string
		wantBody      *mcptransport.Body
		wantTruncated bool
	}{
		{
			name:        "decoded proto wins over raw bytes",
			msg:         entities.Message{DataBase64: b64("\x0a\x03abc"), Decoded: json.RawMessage(`{"name":"abc"}`), DecodedType: "t.v1.M", Truncated: true},
			limit:       1024,
			wantDecoded: `{"name":"abc"}`,
		},
		{
			name:          "decoded preview string stays truncated",
			msg:           entities.Message{Decoded: json.RawMessage(`"{\n  \"name\": …"`), Truncated: true},
			limit:         1024,
			wantDecoded:   `"{\n  \"name\": …"`,
			wantTruncated: true,
		},
		{
			name:          "decoded above the budget becomes clipped text",
			msg:           entities.Message{Decoded: json.RawMessage(`{"name":"abcdef"}`)},
			limit:         8,
			wantBody:      &mcptransport.Body{Text: `{"name":`},
			wantTruncated: true,
		},
		{
			name:     "raw json body",
			msg:      entities.Message{DataBase64: b64(`{"ok":true}`)},
			limit:    1024,
			wantBody: &mcptransport.Body{JSON: json.RawMessage(`{"ok":true}`)},
		},
		{
			name:          "upstream-clipped text",
			msg:           entities.Message{DataBase64: b64("hello wor"), Truncated: true},
			limit:         1024,
			wantBody:      &mcptransport.Body{Text: "hello wor"},
			wantTruncated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.msg.Sequence, tt.msg.Subject, tt.msg.Timestamp = 7, "orders.created", ts
			v := newMessageView(&tt.msg, tt.limit)

			assert.Equal(t, uint64(7), v.Sequence)
			assert.Equal(t, "orders.created", v.Subject)
			assert.Equal(t, ts, v.Timestamp)
			assert.Equal(t, tt.wantDecoded, string(v.Decoded))
			assert.Equal(t, tt.wantBody, v.Body)
			assert.Equal(t, tt.wantTruncated, v.Truncated)
		})
	}
}

func TestListRequest(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	req, err := listRequest(findMessagesInput{Stream: "ORDERS"}, now)
	require.NoError(t, err)
	assert.Equal(t, directionBackward, req.Direction)
	assert.Equal(t, int64(defaultPageSize), *req.Limit)
	assert.Equal(t, int32(defaultPagePayload), *req.MaxPayloadBytes)
	assert.Nil(t, req.StartSeq)
	assert.Nil(t, req.SubjectFilter)

	req, err = listRequest(findMessagesInput{Stream: "ORDERS", Since: "1h", Subject: "orders.>", Contains: "abc", Limit: 500}, now)
	require.NoError(t, err)
	assert.Equal(t, directionForward, req.Direction)
	assert.Equal(t, now.Add(-time.Hour), *req.StartTime)
	assert.Equal(t, "orders.>", *req.SubjectFilter)
	assert.Equal(t, int64(maxPageSize), *req.Limit)

	req, err = listRequest(findMessagesInput{Stream: "ORDERS", StartSeq: 40, Direction: "backward"}, now)
	require.NoError(t, err)
	assert.Equal(t, directionBackward, req.Direction)
	assert.Equal(t, uint64(40), *req.StartSeq)

	_, err = listRequest(findMessagesInput{StartSeq: 1, Since: "5m"}, now)
	require.EqualError(t, err, "pass either startSeq or since, not both")

	_, err = listRequest(findMessagesInput{Direction: "sideways"}, now)
	require.EqualError(t, err, `direction must be "forward" or "backward"`)
}

func TestNewLiveMessageView(t *testing.T) {
	t.Parallel()
	seq := uint64(3)
	nm := entities.NatsMessage{Subject: "orders.created", Data: []byte("\x0a\x01x"), Stream: "ORDERS", Sequence: &seq, Timestamp: time.Unix(1, 0)}

	decoded := `{"id":"x"}`
	v := newLiveMessageView(&entities.LiveMessage{NatsMessage: nm, Decoded: &decoded, DecodedType: new("t.v1.M")}, 1024)
	assert.Equal(t, decoded, string(v.Decoded))
	assert.Equal(t, "t.v1.M", v.DecodedType)
	assert.Nil(t, v.Body)
	assert.Equal(t, &seq, v.Sequence)
	assert.Equal(t, 3, v.Size)

	v = newLiveMessageView(&entities.LiveMessage{NatsMessage: entities.NatsMessage{Subject: "logs", Data: []byte("plain")}}, 1024)
	assert.Equal(t, &mcptransport.Body{Text: "plain"}, v.Body)
	assert.Nil(t, v.Timestamp)

	v = newLiveMessageView(&entities.LiveMessage{NatsMessage: nm, Decoded: &decoded}, 4)
	assert.Nil(t, v.Decoded)
	assert.Equal(t, &mcptransport.Body{Text: `{"id`}, v.Body)
	assert.True(t, v.Truncated)
}

func TestTailCollectorStopsAtMax(t *testing.T) {
	t.Parallel()
	c := &tailCollector{out: tailOutput{Messages: []liveMessageView{}}, max: 2, limit: 64}
	batch := &entities.LiveEvent{Batch: &entities.LiveBatch{Messages: []*entities.LiveMessage{
		{NatsMessage: entities.NatsMessage{Subject: "a", Data: []byte("1")}},
		{NatsMessage: entities.NatsMessage{Subject: "b", Data: []byte("2")}},
		{NatsMessage: entities.NatsMessage{Subject: "c", Data: []byte("3")}},
	}}}

	require.ErrorIs(t, c.emit(batch), errTailFull)
	require.NoError(t, c.emit(&entities.LiveEvent{Stats: &entities.LiveStats{MessagesDropped: 4}}))
	require.NoError(t, c.emit(&entities.LiveEvent{Error: &entities.LiveError{Message: "partial"}}))

	out := c.result()
	assert.Len(t, out.Messages, 2)
	assert.Equal(t, int64(4), out.Dropped)
	assert.Equal(t, []string{"partial"}, out.Errors)
	assert.Equal(t, "maxMessages", out.StoppedBy)
}

func TestTailCollectorReportsTimeoutBelowMax(t *testing.T) {
	t.Parallel()
	c := &tailCollector{out: tailOutput{Messages: []liveMessageView{}}, max: 5, limit: 64}
	require.NoError(t, c.emit(&entities.LiveEvent{Batch: &entities.LiveBatch{Messages: []*entities.LiveMessage{
		{NatsMessage: entities.NatsMessage{Subject: "a", Data: []byte("1")}},
	}}}))
	assert.Equal(t, "timeout", c.result().StoppedBy)
}

func TestPayloadLimitHonorsResponseBudget(t *testing.T) {
	t.Parallel()
	assert.Equal(t, defaultPagePayload, payloadLimit(0, defaultPageSize))
	assert.Equal(t, 1000, payloadLimit(1000, maxPageSize))
	assert.Equal(t, responsePayloadBudget/maxPageSize, payloadLimit(maxPayload, maxPageSize))
	assert.Equal(t, minMessagePayload, payloadLimit(maxPayload, 10_000))
	assert.Equal(t, maxPayload, payloadLimit(10*maxPayload, 1))
}

func TestSearchRequest(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	req, ok, err := searchRequest(findMessagesInput{Stream: "ORDERS", Contains: "needle", Limit: 7}, now)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "needle", req.Text)
	assert.False(t, req.Regex)
	assert.Equal(t, directionBackward, req.Direction)
	assert.Equal(t, 7, req.MaxMatches)
	assert.Nil(t, req.CursorSeq)

	req, ok, err = searchRequest(findMessagesInput{
		Stream: "ORDERS", Contains: `order-\d+`, Regex: true, Header: "X-Trace = abc", StartSeq: 40, Since: "1h", Subject: "orders.>",
	}, now)
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, req.Regex)
	assert.Equal(t, "X-Trace", req.HeaderName)
	assert.Equal(t, "abc", req.HeaderValue)
	assert.Equal(t, uint64(40), *req.CursorSeq)
	assert.Equal(t, now.Add(-time.Hour), *req.FromTime)
	assert.Equal(t, "orders.>", req.SubjectFilter)
	assert.Equal(t, directionForward, req.Direction)

	req, ok, err = searchRequest(findMessagesInput{Stream: "ORDERS", Header: "X-Trace"}, now)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "X-Trace", req.HeaderName)
	assert.Empty(t, req.HeaderValue)

	_, ok, err = searchRequest(findMessagesInput{Stream: "ORDERS", Subject: "orders.>"}, now)
	require.NoError(t, err)
	assert.False(t, ok, "without contains or header a plain page is read")

	_, _, err = searchRequest(findMessagesInput{Stream: "ORDERS", Regex: true}, now)
	require.EqualError(t, err, "regex needs contains")

	req, _, err = searchRequest(findMessagesInput{Stream: "ORDERS", Contains: ` total \d+ `, Regex: true}, now)
	require.NoError(t, err)
	assert.Equal(t, ` total \d+ `, req.Text, "a regular expression is kept as written")
}
