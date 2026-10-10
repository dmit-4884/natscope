// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

// switchProxy forwards a TCP port until it is cut; then it drops every forwarded link and holds new ones silently,
// as a firewall that accepts and never answers does.
type switchProxy struct {
	url string

	mu      sync.Mutex
	cut     bool
	links   []net.Conn
	waiting int
}

func newSwitchProxy(t *testing.T, upstream string) *switchProxy {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &switchProxy{url: "nats://" + lis.Addr().String()}
	t.Cleanup(func() {
		_ = lis.Close()
		p.mu.Lock()
		defer p.mu.Unlock()
		for _, c := range p.links {
			_ = c.Close()
		}
	})
	go func() {
		for {
			client, err := lis.Accept()
			if err != nil {
				return
			}
			p.mu.Lock()
			p.links = append(p.links, client)
			if p.cut {
				p.waiting++
				p.mu.Unlock()
				continue
			}
			p.mu.Unlock()
			go func() {
				srv, err := net.Dial("tcp", upstream)
				if err != nil {
					_ = client.Close()
					return
				}
				go func() { _, _ = io.Copy(srv, client); _ = srv.Close() }()
				_, _ = io.Copy(client, srv)
				_ = client.Close()
			}()
		}
	}()
	return p
}

func (p *switchProxy) cutLinks() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cut = true
	for _, c := range p.links {
		_ = c.Close()
	}
	p.links = nil
}

func (p *switchProxy) held() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waiting
}

func TestClient_StatusDoesNotWaitOnAReconnectStuckInItsHandshake(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, nil)
	proxy := newSwitchProxy(t, strings.TrimPrefix(url, "nats://"))

	conn, err := NewDialer().Dial(t.Context(), &entities.SavedConnection{
		URLs:       []string{proxy.url},
		Connection: &entities.ConnectionConfig{ConnectTimeout: new(5 * time.Second)},
		Reconnect:  &entities.ReconnectConfig{ReconnectWait: new(50 * time.Millisecond)},
	})
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	require.True(t, conn.IsConnected())

	proxy.cutLinks()
	require.Eventually(t, func() bool { return proxy.held() > 0 }, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	connected, reconnecting, status := conn.IsConnected(), conn.IsReconnecting(), conn.Status()
	health, err := conn.Health(t.Context())
	assert.Less(t, time.Since(start), 500*time.Millisecond, "reading the status waited for the reconnect attempt")
	require.NoError(t, err)
	assert.False(t, connected)
	assert.True(t, reconnecting)
	assert.Equal(t, "RECONNECTING", status)
	assert.Equal(t, "reconnecting", health.Status)
}
