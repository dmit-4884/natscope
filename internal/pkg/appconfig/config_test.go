// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package appconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
)

// BindsLoopback must classify every address form correctly; a false positive
// here would let the fail-closed remote-bind gate be bypassed.
func TestBindsLoopback(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:4280", true},
		{"localhost:4280", true},
		{"[::1]:4280", true},
		{"[::ffff:127.0.0.1]:4280", true},
		{"0.0.0.0:4280", false},
		{":4280", false},
		{"[::]:4280", false},
		{"192.168.1.5:4280", false},
		{"10.0.0.9:4280", false},
		{"not-an-address", false},
	}
	for _, tc := range cases {
		c := &appconfig.Config{GRPCWebAddress: tc.addr}
		if got := c.BindsLoopback(); got != tc.want {
			t.Errorf("BindsLoopback(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

func TestWebAuthEnabled(t *testing.T) {
	cases := []struct {
		name string
		cfg  *appconfig.WebAuthConfig
		want bool
	}{
		{"nil", nil, false},
		{"both set", &appconfig.WebAuthConfig{Username: "u", Password: "p"}, true},
		{"empty password", &appconfig.WebAuthConfig{Username: "u", Password: ""}, false},
		{"empty username", &appconfig.WebAuthConfig{Username: "", Password: "p"}, false},
		{"whitespace password", &appconfig.WebAuthConfig{Username: "u", Password: "   "}, false},
		{"whitespace username", &appconfig.WebAuthConfig{Username: "\t", Password: "p"}, false},
	}
	for _, tc := range cases {
		if got := tc.cfg.Enabled(); got != tc.want {
			t.Errorf("%s: Enabled() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestWebAuthValidate_RejectsBlank(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *appconfig.WebAuthConfig
		wantErr bool
	}{
		{"both set", &appconfig.WebAuthConfig{Username: "u", Password: "p"}, false},
		{"whitespace password", &appconfig.WebAuthConfig{Username: "u", Password: "   "}, true},
		{"whitespace username", &appconfig.WebAuthConfig{Username: "\t", Password: "p"}, true},
	}
	for _, tc := range cases {
		err := tc.cfg.Validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: Validate() error = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestAllowedHostsList(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"example.com", []string{"example.com"}},
		{"example.com:4280, other.example:4280 ,,", []string{"example.com:4280", "other.example:4280"}},
	}
	for _, tc := range cases {
		c := &appconfig.Config{AllowedHosts: tc.in}
		got := c.AllowedHostsList()
		if len(got) != len(tc.want) {
			t.Fatalf("AllowedHostsList(%q) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("AllowedHostsList(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestLoad_RejectsUnknownOutputFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("logger:\n  outputFormat: xml\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := appconfig.Load(path, appconfig.LoggerDefaults(false)); err == nil {
		t.Fatal("Load() with outputFormat: xml, want an error")
	}
}

func TestLoad_RejectsUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.conf")
	if err := os.WriteFile(path, []byte("grpcWebAddress: \"127.0.0.1:1\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := appconfig.Load(path, appconfig.LoggerDefaults(false)); err == nil {
		t.Fatal("Load() with a .conf file, want an error")
	}
}

func TestLoad_RejectsUnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "storage:\n  local:\n    datadir: " + dir + "\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := appconfig.Load(path, appconfig.LoggerDefaults(false)); err == nil {
		t.Fatal("Load() with an unknown key, want an error")
	}
}

func TestLoad_RejectsUndefinedEnvSubstitution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "webAuth:\n  username: qa-user\n  password: \"qa-secret-pa$word-7f3a\"\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := appconfig.Load(path, appconfig.LoggerDefaults(false)); err == nil {
		t.Fatal("Load() with an undefined $word reference, want an error")
	}
}

func TestLoad_EmptyEnvOverrideIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "storage:\n  local:\n    dataDir: " + dir + "\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("STORAGE__LOCAL__DATA_DIR", "")

	cfg, err := appconfig.Load(path, appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.GetLocalDataDir() != dir {
		t.Errorf("GetLocalDataDir() = %q, want %q (empty env override must be ignored)", cfg.GetLocalDataDir(), dir)
	}
}

func TestResolveDataDir_ExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &appconfig.Config{Storage: &appconfig.StorageConfig{
		Local: &appconfig.LocalStorageConfig{DataDir: "~/.natscope/data-tilde"},
	}}
	want := filepath.Join(home, ".natscope", "data-tilde")
	if got := cfg.ResolveDataDir(); got != want {
		t.Errorf("ResolveDataDir() = %q, want %q", got, want)
	}
}

func TestLoad_BoolEnvMatchesYAMLWords(t *testing.T) {
	t.Setenv("ALLOW_INSECURE", "yes")

	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.AllowInsecure {
		t.Error("ALLOW_INSECURE=yes must enable allowInsecure")
	}
}

func TestLoad_SecretsFileKeyFromEnv(t *testing.T) {
	key := strings.Repeat("ab", 32)
	t.Setenv("SECRETS__FILE_KEY", key)

	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := cfg.SecretsFileKey(); got != key {
		t.Errorf("SecretsFileKey() = %q, want %q", got, key)
	}
}

// IPv6 loopback is accepted; a bare ":port" counts as a non-loopback bind.
func TestLoad_HTTPListenAddressForms(t *testing.T) {
	t.Setenv("HTTP__LISTEN_ADDRESS", "[::1]:9080")
	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load [::1]:9080: %v", err)
	}
	if !cfg.HTTPBindsLoopback() {
		t.Error("[::1]:9080 must count as a loopback bind")
	}

	t.Setenv("HTTP__LISTEN_ADDRESS", ":9080")
	cfg, err = appconfig.Load("", appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load :9080: %v", err)
	}
	if cfg.HTTPBindsLoopback() {
		t.Error(":9080 binds every interface")
	}
}

func TestMCPDefaults(t *testing.T) {
	if cfg := (&appconfig.Config{}); !cfg.MCPEnabled() || cfg.MCPAllowWrites() {
		t.Error("an unset mcp section must mean enabled, read-only")
	}

	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.MCPEnabled() {
		t.Error("mcp must be enabled by default")
	}
	if cfg.MCPAllowWrites() {
		t.Error("mcp writes must be off by default")
	}
}

func TestLoad_MCPPartialSectionKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("mcp:\n  allowWrites: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := appconfig.Load(path, appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.MCPEnabled() {
		t.Error("enabled must keep its default when the section sets only allowWrites")
	}
	if !cfg.MCPAllowWrites() {
		t.Error("allowWrites: true must be honored")
	}

	t.Setenv("MCP__ENABLED", "off")
	cfg, err = appconfig.Load(path, appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.MCPEnabled() {
		t.Error("MCP__ENABLED=off must disable the endpoint")
	}
}

func TestLoad_MCPDisabledInYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("mcp:\n  enabled: false\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := appconfig.Load(path, appconfig.LoggerDefaults(false))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.MCPEnabled() {
		t.Error("enabled: false must disable the endpoint")
	}
}
