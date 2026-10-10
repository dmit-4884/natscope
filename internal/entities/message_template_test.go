// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageTemplate_ApplyUpdate(t *testing.T) {
	t.Parallel()

	seed := func() *MessageTemplate {
		return MessageTemplateNew(func(m *MessageTemplate) {
			m.Name = "old"
			m.Subject = "orders.*"
			m.Data = "{}"
			m.Headers = map[string]string{"a": "1"}
			m.Wildcards = []string{"x"}
		})
	}

	t.Run("NilKeepsExisting", func(t *testing.T) {
		t.Parallel()
		m := seed()
		m.ApplyUpdate(&MessageTemplateUpdate{Name: new("new")})

		assert.Equal(t, "new", m.Name)
		assert.Equal(t, "orders.*", m.Subject)
		assert.Equal(t, "{}", m.Data)
		assert.Equal(t, map[string]string{"a": "1"}, m.Headers)
		assert.Equal(t, []string{"x"}, m.Wildcards)
	})

	t.Run("EmptyClears", func(t *testing.T) {
		t.Parallel()
		m := seed()
		m.ApplyUpdate(&MessageTemplateUpdate{Headers: map[string]string{}, Wildcards: []string{}})

		assert.NotNil(t, m.Headers)
		assert.Empty(t, m.Headers)
		assert.NotNil(t, m.Wildcards)
		assert.Empty(t, m.Wildcards)
	})

	t.Run("NonEmptyReplaces", func(t *testing.T) {
		t.Parallel()
		m := seed()
		m.ApplyUpdate(&MessageTemplateUpdate{Headers: map[string]string{"b": "2"}, Wildcards: []string{"y", "z"}, Data: new("{\"k\":1}")})

		assert.Equal(t, map[string]string{"b": "2"}, m.Headers)
		assert.Equal(t, []string{"y", "z"}, m.Wildcards)
		assert.JSONEq(t, "{\"k\":1}", m.Data)
	})

	t.Run("NilReceiverAndRequest", func(t *testing.T) {
		t.Parallel()
		var nilTemplate *MessageTemplate
		nilTemplate.ApplyUpdate(&MessageTemplateUpdate{Name: new("x")})
		seed().ApplyUpdate(nil)
	})
}
