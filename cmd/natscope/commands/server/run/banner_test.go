// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/secrets"
)

func TestRenderBanner(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderBanner(&buf, banner{
		Version: "0.2.0",
		Address: "127.0.0.1:4280",
		DataDir: "/Users/dev/.natscope/data",
		Secrets: "macOS Keychain",
		Home:    "/Users/dev",
	})

	want := "Natscope v0.2.0\n" +
		"\n" +
		"  UI        http://127.0.0.1:4280\n" +
		"  Data      ~/.natscope/data\n" +
		"  Secrets   macOS Keychain\n" +
		"\n" +
		"Press Ctrl+C to stop.\n"
	require.Equal(t, want, buf.String())
}

func TestUIURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		address string
		want    string
	}{
		{"127.0.0.1:4280", "http://127.0.0.1:4280"},
		{"localhost:9090", "http://localhost:9090"},
		{"0.0.0.0:4280", "http://localhost:4280"},
		{":4280", "http://localhost:4280"},
		{"[::]:4280", "http://localhost:4280"},
		{"[::1]:4280", "http://[::1]:4280"},
		{"192.168.1.5:4280", "http://192.168.1.5:4280"},
	}
	for _, tc := range cases {
		t.Run(tc.address, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, uiURL(tc.address))
		})
	}
}

func TestCollapseHome(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		home string
		want string
	}{
		{"under home", "/Users/dev/.natscope/data", "/Users/dev", "~/.natscope/data"},
		{"home itself", "/Users/dev", "/Users/dev", "~"},
		{"outside home", "/var/lib/natscope", "/Users/dev", "/var/lib/natscope"},
		{"prefix but not a path boundary", "/Users/developer/x", "/Users/dev", "/Users/developer/x"},
		{"unknown home", "/Users/dev/x", "", "/Users/dev/x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, collapseHome(tc.path, tc.home))
		})
	}
}

func TestVersionLabel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		version string
		want    string
	}{
		{"0.2.0", "v0.2.0"},
		{"v0.2.0", "v0.2.0"},
		{"0.0.0", "dev"},
		{"", "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, versionLabel(tc.version))
		})
	}
}

func TestSecretsLabel(t *testing.T) {
	t.Setenv("SECRETS__FILE_KEY", "")

	fileVault, err := secrets.NewFile(t.TempDir())
	require.NoError(t, err)

	require.Equal(t, keyringLabel(runtime.GOOS), secretsLabel(secrets.NewKeyring("natscope-test")))
	require.Equal(t, "encrypted file vault", secretsLabel(fileVault))
	require.Equal(t, "unknown", secretsLabel(secrets.NewMemory()))
}

func TestKeyringLabel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		goos string
		want string
	}{
		{"darwin", "macOS Keychain"},
		{"windows", "Windows Credential Manager"},
		{"linux", "OS keychain"},
	}
	for _, tc := range cases {
		t.Run(tc.goos, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, keyringLabel(tc.goos))
		})
	}
}
