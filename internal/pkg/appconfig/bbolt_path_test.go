// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
)

// With no config file the path falls back to the default data dir.
func TestResolveBboltPath_Default(t *testing.T) {
	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !strings.HasSuffix(cfg.ResolveBboltPath(), "natscope.bolt") {
		t.Errorf("path = %q, want it to end with natscope.bolt", cfg.ResolveBboltPath())
	}
}

// A configured data dir places the bbolt file under it.
func TestResolveBboltPath_DataDirOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "storage:\n  local:\n    dataDir: " + dir + "\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := appconfig.Load(path, appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if want := filepath.Join(dir, "natscope.bolt"); cfg.ResolveBboltPath() != want {
		t.Errorf("path = %q, want %q", cfg.ResolveBboltPath(), want)
	}
}
