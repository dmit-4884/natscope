// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func TestStreamViewRendersEnumsAndDurations(t *testing.T) {
	t.Parallel()
	info := &entities.StreamInfo{
		Config: entities.StreamConfig{
			Name: "ORDERS", Subjects: []string{"orders.>"}, Retention: entities.RetentionWorkQueue,
			Storage: entities.StorageMemory, Discard: entities.DiscardNew, MaxAge: 36 * time.Hour,
			Compression: entities.CompressionS2, Mirror: &entities.StreamSourceRef{Name: "UPSTREAM"},
		},
		State: &entities.StreamState{Msgs: 12, LastSeq: 40},
	}

	v := converter.Convert(info, &streamView{}, mcptransport.ViewCodecs)

	assert.Equal(t, "workqueue", v.Config.Retention)
	assert.Equal(t, "memory", v.Config.Storage)
	assert.Equal(t, "new", v.Config.Discard)
	assert.Equal(t, "s2", v.Config.Compression)
	assert.Equal(t, "36h0m0s", v.Config.MaxAge)
	require.NotNil(t, v.Config.Mirror)
	assert.Equal(t, "UPSTREAM", v.Config.Mirror.Name)
	require.NotNil(t, v.State)
	assert.Equal(t, uint64(12), v.State.Msgs)

	summary := converter.Convert(info, &streamSummary{}, mcptransport.ViewCodecs)
	assert.Equal(t, "workqueue", summary.Config.Retention)
	assert.Equal(t, []string{"orders.>"}, summary.Config.Subjects)
}

func TestConsumerViewMergesConfig(t *testing.T) {
	t.Parallel()
	info := entities.ConsumerInfo{
		Name: "worker", Stream: "ORDERS", NumPending: 5, NumAckPending: 2,
		Config: &entities.ConsumerConfig{Name: "ignored", Durable: "worker", AckPolicy: entities.AckAll,
			DeliverPolicy: entities.DeliverLastPerSubject, AckWait: 30 * time.Second, FilterSubject: "orders.eu.>"},
	}

	v := converter.Convert(&info, &consumerView{}, mcptransport.ViewCodecs)
	v = converter.Convert(info.Config, v, mcptransport.ViewCodecs, converter.WithIgnoreFields("Name"))

	assert.Equal(t, "worker", v.Name)
	assert.Equal(t, "ORDERS", v.Stream)
	assert.Equal(t, uint64(5), v.NumPending)
	assert.Equal(t, "all", v.AckPolicy)
	assert.Equal(t, "last_per_subject", v.DeliverPolicy)
	assert.Equal(t, "30s", v.AckWait)
	assert.Equal(t, "orders.eu.>", v.FilterSubject)

	stats := converter.Convert(&entities.ConsumerStats{Name: "w2", Stream: "S", AckPolicy: entities.AckNone}, &consumerView{}, mcptransport.ViewCodecs)
	assert.Equal(t, "none", stats.AckPolicy)
	assert.Equal(t, "all", stats.DeliverPolicy)
}
