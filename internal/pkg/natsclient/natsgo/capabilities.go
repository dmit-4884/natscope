// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"

	"golang.org/x/mod/semver"
)

// Feature gates: the minimum JetStream API level that provides each feature.
// This file is the only place in the codebase that knows the NATS release
// history.
const (
	apiLevelConsumerPause = 1 // NATS 2.11
	apiLevelMessageTtl    = 1 // NATS 2.11
	apiLevelAtomicPublish = 2 // NATS 2.12
)

// versionAPILevels maps release floors to the API level that release
// introduced, for servers that don't advertise a level (< 2.12 advertise 0).
// Ordered newest first; first match wins.
var versionAPILevels = []struct {
	minVersion string
	level      int32
}{
	{minVersion: "v2.11.0", level: 1},
}

// computeCapabilities derives feature support for a connected server.
// version is ConnectedServerVersion() (e.g. "2.14.2"), jsEnabled is the
// account-level JetStream availability, apiLevel is the level advertised on
// the handshake. Unknown/unparseable input degrades to "unsupported".
func computeCapabilities(version string, jsEnabled bool, apiLevel int) *entities.ServerCapabilities {
	level := int32(apiLevel)
	if level == 0 {
		level = apiLevelFromVersion(version)
	}
	if !jsEnabled {
		return &entities.ServerCapabilities{ApiLevel: level}
	}
	return &entities.ServerCapabilities{
		ApiLevel:      level,
		ConsumerPause: level >= apiLevelConsumerPause,
		MessageTtl:    level >= apiLevelMessageTtl,
		AtomicPublish: level >= apiLevelAtomicPublish,
	}
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
