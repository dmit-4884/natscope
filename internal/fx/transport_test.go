// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package fx

import (
	"testing"

	"github.com/altessa-s/go-atlas/config/http"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
)

// TestNewHTTPServer_WideBindRequiresAllowRemote checks that AllowInsecure alone can't open a wide bind.
func TestNewHTTPServer_WideBindRequiresAllowRemote(t *testing.T) {
	cfg := &appconfig.Config{
		Http:          &httpconfig.Config{ListenAddress: "0.0.0.0:9080"},
		AllowInsecure: true,
		// AllowRemote intentionally left false.
	}

	if _, err := newHTTPServer(cfg, nil); err == nil {
		t.Fatal("newHTTPServer() on a wide bind without AllowRemote, want an error")
	}
}

// TestNewHTTPServer_WideBindRequiresAllowInsecure checks that AllowRemote alone can't open a wide bind.
func TestNewHTTPServer_WideBindRequiresAllowInsecure(t *testing.T) {
	cfg := &appconfig.Config{
		Http:        &httpconfig.Config{ListenAddress: "0.0.0.0:9080"},
		AllowRemote: true,
		// AllowInsecure intentionally left false.
	}

	if _, err := newHTTPServer(cfg, nil); err == nil {
		t.Fatal("newHTTPServer() on a wide bind without AllowInsecure, want an error")
	}
}

// TestNewHTTPServer_NotConfiguredReturnsNil checks that a missing http section is a no-op.
func TestNewHTTPServer_NotConfiguredReturnsNil(t *testing.T) {
	srv, err := newHTTPServer(&appconfig.Config{}, nil)
	if err != nil {
		t.Fatalf("newHTTPServer() error = %v, want nil", err)
	}
	if srv != nil {
		t.Fatalf("newHTTPServer() = %v, want nil", srv)
	}
}
