// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func TestRequest_ARefusedReplyInboxIsReportedEveryTime(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) {
		o.JetStream, o.StoreDir = true, t.TempDir()
		o.Users = []*server.User{{
			Username: "app", Password: "pw",
			Permissions: &server.Permissions{
				Publish:   &server.SubjectPermission{Allow: []string{">"}},
				Subscribe: &server.SubjectPermission{Allow: []string{">"}, Deny: []string{"_INBOX.>"}},
			},
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

	for i := range 3 {
		ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
		start := time.Now()
		_, reqErr := c.Request(ctx, "svc.echo", []byte("hi"), nil)
		cancel()

		permErr, isPerm := errors.AsType[*errs.NATSPermissionError](reqErr)
		require.True(t, isPerm, "request %d: %v", i, reqErr)
		assert.Equal(t, "_INBOX.>", permErr.Subject)
		assert.Less(t, time.Since(start), 2*time.Second, "request %d waits out its timeout", i)
	}

	_, err = c.ListStreams(t.Context())
	permErr, isPerm := errors.AsType[*errs.NATSPermissionError](err)
	require.True(t, isPerm, "%v", err)
	assert.Equal(t, errs.PermissionOperationSubscribe, permErr.Operation)
	assert.Equal(t, "_INBOX.>", permErr.Subject, "the missing permission, not the connection's random reply subject")
}
