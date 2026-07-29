// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import (
	"time"

	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
)

// connectionDoc is the persistence model: the base plus the connection's
// non-secret fields; secrets are cleared on write and refilled from the vault on read.
type connectionDoc struct {
	bbstore.Base

	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	URLs        []string             `json:"urls,omitempty"`
	Auth        *authDoc             `json:"auth,omitempty"`
	TLS         *tlsDoc              `json:"tls,omitempty"`
	Connection  *connectionConfigDoc `json:"connection,omitempty"`
	Reconnect   *reconnectConfigDoc  `json:"reconnect,omitempty"`
	Ping        *pingConfigDoc       `json:"ping,omitempty"`
	Meta        *connectionMetaDoc   `json:"meta,omitempty"`
}

// Secret fields (Password/Token/NkeySeed/Credentials/JWT) are tagged input_only:
// secretsplit lifts them into the keychain vault and blanks them before the
// document is persisted (see mapper.go).
type authDoc struct {
	Method      int32   `json:"method,omitempty"`
	Username    *string `json:"username,omitempty"`
	Password    *string `json:"password,omitempty"     behavior:"input_only" secret:"auth.password"`
	Token       *string `json:"token,omitempty"        behavior:"input_only" secret:"auth.token"`
	NkeySeed    *string `json:"nkeySeed,omitempty"     behavior:"input_only" secret:"auth.nkeySeed"`
	Credentials *string `json:"credentials,omitempty"  behavior:"input_only" secret:"auth.credentials"`
	JWT         *string `json:"jwt,omitempty"          behavior:"input_only" secret:"auth.jwt"`
}

type tlsDoc struct {
	CaCert     *string `json:"caCert,omitempty"`
	ClientCert *string `json:"clientCert,omitempty"`
	ClientKey  *string `json:"clientKey,omitempty" behavior:"input_only" secret:"tls.clientKey"`
	SkipVerify bool    `json:"skipVerify,omitempty"`
	TlsFirst   bool    `json:"tlsFirst,omitempty"`
}

type connectionConfigDoc struct {
	ConnectTimeout          *time.Duration `json:"connectTimeout,omitempty"`
	ConnectionName          *string        `json:"connectionName,omitempty"`
	InboxPrefix             *string        `json:"inboxPrefix,omitempty"`
	NoEcho                  bool           `json:"noEcho,omitempty"`
	NoRandomize             bool           `json:"noRandomize,omitempty"`
	IgnoreDiscoveredServers bool           `json:"ignoreDiscoveredServers,omitempty"`
}

type reconnectConfigDoc struct {
	MaxReconnects        *int32         `json:"maxReconnects,omitempty"`
	ReconnectWait        *time.Duration `json:"reconnectWait,omitempty"`
	ReconnectBufSize     *int32         `json:"reconnectBufSize,omitempty"`
	RetryOnFailedConnect bool           `json:"retryOnFailedConnect,omitempty"`
}

type pingConfigDoc struct {
	PingInterval        *time.Duration `json:"pingInterval,omitempty"`
	MaxPingsOutstanding *int32         `json:"maxPingsOutstanding,omitempty"`
}

type connectionMetaDoc struct {
	LastTestedAt     time.Time `json:"lastTestedAt,omitempty"`
	LastSuccess      bool      `json:"lastSuccess,omitempty"`
	LastRTTMs        *int64    `json:"lastRttMs,omitempty"`
	ServerVersion    *string   `json:"serverVersion,omitempty"`
	ServerName       *string   `json:"serverName,omitempty"`
	ServerID         *string   `json:"serverId,omitempty"`
	ClusterName      *string   `json:"clusterName,omitempty"`
	MaxPayload       *int64    `json:"maxPayload,omitempty"`
	JetstreamEnabled *bool     `json:"jetstreamEnabled,omitempty"`
	ConnectedURL     *string   `json:"connectedUrl,omitempty"`
	LastError        *string   `json:"lastError,omitempty"`
}
