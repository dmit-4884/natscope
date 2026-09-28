// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

// acceptAndHold accepts every connection on ln and holds it open without
// speaking, until ln is closed by the caller. Run in a goroutine; exits once
// Accept starts failing (ln closed).
func acceptAndHold(ln net.Listener) {
	var conns []net.Conn
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		conns = append(conns, c)
	}
}

// TestConnection_ConnectTimeoutUnitsAreDurationNotMultiplied is the QA-004
// regression: ConnectTimeout is already a time.Duration; it must not be
// re-multiplied by time.Millisecond. A "quiet" TCP listener that accepts and
// never speaks is used so the probe only returns once its timeout elapses.
func TestConnection_ConnectTimeoutUnitsAreDurationNotMultiplied(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	go acceptAndHold(ln)

	d := NewDialer()
	start := time.Now()
	result, err := d.TestConnection(t.Context(), &entities.TestConnectionRequest{
		URLs:           []string{"nats://" + ln.Addr().String()},
		ConnectTimeout: durationPtr(300 * time.Millisecond),
	})
	require.NoError(t, err)
	require.False(t, result.Success)
	elapsed := time.Since(start)
	require.Less(t, elapsed, 5*time.Second, "connect timeout must be ~300ms, not ~3.5 days")
}

// TestConnection_CtxCancelCutsProbeShort is the QA-005 regression: the
// caller's context bounds the whole call, even when ConnectTimeout itself is
// large (today: TestConnection ignores ctx entirely and blocks for the full
// per-URL timeout, holding the socket open the whole time — a client that
// gives up leaves the goroutine/socket running for the requested
// ConnectTimeout, unbounded by anything the caller controls).
func TestConnection_CtxCancelCutsProbeShort(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	go acceptAndHold(ln)

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	d := NewDialer()
	start := time.Now()
	result, err := d.TestConnection(ctx, &entities.TestConnectionRequest{
		URLs:           []string{"nats://" + ln.Addr().String()},
		ConnectTimeout: durationPtr(30 * time.Second), // would hang 30s without the ctx bound
	})
	require.NoError(t, err)
	require.False(t, result.Success)
	elapsed := time.Since(start)
	require.Less(t, elapsed, 5*time.Second, "ctx cancellation must cut the probe short, not wait out ConnectTimeout")
}

// TestConnection_NkeySeedErrorSurfacesAsFailure is the QA-038 regression: an
// unparseable NKey seed must not be silently dropped (falling back to
// anonymous auth while reporting success); it must fail with an error naming
// the seed.
func TestConnection_NkeySeedErrorSurfacesAsFailure(t *testing.T) {
	t.Parallel()

	d := NewDialer()
	result, err := d.TestConnection(t.Context(), &entities.TestConnectionRequest{
		URLs: []string{"nats://127.0.0.1:1"},
		Auth: &entities.AuthConfig{
			Method:   entities.AuthMethodNKey,
			NkeySeed: strPtr("not-a-valid-seed"),
		},
	})
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, strings.ToLower(result.Error), "seed")
}

// TestConnection_MalformedHTTPResponseIsSanitized is the QA-036 regression: a
// non-NATS/non-WebSocket listener's raw banner must not be echoed back to the
// caller (TestConnection was a banner-grab primitive against ws:// targets).
func TestConnection_MalformedHTTPResponseIsSanitized(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	const banner = "SSH-2.0-QA_BANNER_SECRET\r\n"
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Write([]byte(banner))
	}()

	d := NewDialer()
	result, err := d.TestConnection(t.Context(), &entities.TestConnectionRequest{
		URLs: []string{"ws://" + ln.Addr().String()},
	})
	require.NoError(t, err)
	require.False(t, result.Success)
	require.NotContains(t, result.Error, "QA_BANNER_SECRET")
}

func durationPtr(d time.Duration) *time.Duration { return &d }
func strPtr(s string) *string                    { return &s }
