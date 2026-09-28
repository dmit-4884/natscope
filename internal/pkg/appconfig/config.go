// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package appconfig provides natscope configuration management.
package appconfig

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ozzo/ozzo-validation/v4"

	"github.com/altessa-s/go-atlas/config"
	"github.com/altessa-s/go-atlas/config/loader"
	"github.com/altessa-s/go-atlas/config/loader/backend/yaml3"
	"github.com/altessa-s/go-atlas/core/runtime/appinfo"

	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
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

	// AllowedHosts is a comma-separated allowlist of extra Host header values
	// accepted on a non-loopback bind, on top of GRPCWebAddress itself (DNS
	// rebinding protection). Ignored on a loopback bind, where only
	// 127.0.0.1/[::1]/localhost at the bound port are ever accepted. Env
	// ALLOWED_HOSTS.
	AllowedHosts string `yaml:"allowedHosts"`

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
func Load(filePath string, logger *config.Logger) (*Config, error) {
	if filePath != "" {
		if err := validateConfigPath(filePath); err != nil {
			return nil, err
		}
	}

	// An env var explicitly set to "" (common in docker-compose/systemd
	// templates that interpolate an unset variable) must not silently
	// override a configured value with the field's zero default.
	clearEmptyEnvOverrides()

	opts := []loader.Option{
		loader.WithEnvPrefix(appinfo.EnvPrefix),
		// Strict mode turns a "$VAR" reference to an undefined environment
		// variable into a load error instead of silently truncating the
		// value at the "$" — the substitution has no escape/opt-out and
		// would otherwise mangle any secret containing a literal "$".
		loader.WithStrict(),
	}
	if filePath != "" {
		opts = append(opts, loader.WithPath(filePath))
	}

	cfg := loader.New(&yaml3.Backend{}, opts...)
	if _, err := cfg.Load(&Config{Logger: logger}); err != nil {
		return nil, err
	}

	if filePath != "" {
		if err := checkKnownFields(filePath); err != nil {
			return nil, err
		}
	}

	conf := cfg.Config().(*Config) //nolint:errcheck

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return conf, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Logger, validation.NilOrNotEmpty, validation.By(validateLoggerOutputFormat)),
		validation.Field(&c.Node, validation.NilOrNotEmpty),
		validation.Field(&c.Http, validation.NilOrNotEmpty),
		validation.Field(&c.Metrics, validation.NilOrNotEmpty),
		validation.Field(&c.Storage, validation.NilOrNotEmpty),
		validation.Field(&c.WebAuth, validation.NilOrNotEmpty),
		validation.Field(&c.Secrets, validation.NilOrNotEmpty),
	)
}

// knownLogOutputFormats lists the output formats natscope actually supports:
// go-atlas's own text/json, plus the "console" format registered by
// [logconsole.Register]. go-atlas's Logger.Validate does not check
// OutputFormat at all, so an unsupported value (typo, or the documented but
// unimplemented "console" before this package registered it) silently fell
// back to text.
var knownLogOutputFormats = []string{config.LogFormatText, config.LogFormatJSON, logconsole.Format}

// validateLoggerOutputFormat rejects an OutputFormat that resolves to neither
// go-atlas's built-in handlers nor natscope's own "console" one.
func validateLoggerOutputFormat(value any) error {
	l, ok := value.(*config.Logger)
	if !ok || l == nil || l.OutputFormat == "" {
		return nil
	}
	for _, known := range knownLogOutputFormats {
		if l.OutputFormat == known {
			return nil
		}
	}
	return fmt.Errorf("outputFormat: unsupported value %q (expected one of %s)",
		l.OutputFormat, strings.Join(knownLogOutputFormats, ", "))
}

// AllowedHostsList splits AllowedHosts on commas, trimming whitespace and
// dropping empty entries.
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

// Validate requires both credentials once the section is present. A
// whitespace-only value passes ozzo's Required (it only checks for the zero
// value) but fails Enabled(), which would otherwise start the listener
// without auth and without a warning; reject it here instead.
func (c *WebAuthConfig) Validate() error {
	return validation.ValidateStruct(c,
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
// A leading "~" is expanded to the user's home directory: the configured
// default itself is "~/.natscope/data", and without expansion that literal
// tilde becomes a directory name under the current working directory instead.
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

// expandHome expands a leading "~" (or "~/...") to the user's home
// directory; any other path is returned unchanged.
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
