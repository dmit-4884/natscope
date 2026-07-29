// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"log/slog"
	"strings"
	"time"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"
)

// AuthMethod is the auth method; values match proto enum
// natscope.types.nats.AuthMethod.
type AuthMethod int32

const (
	AuthMethodNone        AuthMethod = 0
	AuthMethodUserPass    AuthMethod = 1
	AuthMethodToken       AuthMethod = 2
	AuthMethodNKey        AuthMethod = 3
	AuthMethodCredentials AuthMethod = 4
)

// AuthConfig holds authentication configuration for a NATS connection.
type AuthConfig struct {
	Method      AuthMethod
	Username    *string
	Password    *string
	Token       *string
	NkeySeed    *string
	Credentials *string
	JWT         *string
}

// LogValue masks credentials in slog: only method + "has X" presence flags,
// never raw secrets.
func (a *AuthConfig) LogValue() slog.Value {
	if a == nil {
		return slog.GroupValue(slog.Bool("nil", true))
	}
	hasStr := func(p *string) bool { return p != nil && *p != "" }
	return slog.GroupValue(
		slog.Int("method", int(a.Method)),
		slog.Bool("has_username", hasStr(a.Username)),
		slog.Bool("has_password", hasStr(a.Password)),
		slog.Bool("has_token", hasStr(a.Token)),
		slog.Bool("has_nkey_seed", hasStr(a.NkeySeed)),
		slog.Bool("has_credentials", hasStr(a.Credentials)),
		slog.Bool("has_jwt", hasStr(a.JWT)),
	)
}

// IsEmpty reports no auth intent (nil, or Method=None with no material); folds
// "user cleared auth" to nil at the storage edge.
func (a *AuthConfig) IsEmpty() bool {
	if a == nil {
		return true
	}
	if a.Method != AuthMethodNone {
		return false
	}
	emptyOrNil := func(p *string) bool { return p == nil || *p == "" }
	return emptyOrNil(a.Username) &&
		emptyOrNil(a.Password) &&
		emptyOrNil(a.Token) &&
		emptyOrNil(a.NkeySeed) &&
		emptyOrNil(a.Credentials) &&
		emptyOrNil(a.JWT)
}

// TlsConfig holds TLS configuration for a NATS connection.
type TlsConfig struct {
	CaCert     *string
	ClientCert *string
	ClientKey  *string
	SkipVerify bool
	TlsFirst   bool
}

// LogValue masks TLS material in slog: only presence flags and non-secret
// booleans, never cert/key bytes.
func (t *TlsConfig) LogValue() slog.Value {
	if t == nil {
		return slog.GroupValue(slog.Bool("nil", true))
	}
	hasStr := func(p *string) bool { return p != nil && *p != "" }
	return slog.GroupValue(
		slog.Bool("has_ca_cert", hasStr(t.CaCert)),
		slog.Bool("has_client_cert", hasStr(t.ClientCert)),
		slog.Bool("has_client_key", hasStr(t.ClientKey)),
		slog.Bool("skip_verify", t.SkipVerify),
		slog.Bool("tls_first", t.TlsFirst),
	)
}

// IsEmpty reports no user-provided TLS content; folds "user removed TLS" to nil
// at the storage edge.
func (t *TlsConfig) IsEmpty() bool {
	if t == nil {
		return true
	}
	emptyOrNil := func(p *string) bool { return p == nil || *p == "" }
	return emptyOrNil(t.CaCert) &&
		emptyOrNil(t.ClientCert) &&
		emptyOrNil(t.ClientKey) &&
		!t.SkipVerify &&
		!t.TlsFirst
}

// ConnectionConfig holds connection behavior settings.
type ConnectionConfig struct {
	ConnectTimeout          *time.Duration
	ConnectionName          *string
	InboxPrefix             *string
	NoEcho                  bool
	NoRandomize             bool
	IgnoreDiscoveredServers bool
}

// IsEmpty reports no user-provided settings; folds "user cleared connection
// settings" to nil at the storage edge.
func (c *ConnectionConfig) IsEmpty() bool {
	if c == nil {
		return true
	}
	emptyOrNil := func(p *string) bool { return p == nil || *p == "" }
	return c.ConnectTimeout == nil &&
		emptyOrNil(c.ConnectionName) &&
		emptyOrNil(c.InboxPrefix) &&
		!c.NoEcho &&
		!c.NoRandomize &&
		!c.IgnoreDiscoveredServers
}

// ReconnectConfig holds reconnection behavior settings.
type ReconnectConfig struct {
	MaxReconnects        *int32
	ReconnectWait        *time.Duration
	ReconnectBufSize     *int32
	RetryOnFailedConnect bool
}

// IsEmpty reports whether the reconnect config carries no user overrides.
func (r *ReconnectConfig) IsEmpty() bool {
	if r == nil {
		return true
	}
	return r.MaxReconnects == nil &&
		r.ReconnectWait == nil &&
		r.ReconnectBufSize == nil &&
		!r.RetryOnFailedConnect
}

// PingConfig holds ping/pong health check settings.
type PingConfig struct {
	PingInterval        *time.Duration
	MaxPingsOutstanding *int32
}

// IsEmpty reports whether the ping config carries no user overrides.
func (p *PingConfig) IsEmpty() bool {
	if p == nil {
		return true
	}
	return p.PingInterval == nil && p.MaxPingsOutstanding == nil
}

// ConnectionMeta is the most recent server-probe snapshot; written by the
// connections service on Test, never accepted from clients.
type ConnectionMeta struct {
	LastTestedAt     time.Time
	LastSuccess      bool
	LastRTTMs        *int64
	ServerVersion    *string
	ServerName       *string
	ServerID         *string
	ClusterName      *string
	MaxPayload       *int64
	JetstreamEnabled *bool
	ConnectedURL     *string
	LastError        *string
}

// NewConnectionMetaFromTestResult builds a snapshot: success populates
// server-info fields, failure records only LastError.
func NewConnectionMetaFromTestResult(result *TestConnectionResult) *ConnectionMeta {
	meta := &ConnectionMeta{
		LastTestedAt: time.Now().UTC(),
		LastSuccess:  result.Success,
	}
	if result.Success {
		meta.LastRTTMs = ptr.WrapNonZero(result.RTTMs)
		meta.ServerVersion = ptr.WrapNonZero(result.ServerVersion)
		meta.ServerName = ptr.WrapNonZero(result.ServerName)
		meta.ServerID = ptr.WrapNonZero(result.ServerID)
		meta.ClusterName = ptr.WrapNonZero(result.ClusterName)
		meta.MaxPayload = ptr.WrapNonZero(result.MaxPayload)
		meta.JetstreamEnabled = ptr.Wrap(result.JetstreamEnabled)
		meta.ConnectedURL = ptr.WrapNonZero(result.ConnectedURL)
	} else {
		meta.LastError = ptr.WrapNonZero(result.Error)
	}
	return meta
}

// SavedConnection is a persisted NATS connection with auth, TLS, and behavior settings.
type SavedConnection struct {
	BaseEntity

	// Name is the human-readable name for this connection.
	Name string

	// Description is an optional user note about this connection.
	Description *string

	// URLs is the list of NATS server URLs.
	URLs []string

	// Auth holds authentication configuration.
	Auth *AuthConfig

	// TLS holds TLS configuration.
	TLS *TlsConfig

	// Connection holds connection behavior settings.
	Connection *ConnectionConfig

	// Reconnect holds reconnection behavior settings.
	Reconnect *ReconnectConfig

	// Ping holds ping/pong health check settings.
	Ping *PingConfig

	// Meta is the most recent server-probe snapshot; set by the connections
	// service, never by clients.
	Meta *ConnectionMeta
}

// SavedConnectionNew creates a new SavedConnection with generated Id and
// timestamps.
func SavedConnectionNew(init ...func(*SavedConnection)) *SavedConnection {
	c := &SavedConnection{
		BaseEntity: *New(),
	}

	if len(init) > 0 && init[0] != nil {
		init[0](c)
	}

	return c
}

// ApplyUpdate merges the update then collapses empty sub-trees to nil, since
// converter treats a non-nil zero sub-tree as a full replace, not "no override".
func (c *SavedConnection) ApplyUpdate(req *SavedConnectionUpdate) {
	if c == nil || req == nil {
		return
	}

	converter.Convert(req, c,
		converter.WithIgnoreNilValues(),
		converter.WithIgnoreFields("etag"))

	if c.Auth.IsEmpty() {
		c.Auth = nil
	}
	if c.TLS.IsEmpty() {
		c.TLS = nil
	}
	if c.Connection.IsEmpty() {
		c.Connection = nil
	}
	if c.Reconnect.IsEmpty() {
		c.Reconnect = nil
	}
	if c.Ping.IsEmpty() {
		c.Ping = nil
	}

	c.BeforeUpdate()
}

// PrimaryURL returns the first URL or empty string.
func (c *SavedConnection) PrimaryURL() string {
	if len(c.URLs) > 0 {
		return c.URLs[0]
	}
	return ""
}

// URLsString returns comma-separated URLs for nats.Connect.
func (c *SavedConnection) URLsString() string {
	return strings.Join(c.URLs, ",")
}

// SavedConnections is a slice of SavedConnection pointers.
type SavedConnections []*SavedConnection

// SavedConnectionsList is the listing filter for saved connections.
type SavedConnectionsList struct {
	ListBase
}

// SavedConnectionCreate is the create DTO for a saved connection.
type SavedConnectionCreate struct {
	Name        string  `normalize:"trim"`
	Description *string `normalize:"trim,nil_on_empty"`
	URLs        []string
	Auth        *AuthConfig
	TLS         *TlsConfig
	Connection  *ConnectionConfig
	Reconnect   *ReconnectConfig
	Ping        *PingConfig
}

// SavedConnectionUpdate is the update DTO for a saved connection.
type SavedConnectionUpdate struct {
	Id          string  `normalize:"trim"`
	Name        *string `normalize:"trim"`
	Description *string `normalize:"trim,nil_on_empty"`
	URLs        []string
	Auth        *AuthConfig
	TLS         *TlsConfig
	Connection  *ConnectionConfig
	Reconnect   *ReconnectConfig
	Ping        *PingConfig
}

// TestConnectionRequest tests a connection without saving; non-empty
// ConnectionID persists the probe into SavedConnection.Meta.
type TestConnectionRequest struct {
	URLs           []string
	Auth           *AuthConfig
	TLS            *TlsConfig
	ConnectTimeout *time.Duration
	ConnectionID   string
}

// TestConnectionResult is the outcome of a connection probe.
type TestConnectionResult struct {
	Success           bool
	Error             string
	RTTMs             int64
	ServerVersion     string
	ServerName        string
	ServerID          string
	ClusterName       string
	MaxPayload        int64
	JetstreamEnabled  bool
	ConnectedURL      string
	DiscoveredServers []string
}
