// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func TestRedact(t *testing.T) {
	t.Parallel()
	secret := "s3cr3t-value"
	rtt := int64(3)
	conn := &entities.SavedConnection{
		BaseEntity: entities.BaseEntity{Id: "conn-1"},
		Name:       "local",
		URLs:       []string{"nats://alice:" + secret + "@localhost:4222", "nats://localhost:4223", "nats://bob:" + secret + "%zz@badhost:4222"},
		Auth:       &entities.AuthConfig{Method: entities.AuthMethodToken, Token: &secret, Password: &secret, NkeySeed: &secret},
		TLS:        &entities.TlsConfig{ClientKey: &secret},
		Meta: &entities.ConnectionMeta{
			LastTestedAt: time.Unix(1790000000, 0).UTC(), LastSuccess: true, LastRTTMs: &rtt,
			LastError: new("dial nats://alice:" + secret + "@localhost:4222 failed"),
		},
		Description: new("dev box"),
	}

	view := redact(conn)

	assert.Equal(t, "conn-1", view.Id)
	assert.Equal(t, "local", view.Name)
	assert.Equal(t, []string{"nats://localhost:4222", "nats://localhost:4223", "nats://***@badhost:4222"}, view.URLs)
	require.NotNil(t, view.Auth)
	assert.Equal(t, "token", view.Auth.Method)
	require.NotNil(t, view.Meta)
	assert.True(t, view.Meta.LastSuccess)
	assert.Equal(t, &rtt, view.Meta.LastRTTMs)

	raw, err := json.Marshal(view)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), secret)
}
