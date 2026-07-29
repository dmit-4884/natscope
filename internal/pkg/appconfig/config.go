// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package appconfig provides natscope configuration management.
package appconfig

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ozzo/ozzo-validation/v4"

	"github.com/altessa-s/go-atlas/config"
	"github.com/altessa-s/go-atlas/config/loader"
	"github.com/altessa-s/go-atlas/config/loader/backend/yaml3"
	"github.com/altessa-s/go-atlas/core/runtime/appinfo"
)

// Config is the main natscope service configuration.
type Config struct {
	Logger *config.Logger `yaml:"logger"`

	// GRPCWebAddress serves Connect/gRPC/gRPC-web plus the embedded SPA.
	GRPCWebAddress string `yaml:"grpcWebAddress" default:"127.0.0.1:4280"`

	// AllowRemote permits binding GRPCWebAddress to a non-loopback address.
	// Off by default: the API is unauthenticated unless WebAuth is set.
	AllowRemote bool `yaml:"allowRemote"`

	// AllowInsecure permits a non-loopback bind without WebAuth (fail-closed
	// otherwise). Off by default. Env ALLOW_INSECURE.
	AllowInsecure bool `yaml:"allowInsecure"`

	// WebAuth protects the whole listener (UI + API) with HTTP basic auth.
	WebAuth *WebAuthConfig `yaml:"webAuth" default:"-"`

	// Node supplies only a static service instance ID via Node.Id.
	Node *config.Node `yaml:"node" default:"-"`

	Http *config.Http `yaml:"http" default:"-"`

	Metrics *config.Metrics `yaml:"metrics" default:"-"`

	// Storage. No default:"-" so the loader allocates it and applies nested
	// storage defaults even when the YAML omits the section.
	Storage *StorageConfig `yaml:"storage"`

	// Secrets selects the vault backend; allocated like Storage so nested
	// defaults apply when the YAML omits the section.
	Secrets *SecretsConfig `yaml:"secrets"`
}

// Secret vault backends: auto probes the OS keychain and falls back to the
// encrypted file vault (headless Linux, containers).
const (
	SecretsBackendAuto    = "auto"
	SecretsBackendKeyring = "keyring"
	SecretsBackendFile    = "file"
)

// SecretsConfig selects where secrets live.
type SecretsConfig struct {
	// Backend: auto | keyring | file.
	Backend string `yaml:"backend" default:"auto"`
}

// Validate restricts Backend to the known vault backends.
func (c *SecretsConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Backend, validation.In(
			SecretsBackendAuto, SecretsBackendKeyring, SecretsBackendFile)),
	)
}

// SecretsBackend returns the configured vault backend, defaulting to auto.
func (c *Config) SecretsBackend() string {
	if c.Secrets != nil && c.Secrets.Backend != "" {
		return c.Secrets.Backend
	}
	return SecretsBackendAuto
}

// Load loads the configuration from the specified file or files.
func Load(filePath string) (*Config, error) {
	opts := []loader.Option{
		loader.WithEnvPrefix(appinfo.EnvPrefix),
	}
	if filePath != "" {
		opts = append(opts, loader.WithPath(filePath))
	}

	cfg := loader.New(&yaml3.Backend{}, opts...)
	if _, err := cfg.Load((*Config)(nil)); err != nil {
		return nil, err
	}

	conf := cfg.Config().(*Config) //nolint:errcheck

	// Dev-friendly logger defaults; env vars still override (LOGGER__LEVEL).
	if conf.Logger != nil {
		if conf.Logger.Level == config.LoggerLevelError {
			conf.Logger.Level = config.LoggerLevelInfo
		}
		if conf.Logger.OutputFormat == config.LogFormatText {
			conf.Logger.Colorized = true
		}
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return conf, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Logger, validation.NilOrNotEmpty),
		validation.Field(&c.Node, validation.NilOrNotEmpty),
		validation.Field(&c.Http, validation.NilOrNotEmpty),
		validation.Field(&c.Metrics, validation.NilOrNotEmpty),
		validation.Field(&c.Storage, validation.NilOrNotEmpty),
		validation.Field(&c.WebAuth, validation.NilOrNotEmpty),
		validation.Field(&c.Secrets, validation.NilOrNotEmpty),
	)
}

// WebAuthConfig holds HTTP basic auth credentials guarding the listener.
type WebAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Validate requires both credentials once the section is present.
func (c *WebAuthConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Username, validation.Required),
		validation.Field(&c.Password, validation.Required),
	)
}

// Enabled reports whether basic auth credentials are configured. Whitespace-only
// values do not count, so they can't satisfy the fail-closed remote-bind gate.
func (c *WebAuthConfig) Enabled() bool {
	return c != nil && strings.TrimSpace(c.Username) != "" && strings.TrimSpace(c.Password) != ""
}

// BindsLoopback reports whether GRPCWebAddress binds only a loopback
// interface. Empty or unresolvable hosts count as non-loopback (":4280"
// binds every interface).
func (c *Config) BindsLoopback() bool {
	return addrBindsLoopback(c.GRPCWebAddress)
}

// HTTPBindsLoopback reports whether the internal HTTP server (health/metrics/
// pprof), when enabled, binds only a loopback interface.
func (c *Config) HTTPBindsLoopback() bool {
	if c.Http == nil {
		return true
	}
	return addrBindsLoopback(c.Http.ListenAddress)
}

// addrBindsLoopback reports whether addr binds only a loopback interface.
// Empty or unresolvable hosts count as non-loopback (":9080" binds every
// interface).
func addrBindsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// IsHttpConfigured checks if HTTP server is configured.
func (c *Config) IsHttpConfigured() bool {
	return c.Http != nil
}

// GetLocalDataDir returns the local storage data directory.
func (c *Config) GetLocalDataDir() string {
	if c.Storage != nil {
		return c.Storage.GetDataDir()
	}
	return ""
}

// BboltDBFile is the database file name under the data dir.
const BboltDBFile = "natscope.bolt"

// ResolveDataDir returns the local data directory, defaulting to ~/.natscope/data.
func (c *Config) ResolveDataDir() string {
	if dir := c.GetLocalDataDir(); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".natscope", "data")
	}
	return filepath.Join(home, ".natscope", "data")
}

// ResolveBboltPath returns the bbolt database file path (<dataDir>/natscope.bolt).
func (c *Config) ResolveBboltPath() string {
	return filepath.Join(c.ResolveDataDir(), BboltDBFile)
}
