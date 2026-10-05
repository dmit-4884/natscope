// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func TestParseMicroInfo(t *testing.T) {
	t.Parallel()

	raw := `{
		"name": "orders", "id": "i-1", "version": "1.2.0", "metadata": {"team": "core"},
		"type": "io.nats.micro.v1.info_response", "description": "Order API",
		"endpoints": [{"name": "Create", "subject": "orders.create", "queue_group": "q", "metadata": {"k": "v"}}]
	}`
	report, ok := parseMicroInfo([]byte(raw))
	require.True(t, ok)
	assert.Equal(t, entities.MicroReport{
		Name:        "orders",
		ID:          "i-1",
		Version:     "1.2.0",
		Description: "Order API",
		Metadata:    map[string]string{"team": "core"},
		Endpoints: []entities.MicroEndpoint{
			{Name: "Create", Subject: "orders.create", QueueGroup: "q", Metadata: map[string]string{"k": "v"}},
		},
		Raw: raw,
	}, report)
}

func TestParseMicroStats(t *testing.T) {
	t.Parallel()

	report, ok := parseMicroStats([]byte(`{
		"name": "orders", "id": "i-1", "version": "1.2.0",
		"type": "io.nats.micro.v1.stats_response", "started": "2026-10-05T10:00:00Z",
		"endpoints": [{"name": "Create", "subject": "orders.create", "queue_group": "q",
			"num_requests": 10, "num_errors": 2, "last_error": "boom",
			"processing_time": 5000000, "average_processing_time": 500000}]
	}`))
	require.True(t, ok)
	assert.Equal(t, "orders", report.Name)
	assert.Equal(t, "i-1", report.ID)
	assert.Equal(t, time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC), report.Started.UTC())
	require.Len(t, report.Endpoints, 1)
	assert.Equal(t, &entities.MicroEndpointStats{
		NumRequests:           10,
		NumErrors:             2,
		LastError:             "boom",
		ProcessingTime:        5 * time.Millisecond,
		AverageProcessingTime: 500 * time.Microsecond,
	}, report.Endpoints[0].Stats)
}

func TestParseMicroRejectsMalformedReplies(t *testing.T) {
	t.Parallel()

	_, ok := parseMicroInfo([]byte(`not json`))
	assert.False(t, ok)
	_, ok = parseMicroStats([]byte(`{"id": "no name", "type": "io.nats.micro.v1.stats_response"}`))
	assert.False(t, ok, "a reply without a service name is not a service")
}

func TestParseMicroIgnoresRepliesOfAnotherType(t *testing.T) {
	t.Parallel()

	_, ok := parseMicroInfo([]byte(`{"name": "orders", "id": "i-1", "type": "io.nats.micro.v1.stats_response"}`))
	assert.False(t, ok)
	_, ok = parseMicroInfo([]byte(`{"name": "orders", "id": "i-1"}`))
	assert.False(t, ok, "a reply without a type is not a service answer")
	_, ok = parseMicroStats([]byte(`{"name": "orders", "id": "i-1", "type": "io.nats.micro.v1.info_response"}`))
	assert.False(t, ok)
}

func TestParseMicroStatsAcceptsFractionalNumbers(t *testing.T) {
	t.Parallel()

	report, ok := parseMicroStats([]byte(`{
		"name": "orders", "id": "i-1", "type": "io.nats.micro.v1.stats_response", "started": "2026-10-05T10:00:00.123Z",
		"endpoints": [{"name": "Create", "subject": "orders.create",
			"num_requests": 3, "num_errors": 0, "processing_time": 3700,
			"average_processing_time": 1233.3333333333333, "data": {"cache_hits": 7}}]
	}`))
	require.True(t, ok)
	require.Len(t, report.Endpoints, 1)
	assert.Equal(t, int64(3), report.Endpoints[0].Stats.NumRequests)
	assert.Equal(t, 1233*time.Nanosecond, report.Endpoints[0].Stats.AverageProcessingTime)
	assert.Contains(t, report.Raw, `"cache_hits": 7`)
}

func TestParseMicroInfoKeepsTheRawReply(t *testing.T) {
	t.Parallel()

	raw := `{"name": "orders", "id": "i-1", "type": "io.nats.micro.v1.info_response", "endpoints": []}`
	report, ok := parseMicroInfo([]byte(raw))
	require.True(t, ok)
	assert.Equal(t, raw, report.Raw)
}
