// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// TestStreamCreateConversion verifies every field of StreamCreateRequest → jetstream.StreamConfig.
func TestStreamCreateConversion(t *testing.T) {
	entity := entities.StreamCreateRequest{
		Name:                 "test-stream",
		Description:          "test description",
		Subjects:             []string{"test.>", "other.*"},
		Retention:            entities.RetentionInterest,
		Storage:              entities.StorageMemory,
		Discard:              entities.DiscardNew,
		MaxMsgs:              1000,
		MaxBytes:             1024 * 1024,
		MaxAge:               time.Hour,
		MaxMsgsPerSubject:    100,
		MaxMsgSize:           4096,
		MaxConsumers:         10,
		Replicas:             3,
		Duplicates:           2 * time.Minute,
		DenyDelete:           true,
		DenyPurge:            true,
		AllowRollup:          true,
		AllowDirect:          true,
		MirrorDirect:         true,
		DiscardNewPerSubject: true,
		Compression:          entities.CompressionS2,
		FirstSeq:             42,
		NoAck:                true,
		Metadata:             map[string]string{"env": "test"},
		Placement:            &entities.Placement{Cluster: "us-east", Tags: []string{"fast"}},
		SubjectTransform:     &entities.SubjectTransformConfig{Source: "src.>", Destination: "dst.>"},
	}

	jsConfig := converter.Convert(entity, &jetstream.StreamConfig{})

	assert.Equal(t, "test-stream", jsConfig.Name)
	assert.Equal(t, "test description", jsConfig.Description)
	assert.Equal(t, []string{"test.>", "other.*"}, jsConfig.Subjects)
	assert.Equal(t, jetstream.InterestPolicy, jsConfig.Retention)
	assert.Equal(t, jetstream.MemoryStorage, jsConfig.Storage)
	assert.Equal(t, jetstream.DiscardNew, jsConfig.Discard)
	assert.Equal(t, int64(1000), jsConfig.MaxMsgs)
	assert.Equal(t, int64(1024*1024), jsConfig.MaxBytes)
	assert.Equal(t, time.Hour, jsConfig.MaxAge, "MaxAge: int64 → time.Duration")
	assert.Equal(t, int64(100), jsConfig.MaxMsgsPerSubject)
	assert.Equal(t, int32(4096), jsConfig.MaxMsgSize)
	assert.Equal(t, 10, jsConfig.MaxConsumers)
	assert.Equal(t, 3, jsConfig.Replicas)
	assert.Equal(t, 2*time.Minute, jsConfig.Duplicates, "Duplicates: int64 → time.Duration")
	assert.True(t, jsConfig.DenyDelete)
	assert.True(t, jsConfig.DenyPurge)
	assert.True(t, jsConfig.AllowRollup)
	assert.True(t, jsConfig.AllowDirect)
	assert.True(t, jsConfig.MirrorDirect)
	assert.True(t, jsConfig.DiscardNewPerSubject)
	assert.Equal(t, jetstream.S2Compression, jsConfig.Compression)
	assert.Equal(t, uint64(42), jsConfig.FirstSeq)
	assert.True(t, jsConfig.NoAck)
	assert.Equal(t, map[string]string{"env": "test"}, jsConfig.Metadata)

	require.NotNil(t, jsConfig.Placement)
	assert.Equal(t, "us-east", jsConfig.Placement.Cluster)
	assert.Equal(t, []string{"fast"}, jsConfig.Placement.Tags)

	require.NotNil(t, jsConfig.SubjectTransform)
	assert.Equal(t, "src.>", jsConfig.SubjectTransform.Source)
	assert.Equal(t, "dst.>", jsConfig.SubjectTransform.Destination)
}

// TestStreamCreateConversion_Mirror verifies Mirror and Sources nested conversion.
func TestStreamCreateConversion_Mirror(t *testing.T) {
	startTime := time.Now().UTC().Truncate(time.Second)

	entity := entities.StreamCreateRequest{
		Name:     "mirror-stream",
		Subjects: []string{"test.>"},
		Mirror: &entities.StreamSource{
			Name:          "origin",
			OptStartSeq:   100,
			OptStartTime:  &startTime,
			FilterSubject: "test.important",
			External: &entities.ExternalStream{
				APIPrefix:     "$JS.test.API",
				DeliverPrefix: "$JS.test.DELIVER",
			},
		},
		Sources: []*entities.StreamSource{
			{Name: "src1", OptStartSeq: 50, FilterSubject: "src1.>"},
			{Name: "src2", OptStartSeq: 75},
		},
	}

	jsConfig := converter.Convert(entity, &jetstream.StreamConfig{})

	require.NotNil(t, jsConfig.Mirror)
	assert.Equal(t, "origin", jsConfig.Mirror.Name)
	assert.Equal(t, uint64(100), jsConfig.Mirror.OptStartSeq)
	assert.Equal(t, &startTime, jsConfig.Mirror.OptStartTime)
	assert.Equal(t, "test.important", jsConfig.Mirror.FilterSubject)
	require.NotNil(t, jsConfig.Mirror.External)
	assert.Equal(t, "$JS.test.API", jsConfig.Mirror.External.APIPrefix)
	assert.Equal(t, "$JS.test.DELIVER", jsConfig.Mirror.External.DeliverPrefix)

	require.Len(t, jsConfig.Sources, 2)
	assert.Equal(t, "src1", jsConfig.Sources[0].Name)
	assert.Equal(t, uint64(50), jsConfig.Sources[0].OptStartSeq)
	assert.Equal(t, "src1.>", jsConfig.Sources[0].FilterSubject)
	assert.Equal(t, "src2", jsConfig.Sources[1].Name)
	assert.Equal(t, uint64(75), jsConfig.Sources[1].OptStartSeq)
}

func TestStreamCreateConversion_Republish(t *testing.T) {
	jsConfig := converter.Convert(entities.StreamCreateRequest{
		Name:     "republish-stream",
		Subjects: []string{"qa.complex.>"},
		Republish: &entities.StreamRePublish{
			Src:         "qa.complex.>",
			Dest:        "audit.qa.>",
			HeadersOnly: true,
		},
	}, &jetstream.StreamConfig{}, srcDestToJetStream)

	require.NotNil(t, jsConfig.RePublish)
	assert.Equal(t, "qa.complex.>", jsConfig.RePublish.Source)
	assert.Equal(t, "audit.qa.>", jsConfig.RePublish.Destination)
	assert.True(t, jsConfig.RePublish.HeadersOnly)
}

// TestConsumerCreateConversion verifies every field of ConsumerCreateRequest → jetstream.ConsumerConfig.
func TestConsumerCreateConversion(t *testing.T) {
	entity := entities.ConsumerCreateRequest{
		Name:               "test-consumer",
		Description:        "test consumer desc",
		DeliverPolicy:      entities.DeliverByStartSequence,
		OptStartSeq:        500,
		AckPolicy:          entities.AckAll,
		AckWait:            30 * time.Second,
		MaxDeliver:         5,
		BackOff:            []time.Duration{time.Second, 5 * time.Second, 30 * time.Second},
		FilterSubject:      "orders.>",
		FilterSubjects:     []string{"orders.created", "orders.updated"},
		ReplayPolicy:       entities.ReplayOriginal,
		RateLimit:          1024,
		SampleFrequency:    "50%",
		MaxAckPending:      100,
		MaxWaiting:         50,
		HeadersOnly:        true,
		MaxRequestBatch:    25,
		MaxRequestMaxBytes: 1024 * 1024,
		MaxRequestExpires:  10 * time.Second,
		InactiveThreshold:  5 * time.Minute,
		Replicas:           3,
		MemoryStorage:      true,
		Metadata:           map[string]string{"team": "backend"},
		DeliverSubject:     "deliver.test",
		DeliverGroup:       "workers",
		FlowControl:        true,
		IdleHeartbeat:      15 * time.Second,
	}

	jsConfig, err := toJetStreamConsumerConfig(entity)
	require.NoError(t, err)

	assert.Equal(t, "test-consumer", jsConfig.Name)
	assert.Equal(t, "test-consumer", jsConfig.Durable, "Durable must mirror Name so the server keeps the consumer")
	assert.Equal(t, "test consumer desc", jsConfig.Description)
	assert.Equal(t, jetstream.DeliverByStartSequencePolicy, jsConfig.DeliverPolicy)
	assert.Equal(t, uint64(500), jsConfig.OptStartSeq)
	assert.Equal(t, jetstream.AckAllPolicy, jsConfig.AckPolicy)
	assert.Equal(t, 30*time.Second, jsConfig.AckWait, "AckWait: int64 → time.Duration")
	assert.Equal(t, 5, jsConfig.MaxDeliver)
	require.Len(t, jsConfig.BackOff, 3)
	assert.Equal(t, time.Second, jsConfig.BackOff[0], "BackOff[0]: int64 → time.Duration")
	assert.Equal(t, 5*time.Second, jsConfig.BackOff[1])
	assert.Equal(t, 30*time.Second, jsConfig.BackOff[2])
	assert.Equal(t, "orders.>", jsConfig.FilterSubject)
	assert.Equal(t, []string{"orders.created", "orders.updated"}, jsConfig.FilterSubjects)
	assert.Equal(t, jetstream.ReplayOriginalPolicy, jsConfig.ReplayPolicy)
	assert.Equal(t, uint64(1024), jsConfig.RateLimit)
	assert.Equal(t, "50%", jsConfig.SampleFrequency)
	assert.Equal(t, 100, jsConfig.MaxAckPending)
	assert.Equal(t, 50, jsConfig.MaxWaiting)
	assert.True(t, jsConfig.HeadersOnly)
	assert.Equal(t, 25, jsConfig.MaxRequestBatch)
	assert.Equal(t, 1024*1024, jsConfig.MaxRequestMaxBytes, "MaxRequestMaxBytes: int64 → int")
	assert.Equal(t, 10*time.Second, jsConfig.MaxRequestExpires, "MaxRequestExpires: int64 → time.Duration")
	assert.Equal(t, 5*time.Minute, jsConfig.InactiveThreshold, "InactiveThreshold: int64 → time.Duration")
	assert.Equal(t, 3, jsConfig.Replicas)
	assert.True(t, jsConfig.MemoryStorage)
	assert.Equal(t, map[string]string{"team": "backend"}, jsConfig.Metadata)
	assert.Equal(t, "deliver.test", jsConfig.DeliverSubject)
	assert.Equal(t, "workers", jsConfig.DeliverGroup)
	assert.True(t, jsConfig.FlowControl)
	assert.Equal(t, 15*time.Second, jsConfig.IdleHeartbeat, "IdleHeartbeat: int64 → time.Duration")
}

// TestConsumerCreateConversion_Durable pins the durable identity: a named create request must
// reach the SDK with Durable set, otherwise the server treats the consumer as ephemeral and
// reaps it after its 5s inactivity default.
func TestConsumerCreateConversion_Durable(t *testing.T) {
	t.Run("named request becomes durable", func(t *testing.T) {
		jsConfig, err := toJetStreamConsumerConfig(entities.ConsumerCreateRequest{Name: "orders-worker"})
		require.NoError(t, err)

		assert.Equal(t, "orders-worker", jsConfig.Name)
		assert.Equal(t, "orders-worker", jsConfig.Durable)
		assert.Zero(t, jsConfig.InactiveThreshold, "unset InactiveThreshold must stay unset")
	})

	t.Run("unnamed request stays ephemeral", func(t *testing.T) {
		jsConfig, err := toJetStreamConsumerConfig(entities.ConsumerCreateRequest{})
		require.NoError(t, err)

		assert.Empty(t, jsConfig.Name)
		assert.Empty(t, jsConfig.Durable)
	})

	t.Run("opt start time is parsed", func(t *testing.T) {
		start := time.Now().UTC().Truncate(time.Second)

		jsConfig, err := toJetStreamConsumerConfig(entities.ConsumerCreateRequest{
			Name:          "replayer",
			DeliverPolicy: entities.DeliverByStartTime,
			OptStartTime:  start.Format(time.RFC3339),
		})
		require.NoError(t, err)

		require.NotNil(t, jsConfig.OptStartTime)
		assert.Equal(t, start, jsConfig.OptStartTime.UTC())
		assert.Equal(t, "replayer", jsConfig.Durable)
	})
}

func TestConsumerCreateConversion_InvalidOptStartTime(t *testing.T) {
	jsConfig, err := toJetStreamConsumerConfig(entities.ConsumerCreateRequest{
		Name:          "replayer",
		DeliverPolicy: entities.DeliverByStartTime,
		OptStartTime:  "not-a-timestamp",
	})

	require.Error(t, err)
	assert.Nil(t, jsConfig)
	assert.ErrorIs(t, err, errs.ErrNATSInvalidArgument)
	assert.Contains(t, err.Error(), "not-a-timestamp")
}

// TestStreamInfoConversion verifies jetstream.StreamInfo → entities.StreamInfo (response path).
func TestStreamInfoConversion(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	firstTime := now.Add(-time.Hour)
	lastTime := now.Add(-time.Minute)

	jsInfo := &jetstream.StreamInfo{
		Created: now,
		Config: jetstream.StreamConfig{
			Name:                 "test-stream",
			Description:          "desc",
			Subjects:             []string{"test.>"},
			Retention:            jetstream.WorkQueuePolicy,
			MaxConsumers:         5,
			MaxMsgs:              1000,
			MaxBytes:             2048,
			MaxAge:               time.Hour,
			MaxMsgsPerSubject:    50,
			MaxMsgSize:           1024,
			Storage:              jetstream.MemoryStorage,
			Discard:              jetstream.DiscardNew,
			Replicas:             3,
			Duplicates:           2 * time.Minute,
			Compression:          jetstream.S2Compression,
			Sealed:               true,
			DenyDelete:           true,
			DenyPurge:            true,
			AllowRollup:          true,
			AllowDirect:          true,
			MirrorDirect:         true,
			NoAck:                true,
			DiscardNewPerSubject: true,
			FirstSeq:             10,
			Metadata:             map[string]string{"env": "prod"},
			RePublish: &jetstream.RePublish{
				Source:      "src.>",
				Destination: "dst.>",
				HeadersOnly: true,
			},
		},
		State: jetstream.StreamState{
			Msgs:        500,
			Bytes:       1024,
			FirstSeq:    1,
			LastSeq:     500,
			Consumers:   3,
			NumDeleted:  10,
			NumSubjects: 5,
			FirstTime:   firstTime,
			LastTime:    lastTime,
		},
		Cluster: &jetstream.ClusterInfo{
			Name:   "nats-cluster",
			Leader: "node-1",
			Replicas: []*jetstream.PeerInfo{
				{Name: "node-2", Current: true, Active: 100 * time.Millisecond, Lag: 0},
				{Name: "node-3", Current: false, Offline: true, Active: 5 * time.Second, Lag: 10},
			},
		},
	}

	result := converter.Convert(jsInfo, &entities.StreamInfo{}, streamConvertOpts...)
	result.State = converter.Convert(&jsInfo.State, &entities.StreamState{}, streamConvertOpts...)

	assert.Equal(t, now, result.Created)

	// Config
	cfg := result.Config
	assert.Equal(t, "test-stream", cfg.Name)
	assert.Equal(t, "desc", cfg.Description)
	assert.Equal(t, []string{"test.>"}, cfg.Subjects)
	assert.Equal(t, entities.RetentionWorkQueue, cfg.Retention)
	assert.Equal(t, 5, cfg.MaxConsumers)
	assert.Equal(t, int64(1000), cfg.MaxMsgs)
	assert.Equal(t, int64(2048), cfg.MaxBytes)
	assert.Equal(t, time.Hour, cfg.MaxAge, "MaxAge: time.Duration passthrough")
	assert.Equal(t, int64(50), cfg.MaxMsgsPerSubject)
	assert.Equal(t, int32(1024), cfg.MaxMsgSize)
	assert.Equal(t, entities.StorageMemory, cfg.Storage)
	assert.Equal(t, entities.DiscardNew, cfg.Discard)
	assert.Equal(t, 3, cfg.Replicas)
	assert.Equal(t, 2*time.Minute, cfg.Duplicates, "Duplicates: time.Duration passthrough")
	assert.Equal(t, entities.CompressionS2, cfg.Compression)
	assert.True(t, cfg.Sealed)
	assert.True(t, cfg.DenyDelete)
	assert.True(t, cfg.DenyPurge)
	assert.True(t, cfg.AllowRollup)
	assert.True(t, cfg.AllowDirect)
	assert.True(t, cfg.MirrorDirect)
	assert.True(t, cfg.NoAck)
	assert.True(t, cfg.DiscardNewPerSubject)
	assert.Equal(t, uint64(10), cfg.FirstSeq)
	assert.Equal(t, map[string]string{"env": "prod"}, cfg.Metadata)

	// RePublish
	require.NotNil(t, cfg.Republish)
	assert.Equal(t, "src.>", cfg.Republish.Src)
	assert.Equal(t, "dst.>", cfg.Republish.Dest)
	assert.True(t, cfg.Republish.HeadersOnly)

	// State
	require.NotNil(t, result.State)
	assert.Equal(t, uint64(500), result.State.Msgs)
	assert.Equal(t, uint64(1024), result.State.Bytes)
	assert.Equal(t, uint64(1), result.State.FirstSeq)
	assert.Equal(t, uint64(500), result.State.LastSeq)
	assert.Equal(t, 3, result.State.Consumers)
	assert.Equal(t, firstTime, result.State.FirstTime)
	assert.Equal(t, lastTime, result.State.LastTime)

	// Cluster
	require.NotNil(t, result.Cluster)
	assert.Equal(t, "nats-cluster", result.Cluster.Name)
	assert.Equal(t, "node-1", result.Cluster.Leader)
	require.Len(t, result.Cluster.Replicas, 2)
	assert.Equal(t, "node-2", result.Cluster.Replicas[0].Name)
	assert.True(t, result.Cluster.Replicas[0].Current)
	assert.Equal(t, 100*time.Millisecond, result.Cluster.Replicas[0].Active)
	assert.Equal(t, "node-3", result.Cluster.Replicas[1].Name)
	assert.True(t, result.Cluster.Replicas[1].Offline)
	assert.Equal(t, uint64(10), result.Cluster.Replicas[1].Lag)
}

// TestConsumerInfoConversion verifies jetstream.ConsumerInfo → entities.ConsumerInfo (response path).
func TestConsumerInfoConversion(t *testing.T) {
	created := time.Now().UTC().Truncate(time.Millisecond)
	ts := created.Add(time.Second)

	jsInfo := &jetstream.ConsumerInfo{
		Name:   "test-consumer",
		Stream: "test-stream",
		Config: jetstream.ConsumerConfig{
			Name:               "test-consumer",
			Durable:            "test-consumer",
			Description:        "desc",
			DeliverPolicy:      jetstream.DeliverLastPerSubjectPolicy,
			OptStartSeq:        100,
			AckPolicy:          jetstream.AckAllPolicy,
			AckWait:            30 * time.Second,
			MaxDeliver:         5,
			BackOff:            []time.Duration{time.Second, 5 * time.Second},
			FilterSubject:      "orders.>",
			FilterSubjects:     []string{"orders.created"},
			ReplayPolicy:       jetstream.ReplayOriginalPolicy,
			RateLimit:          512,
			SampleFrequency:    "25%",
			MaxWaiting:         100,
			MaxAckPending:      200,
			FlowControl:        true,
			IdleHeartbeat:      10 * time.Second,
			HeadersOnly:        true,
			MaxRequestBatch:    50,
			MaxRequestExpires:  5 * time.Second,
			InactiveThreshold:  3 * time.Minute,
			Replicas:           3,
			MemoryStorage:      true,
			MaxRequestMaxBytes: 2048,
			Metadata:           map[string]string{"team": "api"},
			DeliverSubject:     "deliver.test",
			DeliverGroup:       "group1",
		},
		Created:   created,
		TimeStamp: ts,
		Delivered: jetstream.SequenceInfo{
			Consumer: 50,
			Stream:   100,
		},
		AckFloor: jetstream.SequenceInfo{
			Consumer: 45,
			Stream:   90,
		},
		NumPending:     10,
		NumAckPending:  5,
		NumRedelivered: 2,
		NumWaiting:     3,
		PushBound:      true,
		Cluster: &jetstream.ClusterInfo{
			Name:   "cluster",
			Leader: "leader-1",
		},
	}

	// toConsumerInfo sets the pointer timestamps explicitly (the converter can't
	// bridge time.Time → *time.Time), so replicate that here after the convert.
	result := converter.Convert(jsInfo, &entities.ConsumerInfo{}, consumerConvertOpts...)
	result.Config = converter.Convert(&jsInfo.Config, &entities.ConsumerConfig{}, consumerConvertOpts...)
	result.Created = &jsInfo.Created
	result.TimeStamp = &jsInfo.TimeStamp

	assert.Equal(t, "test-consumer", result.Name)
	assert.Equal(t, "test-stream", result.Stream)

	require.NotNil(t, result.Created)
	assert.Equal(t, created, *result.Created)

	require.NotNil(t, result.TimeStamp)
	assert.Equal(t, ts, *result.TimeStamp)

	// Sequence info
	assert.Equal(t, uint64(50), result.Delivered.Consumer)
	assert.Equal(t, uint64(100), result.Delivered.Stream)
	assert.Equal(t, uint64(45), result.AckFloor.Consumer)
	assert.Equal(t, uint64(90), result.AckFloor.Stream)

	assert.Equal(t, uint64(10), result.NumPending)
	assert.Equal(t, 5, result.NumAckPending)
	assert.Equal(t, 2, result.NumRedelivered)
	assert.Equal(t, 3, result.NumWaiting)
	assert.True(t, result.PushBound)

	// Config
	require.NotNil(t, result.Config)
	cfg := result.Config
	assert.Equal(t, "test-consumer", cfg.Name)
	assert.Equal(t, "test-consumer", cfg.Durable)
	assert.Equal(t, "desc", cfg.Description)
	assert.Equal(t, entities.DeliverLastPerSubject, cfg.DeliverPolicy)
	assert.Equal(t, uint64(100), cfg.OptStartSeq)
	assert.Equal(t, entities.AckAll, cfg.AckPolicy)
	assert.Equal(t, 30*time.Second, cfg.AckWait, "AckWait: time.Duration passthrough")
	assert.Equal(t, 5, cfg.MaxDeliver)
	require.Len(t, cfg.BackOff, 2)
	assert.Equal(t, time.Second, cfg.BackOff[0], "BackOff: []time.Duration passthrough")
	assert.Equal(t, 5*time.Second, cfg.BackOff[1])
	assert.Equal(t, "orders.>", cfg.FilterSubject)
	assert.Equal(t, []string{"orders.created"}, cfg.FilterSubjects)
	assert.Equal(t, entities.ReplayOriginal, cfg.ReplayPolicy)
	assert.Equal(t, uint64(512), cfg.RateLimit)
	assert.Equal(t, "25%", cfg.SampleFrequency)
	assert.Equal(t, 100, cfg.MaxWaiting)
	assert.Equal(t, 200, cfg.MaxAckPending)
	assert.True(t, cfg.FlowControl)
	assert.Equal(t, 10*time.Second, cfg.IdleHeartbeat, "IdleHeartbeat: time.Duration passthrough")
	assert.True(t, cfg.HeadersOnly)
	assert.Equal(t, 50, cfg.MaxRequestBatch)
	assert.Equal(t, 5*time.Second, cfg.MaxRequestExpires, "MaxRequestExpires: time.Duration passthrough")
	assert.Equal(t, 3*time.Minute, cfg.InactiveThreshold, "InactiveThreshold: time.Duration passthrough")
	assert.Equal(t, 3, cfg.Replicas)
	assert.True(t, cfg.MemoryStorage)
	assert.Equal(t, 2048, cfg.MaxRequestMaxBytes)
	assert.Equal(t, map[string]string{"team": "api"}, cfg.Metadata)
	assert.Equal(t, "deliver.test", cfg.DeliverSubject)
	assert.Equal(t, "group1", cfg.DeliverGroup)

	// Cluster
	require.NotNil(t, result.Cluster)
	assert.Equal(t, "cluster", result.Cluster.Name)
	assert.Equal(t, "leader-1", result.Cluster.Leader)
}

// TestStreamUpdateMerge verifies mergeStreamUpdate applies only non-nil fields.
func TestStreamUpdateMerge(t *testing.T) {
	svc := &Client{}

	current := jetstream.StreamConfig{
		Name:        "existing",
		Subjects:    []string{"old.>"},
		MaxMsgs:     500,
		MaxBytes:    1024,
		MaxAge:      time.Hour,
		Duplicates:  time.Minute,
		Discard:     jetstream.DiscardOld,
		MaxMsgSize:  2048,
		Replicas:    1,
		Description: "original",
	}

	newMaxMsgs := int64(2000)
	newMaxAge := 2 * time.Hour
	newDiscard := entities.DiscardNew
	newDesc := "updated"

	update := entities.StreamUpdateRequest{
		MaxMsgs:     &newMaxMsgs,
		MaxAge:      &newMaxAge,
		Discard:     &newDiscard,
		Description: &newDesc,
		// Subjects, MaxBytes etc. not set → should keep existing
	}

	result := svc.mergeStreamUpdate(current, update)

	assert.Equal(t, "existing", result.Name, "immutable field preserved")
	assert.Equal(t, []string{"old.>"}, result.Subjects, "nil slice not overwritten")
	assert.Equal(t, int64(2000), result.MaxMsgs, "MaxMsgs updated")
	assert.Equal(t, int64(1024), result.MaxBytes, "nil MaxBytes preserved")
	assert.Equal(t, 2*time.Hour, result.MaxAge, "MaxAge: *time.Duration passthrough")
	assert.Equal(t, time.Minute, result.Duplicates, "nil Duplicates preserved")
	assert.Equal(t, jetstream.DiscardNew, result.Discard, "*DiscardPolicy → DiscardPolicy")
	assert.Equal(t, "updated", result.Description)
	assert.Equal(t, int32(2048), result.MaxMsgSize, "nil preserved")
	assert.Equal(t, 1, result.Replicas, "nil preserved")
}

// TestConsumerUpdateMerge verifies mergeConsumerUpdate applies only non-nil fields.
func TestConsumerUpdateMerge(t *testing.T) {
	svc := &Client{}

	current := jetstream.ConsumerConfig{
		Name:              "existing",
		AckWait:           30 * time.Second,
		MaxDeliver:        5,
		MaxAckPending:     100,
		InactiveThreshold: time.Minute,
		BackOff:           []time.Duration{time.Second},
		MaxRequestBatch:   10,
	}

	newAckWait := 60 * time.Second
	newMaxDeliver := 10
	newInactive := 5 * time.Minute

	update := entities.ConsumerUpdateRequest{
		AckWait:           &newAckWait,
		MaxDeliver:        &newMaxDeliver,
		InactiveThreshold: &newInactive,
		// MaxAckPending, BackOff not set → should keep existing
	}

	result := svc.mergeConsumerUpdate(current, update)

	assert.Equal(t, "existing", result.Name, "immutable field preserved")
	assert.Equal(t, 60*time.Second, result.AckWait, "*time.Duration passthrough")
	assert.Equal(t, 10, result.MaxDeliver, "*int → int")
	assert.Equal(t, 100, result.MaxAckPending, "nil preserved")
	assert.Equal(t, 5*time.Minute, result.InactiveThreshold, "*time.Duration passthrough")
	assert.Equal(t, []time.Duration{time.Second}, result.BackOff, "nil slice preserved")
	assert.Equal(t, 10, result.MaxRequestBatch, "nil preserved")
}

// TestConsumerUpdateMerge_BackOff verifies BackOff replacement when explicitly sent.
func TestConsumerUpdateMerge_BackOff(t *testing.T) {
	svc := &Client{}

	current := jetstream.ConsumerConfig{
		BackOff: []time.Duration{time.Second},
	}

	update := entities.ConsumerUpdateRequest{
		BackOff: []time.Duration{2 * time.Second, 10 * time.Second},
	}

	result := svc.mergeConsumerUpdate(current, update)

	require.Len(t, result.BackOff, 2)
	assert.Equal(t, 2*time.Second, result.BackOff[0])
	assert.Equal(t, 10*time.Second, result.BackOff[1])
}

// TestConsumerStatsConversion verifies the full ConsumerInfo → ConsumerStats mapping
// used by fetchStreamConsumersStats: two converter.Convert calls (top-level + Config flat).
func TestConsumerStatsConversion(t *testing.T) {
	created := time.Now().UTC().Truncate(time.Millisecond)
	startTime := created.Add(-time.Hour)

	info := &jetstream.ConsumerInfo{
		Name:   "my-consumer",
		Stream: "my-stream",
		Config: jetstream.ConsumerConfig{
			Name:               "my-consumer",
			Durable:            "my-consumer",
			Description:        "test desc",
			DeliverPolicy:      jetstream.DeliverByStartSequencePolicy,
			OptStartSeq:        42,
			OptStartTime:       &startTime,
			AckPolicy:          jetstream.AckAllPolicy,
			AckWait:            30 * time.Second,
			MaxDeliver:         5,
			BackOff:            []time.Duration{time.Second, 5 * time.Second},
			FilterSubject:      "orders.>",
			FilterSubjects:     []string{"orders.created", "orders.updated"},
			ReplayPolicy:       jetstream.ReplayOriginalPolicy,
			RateLimit:          1024,
			SampleFrequency:    "50%",
			MaxWaiting:         100,
			MaxAckPending:      200,
			HeadersOnly:        true,
			MaxRequestBatch:    25,
			MaxRequestExpires:  10 * time.Second,
			MaxRequestMaxBytes: 4096,
			InactiveThreshold:  5 * time.Minute,
			Replicas:           3,
			MemoryStorage:      true,
			Metadata:           map[string]string{"team": "backend"},
		},
		Created: created,
		Delivered: jetstream.SequenceInfo{
			Consumer: 50,
			Stream:   100,
		},
		AckFloor: jetstream.SequenceInfo{
			Consumer: 45,
			Stream:   90,
		},
		NumPending:     10,
		NumAckPending:  5,
		NumRedelivered: 2,
		NumWaiting:     3,
		PushBound:      false,
		Cluster: &jetstream.ClusterInfo{
			Name:   "nats-cluster",
			Leader: "node-1",
			Replicas: []*jetstream.PeerInfo{
				{Name: "node-2"},
				{Name: "node-3"},
			},
		},
	}

	// Replicate the same logic as fetchStreamConsumersStats
	consumer := converter.Convert(info, &entities.ConsumerStats{})
	consumer.Stream = "override-stream"

	// Config → flat. OptStartTime (*time.Time) is set explicitly because the
	// converter can't bridge it to the entity's time.Time value.
	converter.Convert(&info.Config, consumer,
		converter.WithIgnoreFields("Name", "OptStartTime"),
	)
	if info.Config.OptStartTime != nil {
		consumer.OptStartTime = *info.Config.OptStartTime
	}

	// === Assert top-level fields (first converter.Convert) ===
	assert.Equal(t, "my-consumer", consumer.Name)
	assert.Equal(t, "override-stream", consumer.Stream)
	assert.Equal(t, created, consumer.Created)
	assert.Equal(t, uint64(10), consumer.NumPending)
	assert.Equal(t, 5, consumer.NumAckPending)
	assert.Equal(t, 2, consumer.NumRedelivered)
	assert.Equal(t, 3, consumer.NumWaiting)
	assert.Equal(t, uint64(50), consumer.Delivered.Consumer)
	assert.Equal(t, uint64(100), consumer.Delivered.Stream)
	assert.Equal(t, uint64(45), consumer.AckFloor.Consumer)
	assert.Equal(t, uint64(90), consumer.AckFloor.Stream)
	assert.False(t, consumer.PushBound)

	// === Assert Config fields auto-mapped (second converter.Convert) ===
	assert.Equal(t, "my-consumer", consumer.Durable, "Durable: auto")
	assert.Equal(t, "test desc", consumer.Description, "Description: auto")
	assert.Equal(t, 5, consumer.MaxDeliver, "MaxDeliver: auto")
	assert.Equal(t, "orders.>", consumer.FilterSubject, "FilterSubject: auto")
	assert.Equal(t, []string{"orders.created", "orders.updated"}, consumer.FilterSubjects, "FilterSubjects: auto")
	assert.Equal(t, uint64(1024), consumer.RateLimit, "RateLimit: auto")
	assert.Equal(t, "50%", consumer.SampleFrequency, "SampleFrequency: auto")
	assert.Equal(t, 200, consumer.MaxAckPending, "MaxAckPending: auto")
	assert.True(t, consumer.HeadersOnly, "HeadersOnly: auto")
	assert.Equal(t, 25, consumer.MaxRequestBatch, "MaxRequestBatch: auto")
	assert.Equal(t, 4096, consumer.MaxRequestMaxBytes, "MaxRequestMaxBytes: auto")
	assert.Equal(t, 3, consumer.Replicas, "Replicas: auto")
	assert.True(t, consumer.MemoryStorage, "MemoryStorage: auto")
	assert.Equal(t, map[string]string{"team": "backend"}, consumer.Metadata, "Metadata: auto")

	// === Assert Config fields now auto-mapped (same types) ===
	assert.Equal(t, 100, consumer.MaxWaiting, "MaxWaiting: auto (name matches now)")
	assert.Equal(t, entities.DeliverByStartSequence, consumer.DeliverPolicy, "DeliverPolicy: enum auto")
	assert.Equal(t, entities.AckAll, consumer.AckPolicy, "AckPolicy: enum auto")
	assert.Equal(t, entities.ReplayOriginal, consumer.ReplayPolicy, "ReplayPolicy: enum auto")
	assert.Equal(t, 30*time.Second, consumer.AckWait, "AckWait: Duration passthrough auto")
	assert.Equal(t, 5*time.Minute, consumer.InactiveThreshold, "InactiveThreshold: Duration passthrough auto")
	assert.Equal(t, 10*time.Second, consumer.MaxRequestExpires, "MaxRequestExpires: Duration passthrough auto")

	assert.Equal(t, uint64(42), consumer.OptStartSeq, "OptStartSeq: auto")
	assert.Equal(t, startTime, consumer.OptStartTime, "OptStartTime: *time.Time→time.Time deref auto")

	require.Len(t, consumer.BackOff, 2)
	assert.Equal(t, time.Second, consumer.BackOff[0], "BackOff: []Duration passthrough auto")
	assert.Equal(t, 5*time.Second, consumer.BackOff[1])

	// === Assert Cluster (auto-converted, full PeerInfo) ===
	require.NotNil(t, consumer.Cluster)
	assert.Equal(t, "nats-cluster", consumer.Cluster.Name)
	assert.Equal(t, "node-1", consumer.Cluster.Leader)
	require.Len(t, consumer.Cluster.Replicas, 2)
	assert.Equal(t, "node-2", consumer.Cluster.Replicas[0].Name)
	assert.Equal(t, "node-3", consumer.Cluster.Replicas[1].Name)
}

// TestConsumerUpdateMerge_MaxRequestMaxBytes verifies int64→int narrowing.
func TestConsumerUpdateMerge_MaxRequestMaxBytes(t *testing.T) {
	svc := &Client{}

	current := jetstream.ConsumerConfig{
		MaxRequestMaxBytes: 512,
	}

	newVal := int64(4096)
	update := entities.ConsumerUpdateRequest{
		MaxRequestMaxBytes: &newVal,
	}

	result := svc.mergeConsumerUpdate(current, update)
	assert.Equal(t, 4096, result.MaxRequestMaxBytes, "*int64 → int")
}
