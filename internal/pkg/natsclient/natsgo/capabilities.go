// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"slices"
	"strings"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"golang.org/x/mod/semver"
)

// JetStream API levels by first release (nats-server JSApiLevel): 1 = 2.11.0,
// 2 = 2.12.0, 3 = 2.12.5 (snapshot window_size only), 4 = 2.14.0 (2.13 was
// skipped), 5 = 2.15.0 (cluster admin endpoints only). This file is the only
// place in the codebase that knows the NATS release history.

// feature is a version-gated NATS capability: the minimum JetStream API level
// that provides it and the release that introduced it.
type feature struct {
	name  string
	level int32
	since string
}

const (
	release211 = "2.11"
	release212 = "2.12"
	release214 = "2.14"

	apiLevel211 int32 = 1
	apiLevel212 int32 = 2
	apiLevel214 int32 = 4
)

var (
	featConsumerPause       = feature{name: "consumer pause", level: apiLevel211, since: release211}
	featMessageTTL          = feature{name: "per-message TTL", level: apiLevel211, since: release211}
	featPriorityGroups      = feature{name: "consumer priority groups", level: apiLevel211, since: release211}
	featAtomicPublish       = feature{name: "atomic batch publish", level: apiLevel212, since: release212}
	featMsgCounters         = feature{name: "message counters", level: apiLevel212, since: release212}
	featMsgSchedules        = feature{name: "message schedules", level: apiLevel212, since: release212}
	featPriorityPrioritized = feature{name: "prioritized priority policy", level: apiLevel212, since: release212}
	featAsyncPersist        = feature{name: "async persist mode", level: apiLevel212, since: release212}
	featConsumerReset       = feature{name: "consumer reset", level: apiLevel214, since: release214}
	featCronSchedules       = feature{name: "repeating message schedules", level: apiLevel214, since: release214}
	featBatchPublish        = feature{name: "fast batch publish", level: apiLevel214, since: release214}
)

// versionAPILevels maps release floors to the API level that release
// introduced, for servers that don't advertise a level (only 2.11.x; every
// release since 2.12.0 sends api_lvl in INFO). Ordered newest first; first
// match wins.
var versionAPILevels = []struct {
	minVersion string
	level      int32
}{
	{minVersion: "v2.11.0", level: apiLevel211},
}

// Publish headers whose server-side handling is version-gated.
const (
	headerMsgTTL           = "Nats-TTL"
	headerMsgIncr          = "Nats-Incr"
	headerSchedule         = "Nats-Schedule"
	headerScheduleTTL      = "Nats-Schedule-TTL"
	headerScheduleTimeZone = "Nats-Schedule-Time-Zone"
	headerScheduleSource   = "Nats-Schedule-Source"
	headerScheduleRollup   = "Nats-Schedule-Rollup"
	scheduleAtPrefix       = "@at "
)

// computeCapabilities derives feature support for a connected server.
// version is ConnectedServerVersion() (e.g. "2.14.2"), jsEnabled is the
// account-level JetStream availability, apiLevel is the level advertised on
// the handshake. Unknown/unparseable input degrades to "unsupported".
func computeCapabilities(version string, jsEnabled bool, apiLevel int) *entities.ServerCapabilities {
	level := effectiveAPILevel(version, apiLevel)
	if !jsEnabled {
		return &entities.ServerCapabilities{ApiLevel: level}
	}
	has := func(f feature) bool { return level >= f.level }
	return &entities.ServerCapabilities{
		ApiLevel:            level,
		ConsumerPause:       has(featConsumerPause),
		MessageTtl:          has(featMessageTTL),
		AtomicPublish:       has(featAtomicPublish),
		PriorityGroups:      has(featPriorityGroups),
		MsgCounters:         has(featMsgCounters),
		MsgSchedules:        has(featMsgSchedules),
		PriorityPrioritized: has(featPriorityPrioritized),
		AsyncPersist:        has(featAsyncPersist),
		ConsumerReset:       has(featConsumerReset),
		CronSchedules:       has(featCronSchedules),
		BatchPublish:        has(featBatchPublish),
	}
}

// effectiveAPILevel is the advertised level, or the semver fallback when the
// server advertises none.
func effectiveAPILevel(version string, advertised int) int32 {
	if advertised > 0 {
		return int32(advertised)
	}
	return apiLevelFromVersion(version)
}

// apiLevelFromVersion is the semver fallback for servers that don't advertise
// an API level.
func apiLevelFromVersion(version string) int32 {
	v := "v" + strings.TrimPrefix(strings.TrimSpace(version), "v")
	if !semver.IsValid(v) {
		return 0
	}
	for _, e := range versionAPILevels {
		if semver.Compare(v, e.minVersion) >= 0 {
			return e.level
		}
	}
	return 0
}

// checkFeatures rejects the first feature above level with
// [errs.FeatureUnsupportedError].
func checkFeatures(level int32, version string, features ...feature) error {
	for _, f := range features {
		if level < f.level {
			return &errs.FeatureUnsupportedError{
				Feature:       f.name,
				MinVersion:    f.since,
				ServerVersion: strings.TrimPrefix(strings.TrimSpace(version), "v"),
			}
		}
	}
	return nil
}

// requireFeatures guards calls that older servers would ignore or not answer
// with a clear error, using the connected server's API level.
func (c *Client) requireFeatures(features ...feature) error {
	version := c.conn.ConnectedServerVersion()
	if version == "" {
		return nil
	}
	_, advertised := c.conn.ConnectedServerJetStream()
	return checkFeatures(effectiveAPILevel(version, advertised), version, features...)
}

// headerFeatures lists the version-gated features a publish's headers use.
func headerFeatures(headers map[string]string) []feature {
	var out []feature
	add := func(f feature) {
		if !slices.Contains(out, f) {
			out = append(out, f)
		}
	}
	if _, ok := headers[headerMsgTTL]; ok {
		add(featMessageTTL)
	}
	if _, ok := headers[headerMsgIncr]; ok {
		add(featMsgCounters)
	}
	if pattern, ok := headers[headerSchedule]; ok {
		add(featMsgSchedules)
		if !strings.HasPrefix(pattern, scheduleAtPrefix) {
			add(featCronSchedules)
		}
	}
	if _, ok := headers[headerScheduleTTL]; ok {
		add(featMessageTTL)
	}
	for _, key := range []string{headerScheduleTimeZone, headerScheduleSource, headerScheduleRollup} {
		if _, ok := headers[key]; ok {
			add(featCronSchedules)
		}
	}
	return out
}

// streamConfigFeatures lists the version-gated features a stream config uses,
// mirroring the server's own required-API-level rules for streams.
func streamConfigFeatures(cfg jetstream.StreamConfig) []feature {
	var out []feature
	if cfg.AllowMsgTTL || cfg.SubjectDeleteMarkerTTL > 0 {
		out = append(out, featMessageTTL)
	}
	if cfg.AllowMsgCounter {
		out = append(out, featMsgCounters)
	}
	if cfg.AllowAtomicPublish {
		out = append(out, featAtomicPublish)
	}
	if cfg.AllowMsgSchedules {
		out = append(out, featMsgSchedules)
	}
	if cfg.PersistMode == jetstream.AsyncPersistMode {
		out = append(out, featAsyncPersist)
	}
	if cfg.AllowBatchPublish {
		out = append(out, featBatchPublish)
	}
	return out
}

// consumerConfigFeatures lists the version-gated features a consumer config
// uses, mirroring the server's own required-API-level rules for consumers.
func consumerConfigFeatures(cfg jetstream.ConsumerConfig) []feature {
	var out []feature
	if cfg.PauseUntil != nil && !cfg.PauseUntil.IsZero() {
		out = append(out, featConsumerPause)
	}
	if cfg.PriorityPolicy != jetstream.PriorityPolicyNone {
		out = append(out, featPriorityGroups)
	}
	if cfg.PriorityPolicy == jetstream.PriorityPolicyPrioritized {
		out = append(out, featPriorityPrioritized)
	}
	return out
}
