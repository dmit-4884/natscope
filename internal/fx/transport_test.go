// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package fx

import (
	"testing"

	"github.com/altessa-s/go-atlas/config"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
)

// TestNewHTTPServer_WideBindRequiresAllowRemote covers the fix: the internal
// HTTP server (health/metrics/pprof) used to gate a non-loopback bind on
// AllowInsecure alone, unlike the main listener's two-flag AllowRemote +
// AllowInsecure gate — so ALLOW_INSECURE=true by itself exposed pprof
// (heap/goroutine/cmdline) to the network with no other setting required.
func TestNewHTTPServer_WideBindRequiresAllowRemote(t *testing.T) {
	cfg := &appconfig.Config{
		Http:          &config.Http{ListenAddress: "0.0.0.0:9080"},
		AllowInsecure: true,
		// AllowRemote intentionally left false.
	}

	if _, err := newHTTPServer(cfg, nil); err == nil {
		t.Fatal("newHTTPServer() on a wide bind without AllowRemote, want an error")
	}
}

// TestNewHTTPServer_WideBindRequiresAllowInsecure is the other half of the
// same gate: AllowRemote alone (no AllowInsecure) must also be refused,
// since the internal server has no basic-auth option of its own.
func TestNewHTTPServer_WideBindRequiresAllowInsecure(t *testing.T) {
	cfg := &appconfig.Config{
		Http:        &config.Http{ListenAddress: "0.0.0.0:9080"},
		AllowRemote: true,
		// AllowInsecure intentionally left false.
	}

	if _, err := newHTTPServer(cfg, nil); err == nil {
		t.Fatal("newHTTPServer() on a wide bind without AllowInsecure, want an error")
	}
}

// TestNewHTTPServer_NotConfiguredReturnsNil is the control: no http section
// at all must stay a no-op, not an error.
func TestNewHTTPServer_NotConfiguredReturnsNil(t *testing.T) {
	srv, err := newHTTPServer(&appconfig.Config{}, nil)
	if err != nil {
		t.Fatalf("newHTTPServer() error = %v, want nil", err)
	}
	if srv != nil {
		t.Fatalf("newHTTPServer() = %v, want nil", srv)
	}
}
