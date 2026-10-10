// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// TestComputeCapabilities verifies API-level gating with semver fallback for
// servers that don't advertise a level (< 2.12 advertise 0).
func TestComputeCapabilities(t *testing.T) {
	level1 := entities.ServerCapabilities{
		ApiLevel: 1, ConsumerPause: true, MessageTtl: true, PriorityGroups: true,
	}
	level2 := entities.ServerCapabilities{
		ApiLevel: 2, ConsumerPause: true, MessageTtl: true, PriorityGroups: true,
		AtomicPublish: true, MsgCounters: true, MsgSchedules: true, PriorityPrioritized: true, AsyncPersist: true,
	}
	level3 := level2
	level3.ApiLevel = 3
	level4 := entities.ServerCapabilities{
		ApiLevel: 4, ConsumerPause: true, MessageTtl: true, PriorityGroups: true,
		AtomicPublish: true, MsgCounters: true, MsgSchedules: true, PriorityPrioritized: true, AsyncPersist: true,
		ConsumerReset: true, CronSchedules: true, BatchPublish: true,
	}
	level5 := level4
	level5.ApiLevel = 5

	tests := []struct {
		name      string
		version   string
		jsEnabled bool
		apiLevel  int
		want      entities.ServerCapabilities
	}{
		{name: "2.15 advertises level 5", version: "2.15.0", jsEnabled: true, apiLevel: 5, want: level5},
		{name: "2.14 advertises level 4", version: "2.14.2", jsEnabled: true, apiLevel: 4, want: level4},
		{name: "2.12.5+ advertises level 3", version: "2.12.15", jsEnabled: true, apiLevel: 3, want: level3},
		{name: "2.12 advertises level 2", version: "2.12.0", jsEnabled: true, apiLevel: 2, want: level2},
		{name: "2.11 without advertised level falls back to semver", version: "2.11.3", jsEnabled: true, apiLevel: 0, want: level1},
		{name: "v-prefixed version string", version: "v2.11.0", jsEnabled: true, apiLevel: 0, want: level1},
		{
			name: "2.10 without advertised level has no gated features", version: "2.10.24", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 0},
		},
		{
			name: "jetstream disabled clears feature flags but keeps level", version: "2.14.2", jsEnabled: false, apiLevel: 4,
			want: entities.ServerCapabilities{ApiLevel: 4},
		},
		{
			name: "empty version is conservative", version: "", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 0},
		},
		{
			name: "garbage version is conservative", version: "not-a-version", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeCapabilities(tt.version, tt.jsEnabled, tt.apiLevel)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}

// TestCheckFeatures verifies the backend guard rejects features above the
// server's API level with ErrFeatureUnsupported and a version hint.
func TestCheckFeatures(t *testing.T) {
	tests := []struct {
		name     string
		level    int32
		version  string
		features []feature
		wantErr  string
	}{
		{name: "no features", level: 0, version: "2.10.0"},
		{name: "supported at exact level", level: 4, version: "2.14.0", features: []feature{featConsumerReset}},
		{name: "supported above level", level: 5, version: "2.15.0", features: []feature{featMsgCounters, featConsumerReset}},
		{
			name: "unsupported reports first failing feature", level: 2, version: "2.12.3",
			features: []feature{featMsgCounters, featConsumerReset},
			wantErr:  "consumer reset requires NATS 2.14+ (connected server v2.12.3)",
		},
		{
			name: "unknown version", level: 0, version: "",
			features: []feature{featMessageTTL},
			wantErr:  "per-message TTL requires NATS 2.11+ (connected server version unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkFeatures(tt.level, tt.version, tt.features...)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, errs.ErrFeatureUnsupported))
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestHeaderFeatures verifies which version-gated features a publish's headers
// use; keys match exactly because the server looks headers up case-sensitively.
func TestHeaderFeatures(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    []feature
	}{
		{name: "no headers", headers: nil, want: nil},
		{name: "unrelated headers", headers: map[string]string{"Nats-Msg-Id": "a", "X-Trace": "b"}, want: nil},
		{name: "message ttl", headers: map[string]string{"Nats-TTL": "5s"}, want: []feature{featMessageTTL}},
		{name: "counter increment", headers: map[string]string{"Nats-Incr": "+1"}, want: []feature{featMsgCounters}},
		{
			name:    "single delayed schedule",
			headers: map[string]string{"Nats-Schedule": "@at 2030-01-01T00:00:00Z", "Nats-Schedule-Target": "orders"},
			want:    []feature{featMsgSchedules},
		},
		{
			name:    "schedule ttl needs message ttl",
			headers: map[string]string{"Nats-Schedule": "@at 2030-01-01T00:00:00Z", "Nats-Schedule-TTL": "5m"},
			want:    []feature{featMsgSchedules, featMessageTTL},
		},
		{
			name:    "interval schedule",
			headers: map[string]string{"Nats-Schedule": "@every 5m"},
			want:    []feature{featMsgSchedules, featCronSchedules},
		},
		{
			name:    "cron schedule with time zone",
			headers: map[string]string{"Nats-Schedule": "0 0 * * * *", "Nats-Schedule-Time-Zone": "Europe/Amsterdam"},
			want:    []feature{featMsgSchedules, featCronSchedules},
		},
		{
			name:    "subject sampling",
			headers: map[string]string{"Nats-Schedule": "@at 2030-01-01T00:00:00Z", "Nats-Schedule-Source": "sensors.temp"},
			want:    []feature{featMsgSchedules, featCronSchedules},
		},
		{
			name:    "scheduled rollup",
			headers: map[string]string{"Nats-Schedule": "@hourly", "Nats-Schedule-Rollup": "sub"},
			want:    []feature{featMsgSchedules, featCronSchedules},
		},
		{name: "lowercase keys are not recognized by the server", headers: map[string]string{"nats-ttl": "5s", "nats-incr": "1"}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, headerFeatures(tt.headers))
		})
	}
}

// TestStreamConfigFeatures mirrors the server's setStaticStreamMetadata: every
// flag an older server would silently drop maps to its feature.
func TestStreamConfigFeatures(t *testing.T) {
	tests := []struct {
		name string
		cfg  jetstream.StreamConfig
		want []feature
	}{
		{name: "plain stream", cfg: jetstream.StreamConfig{Name: "S"}, want: nil},
		{name: "message ttl", cfg: jetstream.StreamConfig{AllowMsgTTL: true}, want: []feature{featMessageTTL}},
		{name: "delete markers", cfg: jetstream.StreamConfig{SubjectDeleteMarkerTTL: time.Minute}, want: []feature{featMessageTTL}},
		{name: "counter", cfg: jetstream.StreamConfig{AllowMsgCounter: true}, want: []feature{featMsgCounters}},
		{name: "atomic", cfg: jetstream.StreamConfig{AllowAtomicPublish: true}, want: []feature{featAtomicPublish}},
		{name: "schedules", cfg: jetstream.StreamConfig{AllowMsgSchedules: true}, want: []feature{featMsgSchedules}},
		{name: "async persist", cfg: jetstream.StreamConfig{PersistMode: jetstream.AsyncPersistMode}, want: []feature{featAsyncPersist}},
		{name: "fast batch", cfg: jetstream.StreamConfig{AllowBatchPublish: true}, want: []feature{featBatchPublish}},
		{
			name: "several",
			cfg:  jetstream.StreamConfig{AllowMsgTTL: true, SubjectDeleteMarkerTTL: time.Minute, AllowMsgSchedules: true},
			want: []feature{featMessageTTL, featMsgSchedules},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, streamConfigFeatures(tt.cfg))
		})
	}
}

// TestConsumerConfigFeatures mirrors the server's setStaticConsumerMetadata.
func TestConsumerConfigFeatures(t *testing.T) {
	pauseUntil := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		cfg  jetstream.ConsumerConfig
		want []feature
	}{
		{name: "plain consumer", cfg: jetstream.ConsumerConfig{Durable: "c"}, want: nil},
		{name: "pinned client", cfg: jetstream.ConsumerConfig{PriorityPolicy: jetstream.PriorityPolicyPinned}, want: []feature{featPriorityGroups}},
		{name: "overflow", cfg: jetstream.ConsumerConfig{PriorityPolicy: jetstream.PriorityPolicyOverflow}, want: []feature{featPriorityGroups}},
		{
			name: "prioritized",
			cfg:  jetstream.ConsumerConfig{PriorityPolicy: jetstream.PriorityPolicyPrioritized},
			want: []feature{featPriorityGroups, featPriorityPrioritized},
		},
		{name: "paused at creation", cfg: jetstream.ConsumerConfig{PauseUntil: &pauseUntil}, want: []feature{featConsumerPause}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, consumerConfigFeatures(tt.cfg))
		})
	}
}

// TestRequireFeatures_UnknownServerFailsOpen verifies the guard stays out of
// the way while disconnected, so the call fails with its real connection error.
func TestRequireFeatures_UnknownServerFailsOpen(t *testing.T) {
	c := &Client{}
	assert.NoError(t, c.requireFeatures(t.Context(), featConsumerReset, featMsgCounters))
}

// remoteJetStream answers AccountInfo for another domain's JetStream with a fixed API level.
type remoteJetStream struct {
	jetstream.JetStream
	level int
	err   error
	calls int
}

func (r *remoteJetStream) AccountInfo(context.Context) (*jetstream.AccountInfo, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return &jetstream.AccountInfo{API: jetstream.APIStats{Level: r.level}}, nil
}

// TestRequireFeatures_AnotherDomainUsesItsOwnLevel checks that a domain or API prefix gates on the JetStream that
// answers, not on the server the client is connected to.
func TestRequireFeatures_AnotherDomainUsesItsOwnLevel(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)

	t.Run("an older hub rejects the feature", func(t *testing.T) {
		t.Parallel()
		c := dialClient(t, url)
		remote := &remoteJetStream{level: int(apiLevel211)}
		c.api, c.jetStream = "$JS.hub.API", remote

		err := c.requireFeatures(t.Context(), featConsumerReset)
		require.ErrorIs(t, err, errs.ErrFeatureUnsupported)
		assert.Contains(t, err.Error(), "$JS.hub.API")
		require.ErrorIs(t, c.requireFeatures(t.Context(), featMsgCounters), errs.ErrFeatureUnsupported)
		assert.NoError(t, c.requireFeatures(t.Context(), featConsumerPause))
		assert.Equal(t, 1, remote.calls, "the level is read once")
	})

	t.Run("an unknown level lets the server decide", func(t *testing.T) {
		t.Parallel()
		c := dialClient(t, url)
		c.api, c.jetStream = "$JS.hub.API", &remoteJetStream{err: errors.New("no responders")}

		assert.NoError(t, c.requireFeatures(t.Context(), featConsumerReset))
	})

	t.Run("a refused level is not asked again", func(t *testing.T) {
		t.Parallel()
		c := dialClient(t, url)
		remote := &remoteJetStream{err: &errs.NATSPermissionError{Operation: errs.PermissionOperationPublish, Subject: "$JS.hub.API.INFO"}}
		c.api, c.jetStream = "$JS.hub.API", remote

		assert.NoError(t, c.requireFeatures(t.Context(), featMessageTTL))
		assert.NoError(t, c.requireFeatures(t.Context(), featMessageTTL))
		assert.Equal(t, 1, remote.calls)
	})

	t.Run("server info reports the hub's capabilities", func(t *testing.T) {
		t.Parallel()
		c := dialClient(t, url)
		c.api, c.jetStream = "$JS.hub.API", &remoteJetStream{level: int(apiLevel211)}

		info, err := c.GetServerInfo(t.Context())
		require.NoError(t, err)
		assert.Equal(t, apiLevel211, info.Capabilities.ApiLevel)
		assert.False(t, info.Capabilities.ConsumerReset)
		assert.True(t, info.Capabilities.ConsumerPause)
	})
}

// TestRequireFeatures_RefusedLevelDoesNotHoldTheCall checks that a domain whose API INFO the user may not ask lets a
// gated call through at once and is not asked again.
func TestRequireFeatures_RefusedLevelDoesNotHoldTheCall(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) {
		o.JetStream, o.JetStreamDomain, o.StoreDir = true, "hub", t.TempDir()
		o.Users = []*server.User{{
			Username: "app", Password: "pw",
			Permissions: &server.Permissions{Publish: &server.SubjectPermission{Allow: []string{">"}, Deny: []string{"$JS.API.INFO"}}},
		}}
	})
	conn, err := NewDialer().Dial(t.Context(), &entities.SavedConnection{
		URLs:       []string{url},
		Auth:       &entities.AuthConfig{Method: entities.AuthMethodUserPass, Username: new("app"), Password: new("pw")},
		Connection: &entities.ConnectionConfig{JetstreamDomain: new("hub")},
	})
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	c, ok := conn.(*Client)
	require.True(t, ok)

	for i := range 2 {
		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		start := time.Now()
		sent := c.conn.Stats().OutMsgs
		gateErr := c.requireFeatures(ctx, featMessageTTL)
		cancel()
		require.NoError(t, gateErr)
		assert.Less(t, time.Since(start), 2*time.Second, "call %d waits out its timeout", i)
		if i > 0 {
			assert.Equal(t, sent, c.conn.Stats().OutMsgs, "the refused level is asked again")
		}
	}
}

// TestRequireFeatures_TheDefaultJetStreamGatesOnTheConnectedServer checks that a connection without a domain or
// prefix takes the API level from the server it connected to, so a denied $JS.API.INFO does not turn the check off.
func TestRequireFeatures_TheDefaultJetStreamGatesOnTheConnectedServer(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) {
		o.JetStream, o.StoreDir = true, t.TempDir()
		o.Users = []*server.User{{
			Username: "app", Password: "pw",
			Permissions: &server.Permissions{Publish: &server.SubjectPermission{Allow: []string{">"}, Deny: []string{"$JS.API.INFO"}}},
		}}
	})
	conn, err := NewDialer().Dial(t.Context(), &entities.SavedConnection{
		URLs: []string{url},
		Auth: &entities.AuthConfig{Method: entities.AuthMethodUserPass, Username: new("app"), Password: new("pw")},
	})
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	c, ok := conn.(*Client)
	require.True(t, ok)

	future := feature{name: "a feature of a later release", level: apiLevel214 + 1, since: "2.15"}
	sent := c.conn.Stats().OutMsgs
	require.ErrorIs(t, c.requireFeatures(t.Context(), future), errs.ErrFeatureUnsupported)
	assert.Equal(t, sent, c.conn.Stats().OutMsgs, "the check asked the server")
}
