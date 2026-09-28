// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package fx

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
)

// TestNewSecretsVault_KeyringUnavailableFailsClosed covers QA-090: an
// explicit SECRETS__BACKEND=keyring used to skip the availability probe that
// "auto" already had, so the server started "healthy" and the first
// secret-reading RPC failed as an unmapped internal error instead of a clear
// startup failure naming the real cause.
func TestNewSecretsVault_KeyringUnavailableFailsClosed(t *testing.T) {
	probeErr := errors.New("dbus-launch: executable file not found in $PATH")
	keyring.MockInitWithError(probeErr)
	t.Cleanup(keyring.MockInit)

	cfg := &appconfig.Config{Secrets: &appconfig.SecretsConfig{Backend: appconfig.SecretsBackendKeyring}}

	_, err := newSecretsVault(cfg)
	if err == nil {
		t.Fatal("newSecretsVault() with an unavailable keychain, want an error")
	}
	if !errors.Is(err, probeErr) {
		t.Errorf("newSecretsVault() error = %v, want it to wrap %v", err, probeErr)
	}
}

// TestNewSecretsVault_KeyringAvailableSucceeds is the control: an explicit
// keyring backend that does work must still be usable.
func TestNewSecretsVault_KeyringAvailableSucceeds(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)

	cfg := &appconfig.Config{Secrets: &appconfig.SecretsConfig{Backend: appconfig.SecretsBackendKeyring}}

	vault, err := newSecretsVault(cfg)
	if err != nil {
		t.Fatalf("newSecretsVault() error = %v, want nil", err)
	}
	if vault == nil {
		t.Fatal("newSecretsVault() returned a nil vault with no error")
	}
}
