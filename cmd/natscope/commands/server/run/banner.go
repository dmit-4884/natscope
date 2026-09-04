// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"fmt"
	"io"
	"net"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dmit-4884/natscope/internal/pkg/secrets"
)

type banner struct {
	Version string
	Address string
	DataDir string
	Secrets string
	Home    string
}

func renderBanner(w io.Writer, b banner) {
	fmt.Fprintf(w, "Natscope %s\n\n", versionLabel(b.Version))
	fmt.Fprintf(w, "  %-8s  %s\n", "UI", uiURL(b.Address))
	fmt.Fprintf(w, "  %-8s  %s\n", "Data", collapseHome(b.DataDir, b.Home))
	fmt.Fprintf(w, "  %-8s  %s\n", "Secrets", b.Secrets)
	fmt.Fprint(w, "\nPress Ctrl+C to stop.\n")
}

func uiURL(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "http://" + address
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func collapseHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rel
	}
	return path
}

func versionLabel(version string) string {
	if version == "" || version == "0.0.0" {
		return "dev"
	}
	return "v" + strings.TrimPrefix(version, "v")
}

func secretsLabel(vault secrets.Vault) string {
	switch vault.(type) {
	case *secrets.Keyring:
		return keyringLabel(runtime.GOOS)
	case *secrets.File:
		return "encrypted file vault"
	default:
		return "unknown"
	}
}

func keyringLabel(goos string) string {
	switch goos {
	case "darwin":
		return "macOS Keychain"
	case "windows":
		return "Windows Credential Manager"
	default:
		return "OS keychain"
	}
}
