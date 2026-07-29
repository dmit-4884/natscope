// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavedConnectionNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew()
		require.NotNil(t, c)
		assert.NotEmpty(t, c.Id, "Id should be set")
		assert.Positive(t, c.CreatedAt, "CreatedAt should be positive")
		assert.Empty(t, c.Name, "Name should be empty")
		assert.Nil(t, c.URLs, "URLs should be nil")
		assert.Nil(t, c.Auth, "Auth should be nil")
		assert.Nil(t, c.TLS, "TLS should be nil")
		assert.Nil(t, c.Connection, "Connection should be nil")
		assert.Nil(t, c.Reconnect, "Reconnect should be nil")
		assert.Nil(t, c.Ping, "Ping should be nil")
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew(func(c *SavedConnection) {
			c.Name = "test"
			c.URLs = []string{"nats://localhost:4222"}
		})
		assert.Equal(t, "test", c.Name)
		assert.Equal(t, []string{"nats://localhost:4222"}, c.URLs)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew(nil)
		require.NotNil(t, c)
		assert.NotEmpty(t, c.Id)
	})
}

func TestSavedConnection_ApplyUpdate(t *testing.T) {
	t.Parallel()

	t.Run("UpdatesFields", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew(func(c *SavedConnection) {
			c.Name = "old-name"
			c.URLs = []string{"nats://old:4222"}
		})
		// Seed a known-old UpdatedAt instead of sleeping (deterministic, race-free).
		c.UpdatedAt = time.UnixMilli(1).UTC()
		c.ApplyUpdate(&SavedConnectionUpdate{
			Name: ptrString("new-name"),
			URLs: []string{"nats://new:4222"},
		})

		assert.Equal(t, "new-name", c.Name)
		assert.Equal(t, []string{"nats://new:4222"}, c.URLs)
		assert.True(t, c.UpdatedAt.After(time.UnixMilli(1).UTC()), "UpdatedAt should advance")
	})

	t.Run("PartialUpdate_NilFieldsIgnored", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew(func(c *SavedConnection) {
			c.Name = "keep-this"
			c.URLs = []string{"nats://keep:4222"}
		})

		c.ApplyUpdate(&SavedConnectionUpdate{
			Name: ptrString("changed"),
			// URLs is nil — should not change
		})

		assert.Equal(t, "changed", c.Name)
		assert.Equal(t, []string{"nats://keep:4222"}, c.URLs, "URLs should not change when nil in update")
	})

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()
		var c *SavedConnection
		c.ApplyUpdate(&SavedConnectionUpdate{Name: ptrString("x")})
	})

	t.Run("NilRequest", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew()
		oldUpdatedAt := c.UpdatedAt
		c.ApplyUpdate(nil)
		assert.Equal(t, oldUpdatedAt, c.UpdatedAt, "UpdatedAt should not change when request is nil")
	})

	t.Run("UpdateAuthConfig", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew()

		c.ApplyUpdate(&SavedConnectionUpdate{
			Auth: &AuthConfig{
				Method:   AuthMethodUserPass,
				Username: ptrString("admin"),
				Password: ptrString("secret"),
			},
		})

		require.NotNil(t, c.Auth)
		assert.Equal(t, AuthMethodUserPass, c.Auth.Method)
		require.NotNil(t, c.Auth.Username)
		assert.Equal(t, "admin", *c.Auth.Username)
		require.NotNil(t, c.Auth.Password)
		assert.Equal(t, "secret", *c.Auth.Password)
	})

	t.Run("UpdateTLSConfig", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew()

		c.ApplyUpdate(&SavedConnectionUpdate{
			TLS: &TlsConfig{
				SkipVerify: true,
				TlsFirst:   true,
			},
		})

		require.NotNil(t, c.TLS)
		assert.True(t, c.TLS.SkipVerify)
		assert.True(t, c.TLS.TlsFirst)
	})
}

func TestSavedConnection_PrimaryURL(t *testing.T) {
	t.Parallel()

	t.Run("WithURLs", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew(func(c *SavedConnection) {
			c.URLs = []string{"nats://a:4222", "nats://b:4222"}
		})
		assert.Equal(t, "nats://a:4222", c.PrimaryURL())
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		c := SavedConnectionNew()
		assert.Empty(t, c.PrimaryURL())
	})
}

func TestSavedConnection_URLsString(t *testing.T) {
	t.Parallel()

	c := SavedConnectionNew(func(c *SavedConnection) {
		c.URLs = []string{"nats://a:4222", "nats://b:4222", "nats://c:4222"}
	})
	assert.Equal(t, "nats://a:4222,nats://b:4222,nats://c:4222", c.URLsString())
}

func TestSavedConnectionsList(t *testing.T) {
	t.Parallel()

	l := &SavedConnectionsList{
		ListBase: ListBase{Cursor: "abc", Limit: ptrInt64(10)},
	}
	assert.Equal(t, "abc", l.Cursor)
	assert.Equal(t, int64(10), l.GetLimit())
}
