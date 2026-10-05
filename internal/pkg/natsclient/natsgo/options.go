// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"

	"github.com/altessa-s/go-atlas/security/tlsutils"

	"github.com/dmit-4884/natscope/internal/entities"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// Defaults for connections that don't pin their own values: short connect
// timeout, infinite retries, slow reconnect cadence (long-lived UI viewer).
const (
	defaultConnectTimeout = 10 * time.Second
	defaultReconnectWait  = 2 * time.Second
	infiniteReconnects    = -1
)

// buildOptions builds nats.Option from a SavedConnection. Pure translation,
// no I/O or service state.
func buildOptions(saved *entities.SavedConnection) ([]nats.Option, error) {
	var opts []nats.Option

	authOpts, err := buildAuthOptions(saved.Auth)
	if err != nil {
		return nil, err
	}
	opts = append(opts, authOpts...)
	tlsOpts, err := buildTLSOptions(saved.TLS)
	if err != nil {
		return nil, err
	}
	opts = append(opts, tlsOpts...)
	opts = append(opts, buildConnectionOptions(saved.Connection)...)
	opts = append(opts, buildReconnectOptions(saved.Reconnect)...)
	opts = append(opts, buildPingOptions(saved.Ping)...)

	if saved.Connection == nil || saved.Connection.ConnectTimeout == nil {
		opts = append(opts, nats.Timeout(defaultConnectTimeout))
	}
	if saved.Reconnect == nil || saved.Reconnect.MaxReconnects == nil {
		opts = append(opts, nats.MaxReconnects(infiniteReconnects))
	}
	if saved.Reconnect == nil || saved.Reconnect.ReconnectWait == nil {
		opts = append(opts, nats.ReconnectWait(defaultReconnectWait))
	}

	return opts, nil
}

// buildTestOptions builds nats.Option from an ad-hoc test request (only
// Auth/TLS; connect-timeout default always applies).
func buildTestOptions(in *entities.TestConnectionRequest) ([]nats.Option, entities.ConnectionCheckStep, error) {
	var opts []nats.Option //nolint:prealloc // fan-in from variadic helpers; final size unknown

	authOpts, err := buildAuthOptions(in.Auth)
	if err != nil {
		return nil, entities.CheckStepAuth, err
	}
	opts = append(opts, authOpts...)
	tlsOpts, err := buildTLSOptions(in.TLS)
	if err != nil {
		return nil, entities.CheckStepTLS, err
	}
	opts = append(opts, tlsOpts...)
	if in.Connection != nil && in.Connection.InboxPrefix != nil && *in.Connection.InboxPrefix != "" {
		opts = append(opts, nats.CustomInboxPrefix(*in.Connection.InboxPrefix))
	}

	timeout := defaultConnectTimeout
	if in.ConnectTimeout != nil {
		timeout = *in.ConnectTimeout
	}
	opts = append(opts, nats.Timeout(timeout))

	return opts, entities.CheckStepUnspecified, nil
}

// buildAuthOptions translates the auth config into nats.Options; an unparsable NKey seed is an error.
func buildAuthOptions(auth *entities.AuthConfig) ([]nats.Option, error) {
	if auth == nil {
		return nil, nil
	}

	var opts []nats.Option

	switch auth.Method {
	case entities.AuthMethodToken:
		if auth.Token != nil && *auth.Token != "" {
			opts = append(opts, nats.Token(*auth.Token))
		}
	case entities.AuthMethodUserPass:
		if auth.Username != nil && auth.Password != nil {
			opts = append(opts, nats.UserInfo(*auth.Username, *auth.Password))
		}
	case entities.AuthMethodNKey:
		if auth.NkeySeed != nil && *auth.NkeySeed != "" {
			kp, err := nkeys.FromSeed([]byte(*auth.NkeySeed))
			if err != nil {
				return nil, coreerrs.Wrap(err, "invalid nkey seed")
			}
			pub, err := kp.PublicKey()
			if err != nil {
				return nil, coreerrs.Wrap(err, "invalid nkey seed")
			}
			opts = append(opts, nats.Nkey(pub, kp.Sign))
		}
	case entities.AuthMethodCredentials:
		if auth.Credentials != nil && *auth.Credentials != "" {
			opts = append(opts, nats.UserCredentialBytes([]byte(*auth.Credentials)))
		} else if auth.JWT != nil && auth.NkeySeed != nil {
			opts = append(opts, nats.UserCredentialBytes([]byte(*auth.JWT), []byte(*auth.NkeySeed)))
		}
	case entities.AuthMethodNone:
		// No auth; explicit case for exhaustiveness.
	default:
		return nil, fmt.Errorf("unsupported auth method: %d", auth.Method)
	}

	return opts, nil
}

func buildTLSOptions(tlsCfg *entities.TlsConfig) ([]nats.Option, error) {
	tlsConfig, hasTLS, err := buildTLSConfig(tlsCfg)
	if err != nil {
		return nil, err
	}

	var opts []nats.Option
	if hasTLS {
		opts = append(opts, nats.Secure(tlsConfig))
	}
	if tlsCfg != nil && tlsCfg.TlsFirst {
		opts = append(opts, nats.TLSHandshakeFirst())
	}
	return opts, nil
}

// buildTLSConfig builds the client TLS config and reports whether the connection sets any TLS material.
func buildTLSConfig(tlsCfg *entities.TlsConfig) (*tls.Config, bool, error) {
	tlsConfig := &tls.Config{} //nolint:gosec // user-controlled skip_verify is intentional
	if tlsCfg == nil {
		return tlsConfig, false, nil
	}
	hasTLS := false

	if tlsCfg.CaCert != nil && *tlsCfg.CaCert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(*tlsCfg.CaCert)) {
			return nil, false, errors.New("invalid TLS CA certificate: no PEM certificates found")
		}
		tlsConfig.RootCAs = pool
		hasTLS = true
	}

	if tlsCfg.ClientCert != nil && tlsCfg.ClientKey != nil &&
		*tlsCfg.ClientCert != "" && *tlsCfg.ClientKey != "" {
		cert, err := tlsutils.LoadFromBytes([]byte(*tlsCfg.ClientCert), []byte(*tlsCfg.ClientKey), "")
		if err != nil {
			return nil, false, coreerrs.Wrap(err, "invalid TLS client certificate/key")
		}
		tlsConfig.Certificates = []tls.Certificate{*cert}
		hasTLS = true
	}

	if tlsCfg.SkipVerify {
		tlsConfig.InsecureSkipVerify = true
		hasTLS = true
	}

	return tlsConfig, hasTLS, nil
}

func buildConnectionOptions(connCfg *entities.ConnectionConfig) []nats.Option {
	if connCfg == nil {
		return nil
	}

	var opts []nats.Option

	if connCfg.ConnectTimeout != nil {
		opts = append(opts, nats.Timeout(*connCfg.ConnectTimeout))
	}
	if connCfg.ConnectionName != nil && *connCfg.ConnectionName != "" {
		opts = append(opts, nats.Name(*connCfg.ConnectionName))
	}
	if connCfg.InboxPrefix != nil && *connCfg.InboxPrefix != "" {
		opts = append(opts, nats.CustomInboxPrefix(*connCfg.InboxPrefix))
	}
	if connCfg.NoEcho {
		opts = append(opts, nats.NoEcho())
	}
	if connCfg.NoRandomize {
		opts = append(opts, nats.DontRandomize())
	}
	if connCfg.IgnoreDiscoveredServers {
		opts = append(opts, nats.IgnoreDiscoveredServers())
	}

	return opts
}

func buildReconnectOptions(reconnCfg *entities.ReconnectConfig) []nats.Option {
	if reconnCfg == nil {
		return nil
	}

	var opts []nats.Option

	if reconnCfg.MaxReconnects != nil {
		opts = append(opts, nats.MaxReconnects(int(*reconnCfg.MaxReconnects)))
	}
	if reconnCfg.ReconnectWait != nil {
		opts = append(opts, nats.ReconnectWait(*reconnCfg.ReconnectWait))
	}
	if reconnCfg.ReconnectBufSize != nil {
		opts = append(opts, nats.ReconnectBufSize(int(*reconnCfg.ReconnectBufSize)))
	}
	if reconnCfg.RetryOnFailedConnect {
		opts = append(opts, nats.RetryOnFailedConnect(true))
	}

	return opts
}

func buildPingOptions(pingCfg *entities.PingConfig) []nats.Option {
	if pingCfg == nil {
		return nil
	}

	var opts []nats.Option

	if pingCfg.PingInterval != nil {
		opts = append(opts, nats.PingInterval(*pingCfg.PingInterval))
	}
	if pingCfg.MaxPingsOutstanding != nil {
		opts = append(opts, nats.MaxPingsOutstanding(int(*pingCfg.MaxPingsOutstanding)))
	}

	return opts
}
