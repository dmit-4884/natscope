// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsclient

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
)

// fakeClient embeds a nil Client, so unexercised methods panic.
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

// TestPool_OnDisconnect_NotifiesOnExplicitDisconnect checks that a listener fires on Disconnect.
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

// TestPool_OnDisconnect_UnknownConnectionDoesNotNotify checks that Disconnect on an unknown id is a no-op.
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

// blockingDeadClient is a dead client whose first Status call waits, so a caller can be held right before it
// drops the client.
type blockingDeadClient struct {
	fakeClient
	held    atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (c *blockingDeadClient) Status() string {
	if c.held.CompareAndSwap(false, true) {
		close(c.entered)
		<-c.release
	}
	return "closed"
}

type countingDialer struct {
	mu    sync.Mutex
	dials []*fakeClient
}

func (d *countingDialer) Dial(context.Context, *entities.SavedConnection) (Client, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	c := &fakeClient{connected: true}
	d.dials = append(d.dials, c)
	return c, nil
}

func (d *countingDialer) TestConnection(context.Context, *entities.TestConnectionRequest) (*entities.TestConnectionResult, error) {
	return nil, nil //nolint:nilnil // unused by this test
}

// TestPool_ALateCallerDoesNotDropTheFreshClient checks that a caller who saw the old dead client removes only that one.
func TestPool_ALateCallerDoesNotDropTheFreshClient(t *testing.T) {
	dialer := &countingDialer{}
	pool := NewPool(dialer, func(context.Context, string) (*entities.SavedConnection, error) {
		return &entities.SavedConnection{}, nil
	})
	dead := &blockingDeadClient{entered: make(chan struct{}), release: make(chan struct{})}
	pool.clients["c1"] = dead

	late := make(chan Client, 1)
	go func() {
		c, err := pool.Client(t.Context(), "c1")
		if err != nil {
			t.Errorf("late caller: %v", err)
		}
		late <- c
	}()
	<-dead.entered

	fresh, err := pool.Client(t.Context(), "c1")
	if err != nil {
		t.Fatalf("Client() error = %v", err)
	}
	close(dead.release)
	got := <-late

	if got != fresh {
		t.Errorf("late caller got %p, want the fresh client %p", got, fresh)
	}
	if !fresh.IsConnected() {
		t.Error("the fresh client was closed by a caller that saw the old dead one")
	}
	if n := len(dialer.dials); n != 1 {
		t.Errorf("dials = %d, want 1", n)
	}
}

// blockingCloseClient waits in Close until released, as a nats.go connection does while a reconnect attempt dials.
type blockingCloseClient struct {
	fakeClient
	closing chan struct{}
	release chan struct{}
}

func (c *blockingCloseClient) Close() {
	close(c.closing)
	<-c.release
}

// TestPool_ASlowCloseDoesNotHoldOtherConnections checks that closing one connection leaves the others usable.
func TestPool_ASlowCloseDoesNotHoldOtherConnections(t *testing.T) {
	t.Parallel()

	pool := NewPool(&fakeDialer{}, func(context.Context, string) (*entities.SavedConnection, error) {
		return &entities.SavedConnection{}, nil
	})
	slow := &blockingCloseClient{fakeClient: fakeClient{connected: true}, closing: make(chan struct{}), release: make(chan struct{})}
	other := &fakeClient{connected: true}
	pool.clients["a"] = slow
	pool.clients["b"] = other

	go pool.Disconnect("a")
	<-slow.closing
	defer close(slow.release)

	got := make(chan Client, 1)
	go func() {
		c, _ := pool.Client(t.Context(), "b")
		got <- c
	}()
	select {
	case c := <-got:
		if c != other {
			t.Errorf("Client(b) = %p, want %p", c, other)
		}
	case <-time.After(time.Second):
		t.Fatal("closing connection a held the pool for connection b")
	}
}
