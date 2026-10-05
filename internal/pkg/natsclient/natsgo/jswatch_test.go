// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJetStreamWatch_SubjectsFollowTheAPIPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		domain, prefix string
		want           []string
	}{
		{"default", "", "", []string{"$JS.API.STREAM.INFO.ORDERS", "$JS.API.CONSUMER.CREATE.ORDERS."}},
		{"domain", "hub", "", []string{"$JS.hub.API.STREAM.INFO.ORDERS", "$JS.hub.API.CONSUMER.CREATE.ORDERS."}},
		{"imported prefix with a trailing dot", "", "JS.A.API.", []string{"JS.A.API.STREAM.INFO.ORDERS", "JS.A.API.CONSUMER.CREATE.ORDERS."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &jetStreamWatch{api: apiPrefix(tt.domain, tt.prefix)}
			assert.Equal(t, tt.want, w.subjectsFor("ORDERS", subjStreamInfo, subjConsumerCreate))
			assert.Equal(t, w.api+".STREAM.LIST", w.subject(subjStreamList))
			assert.Equal(t, w.api+".CONSUMER.INFO.ORDERS.c", (&Client{api: w.api}).apiSubject("CONSUMER.INFO.ORDERS.c"))
		})
	}
}
