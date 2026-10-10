// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package appconfig provides natscope configuration management.
package appconfig

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-ozzo/ozzo-validation/v4"

	"github.com/altessa-s/go-atlas/config/http"
	"github.com/altessa-s/go-atlas/config/loader"
	"github.com/altessa-s/go-atlas/config/loader/backend/yaml3"
	"github.com/altessa-s/go-atlas/config/node"
	"github.com/altessa-s/go-atlas/config/observability"
	"github.com/altessa-s/go-atlas/config/validation"

	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
)

// Config is the main natscope service configuration.
type Config struct {
	Logger *observabilityconfig.Logger `yaml:"logger"`

	// GRPCWebAddress serves Connect/gRPC/gRPC-web plus the embedded SPA.
	GRPCWebAddress string `yaml:"grpcWebAddress" default:"127.0.0.1:4280"`

	// AllowRemote permits binding GRPCWebAddress to a non-loopback address.
	// Off by default: the API is unauthenticated unless WebAuth is set.
	AllowRemote bool `yaml:"allowRemote"`

	// AllowInsecure permits a non-loopback bind without WebAuth (fail-closed
	// otherwise). Off by default. Env ALLOW_INSECURE.
	AllowInsecure bool `yaml:"allowInsecure"`

	// AllowedHosts is a comma-separated list of extra Host names to accept (env ALLOWED_HOSTS).
	AllowedHosts string `yaml:"allowedHosts"`

	// WebAuth protects the whole listener (UI + API) with HTTP basic auth.
	WebAuth *WebAuthConfig `yaml:"webAuth" default:"-"`

	// Node supplies only a static service instance ID via Node.Id.
	Node *nodeconfig.Config `yaml:"node" default:"-"`

	Http *httpconfig.Config `yaml:"http" default:"-"`

	Metrics *observabilityconfig.Metrics `yaml:"metrics" default:"-"`

	// Storage. No default:"-" so the loader allocates it and applies nested
	// storage defaults even when the YAML omits the section.
	Storage *StorageConfig `yaml:"storage"`

	// Secrets selects the vault backend; allocated like Storage so nested
	// defaults apply when the YAML omits the section.
	Secrets *SecretsConfig `yaml:"secrets"`

	// MCP configures the /mcp endpoint; allocated like Storage so nested defaults apply.
	MCP *MCPConfig `yaml:"mcp"`
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
	// FileKey: hex AES-256 key for the file vault (64 chars); empty keeps vault.key beside the vault.
	FileKey string `yaml:"fileKey"`
}

// Validate restricts Backend to the known vault backends.
func (c *SecretsConfig) Validate() error {
	return validationconfig.ValidateStruct(c,
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

// SecretsFileKey returns the hex AES key for the file vault, or "" to use the key file beside the vault.
func (c *Config) SecretsFileKey() string {
	if c.Secrets == nil {
		return ""
	}
	return strings.TrimSpace(c.Secrets.FileKey)
}

// Load loads the configuration from the specified file or files.
func Load(filePath string, logger *observabilityconfig.Logger) (*Config, error) {
	if filePath != "" {
		if err := validateConfigPath(filePath); err != nil {
			return nil, err
		}
	}

	clearEmptyEnvOverrides()
	normalizeBoolEnv()

	opts := []loader.Option{
		loader.WithEnvPrefix(EnvPrefix()),
		// Strict mode fails the load on an undefined $VAR instead of truncating the value.
		loader.WithStrict(),
	}
	if filePath != "" {
		opts = append(opts, loader.WithPath(filePath))
	}

	cfg := loader.New(&yaml3.Backend{}, opts...)
	if _, err := cfg.Load(&Config{Logger: logger}); err != nil {
		return nil, err
	}

	conf := cfg.Config().(*Config) //nolint:errcheck

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return conf, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	return validationconfig.ValidateStruct(c,
		validation.Field(&c.Logger, validation.NilOrNotEmpty, validation.By(validateLoggerOutputFormat)),
		validation.Field(&c.Node, validation.NilOrNotEmpty),
		validation.Field(&c.Http, validation.NilOrNotEmpty, validation.By(validateHTTP), validation.Skip),
		validation.Field(&c.Metrics, validation.NilOrNotEmpty),
		validation.Field(&c.Storage, validation.NilOrNotEmpty),
		validation.Field(&c.WebAuth, validation.NilOrNotEmpty),
		validation.Field(&c.Secrets, validation.NilOrNotEmpty),
	)
}

// validateHTTP applies go-atlas's rules, except the listen address only has to parse as host:port.
func validateHTTP(value any) error {
	h, ok := value.(*httpconfig.Config)
	if !ok || h == nil {
		return nil
	}
	_, port, err := net.SplitHostPort(h.ListenAddress)
	if err != nil {
		return fmt.Errorf("listenAddress: %w", err)
	}
	probe := *h
	probe.ListenAddress = net.JoinHostPort("127.0.0.1", port)
	return probe.Validate()
}

// knownLogOutputFormats lists go-atlas's text and json formats plus [logconsole.Format].
var knownLogOutputFormats = []string{observabilityconfig.LogFormatText, observabilityconfig.LogFormatJSON, logconsole.Format}

// validateLoggerOutputFormat rejects an OutputFormat not in knownLogOutputFormats.
func validateLoggerOutputFormat(value any) error {
	l, ok := value.(*observabilityconfig.Logger)
	if !ok || l == nil || l.OutputFormat == "" {
		return nil
	}
	if slices.Contains(knownLogOutputFormats, l.OutputFormat) {
		return nil
	}
	return fmt.Errorf("outputFormat: unsupported value %q (expected one of %s)",
		l.OutputFormat, strings.Join(knownLogOutputFormats, ", "))
}

// AllowedHostsList returns the trimmed, non-empty entries of AllowedHosts.
func (c *Config) AllowedHostsList() []string {
	if c.AllowedHosts == "" {
		return nil
	}
	parts := strings.Split(c.AllowedHosts, ",")
	hosts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			hosts = append(hosts, p)
		}
	}
	return hosts
}

// WebAuthConfig holds HTTP basic auth credentials guarding the listener.
type WebAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Validate requires both credentials to be non-blank once the section is present.
func (c *WebAuthConfig) Validate() error {
	return validationconfig.ValidateStruct(c,
		validation.Field(&c.Username, validation.Required, validation.By(notBlank)),
		validation.Field(&c.Password, validation.Required, validation.By(notBlank)),
	)
}

// notBlank rejects a string that is empty after trimming whitespace.
func notBlank(value any) error {
	s, ok := value.(string)
	if ok && s != "" && strings.TrimSpace(s) == "" {
		return fmt.Errorf("cannot be blank")
	}
	return nil
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
// A leading "~" expands to the user's home directory.
func (c *Config) ResolveDataDir() string {
	if dir := c.GetLocalDataDir(); dir != "" {
		return expandHome(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".natscope", "data")
	}
	return filepath.Join(home, ".natscope", "data")
}

// expandHome expands a leading "~" to the user's home directory.
func expandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(home, rest)
	}
	return path
}

// ResolveBboltPath returns the bbolt database file path (<dataDir>/natscope.bolt).
func (c *Config) ResolveBboltPath() string {
	return filepath.Join(c.ResolveDataDir(), BboltDBFile)
}
