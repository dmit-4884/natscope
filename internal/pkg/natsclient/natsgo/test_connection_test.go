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

// acceptAndHold accepts and silently holds connections on ln until it is closed.
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

// TestConnection_ConnectTimeoutUnitsAreDurationNotMultiplied checks that ConnectTimeout isn't scaled by time.Millisecond.
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

// TestConnection_CtxCancelCutsProbeShort checks that ctx cancellation ends the probe before ConnectTimeout.
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
		ConnectTimeout: durationPtr(30 * time.Second),
	})
	require.NoError(t, err)
	require.False(t, result.Success)
	elapsed := time.Since(start)
	require.Less(t, elapsed, 5*time.Second, "ctx cancellation must cut the probe short, not wait out ConnectTimeout")
}

// TestConnection_NkeySeedErrorSurfacesAsFailure checks that an unparsable NKey seed fails with an error naming it.
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

// TestConnection_MalformedHTTPResponseIsSanitized checks that a foreign listener's banner isn't echoed back.
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
