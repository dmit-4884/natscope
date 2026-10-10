// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package nats

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsclient"

	connectionsStorage "github.com/dmit-4884/natscope/internal/storages/connections"
)

type fakeClient struct {
	natsclient.Client
	connected atomic.Bool
}

func (f *fakeClient) IsConnected() bool { return f.connected.Load() }

type fakeDialer struct {
	natsclient.Dialer
	client *fakeClient
}

func (d *fakeDialer) Dial(context.Context, *entities.SavedConnection) (natsclient.Client, error) {
	return d.client, nil
}

type fakeStore struct{ connectionsStorage.Storage }

func (fakeStore) Get(context.Context, string, ...bool) (*entities.SavedConnection, error) {
	return &entities.SavedConnection{}, nil
}

func TestLinkDown_FollowsThePooledConnection(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	client.connected.Store(true)
	svc := New(&fakeDialer{client: client}, fakeStore{})

	assert.False(t, svc.LinkDown("conn"), "a connection that is not pooled is not reported")

	_, err := svc.client(t.Context(), "conn")
	require.NoError(t, err)
	assert.False(t, svc.LinkDown("conn"))

	client.connected.Store(false)
	assert.True(t, svc.LinkDown("conn"))
}
