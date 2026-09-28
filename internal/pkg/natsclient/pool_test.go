// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsclient

import (
	"context"
	"sync"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
)

// fakeClient implements Client via embedding a nil interface (panics if an
// unexercised method is called) so the test only needs to fill in what
// Pool.Disconnect actually touches.
type fakeClient struct {
	Client
	connected bool
}

func (f *fakeClient) IsConnected() bool    { return f.connected }
func (f *fakeClient) IsReconnecting() bool { return false }
func (f *fakeClient) Status() string       { return "connected" }
func (f *fakeClient) Close()               { f.connected = false }

type fakeDialer struct {
	client Client
}

func (d *fakeDialer) Dial(context.Context, *entities.SavedConnection) (Client, error) {
	return d.client, nil
}

func (d *fakeDialer) TestConnection(
	context.Context,
	*entities.TestConnectionRequest,
) (*entities.TestConnectionResult, error) {
	return nil, nil //nolint:nilnil // unused by this test
}

// TestPool_OnDisconnect_NotifiesOnExplicitDisconnect (QA-028) verifies a
// registered listener fires when a connection is explicitly dropped from the
// pool — the hook a live session uses to end itself instead of silently
// going quiet once its connection is replaced.
func TestPool_OnDisconnect_NotifiesOnExplicitDisconnect(t *testing.T) {
	t.Parallel()

	fc := &fakeClient{connected: true}
	pool := NewPool(&fakeDialer{client: fc}, func(context.Context, string) (*entities.SavedConnection, error) {
		return &entities.SavedConnection{}, nil
	})

	var mu sync.Mutex
	var notified []string
	pool.OnDisconnect(func(connectionID string) {
		mu.Lock()
		defer mu.Unlock()
		notified = append(notified, connectionID)
	})

	if _, err := pool.Client(t.Context(), "conn-1"); err != nil {
		t.Fatalf("Client() error = %v", err)
	}

	pool.Disconnect("conn-1")

	mu.Lock()
	defer mu.Unlock()
	if len(notified) != 1 || notified[0] != "conn-1" {
		t.Fatalf("expected one notification for conn-1, got %v", notified)
	}
	if fc.connected {
		t.Fatalf("expected the client to be closed")
	}
}

// TestPool_OnDisconnect_NoListenersIsANoop ensures the default (no
// registered listeners) path behaves exactly as before.
func TestPool_OnDisconnect_NoListenersIsANoop(t *testing.T) {
	t.Parallel()

	fc := &fakeClient{connected: true}
	pool := NewPool(&fakeDialer{client: fc}, func(context.Context, string) (*entities.SavedConnection, error) {
		return &entities.SavedConnection{}, nil
	})

	if _, err := pool.Client(t.Context(), "conn-1"); err != nil {
		t.Fatalf("Client() error = %v", err)
	}
	pool.Disconnect("conn-1") // must not panic with zero listeners
}

// TestPool_OnDisconnect_UnknownConnectionDoesNotNotify verifies Disconnect on
// an id that was never dialed is a no-op, matching the pre-existing early
// return.
func TestPool_OnDisconnect_UnknownConnectionDoesNotNotify(t *testing.T) {
	t.Parallel()

	pool := NewPool(&fakeDialer{}, func(context.Context, string) (*entities.SavedConnection, error) {
		return &entities.SavedConnection{}, nil
	})

	called := false
	pool.OnDisconnect(func(string) { called = true })

	pool.Disconnect("never-dialed")

	if called {
		t.Fatalf("expected no notification for a connection that was never dialed")
	}
}
