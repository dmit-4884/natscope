// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsclient"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// Dialer establishes nats.go connections and implements natsclient.Dialer.
type Dialer struct {
	logger *slog.Logger
}

var _ natsclient.Dialer = (*Dialer)(nil)

// NewDialer creates a Dialer that opens NATS connections.
func NewDialer() *Dialer {
	return &Dialer{
		logger: slog.Default().With(slogx.Module("natsclient:natsgo")),
	}
}

// Dial establishes a connection from a SavedConnection; ctx is accepted for
// caller symmetry though nats.Connect ignores it.
func (d *Dialer) Dial(_ context.Context, saved *entities.SavedConnection) (natsclient.Client, error) {
	url := saved.URLsString()
	// Credentials are lifted into the vault upstream; mask anyway, this is the
	// last place a stray one would reach logs and Health.
	logURL := natsutil.MaskURLs(saved.URLs)

	natsOpts, err := buildOptions(saved)
	if err != nil {
		return nil, wrapErr(err)
	}

	// Pre-allocate the client so the ErrorHandler closure can capture async
	// errors on its permission watcher.
	client := &Client{
		id:             saved.Id,
		url:            logURL,
		permWatch:      NewPermissionWatcher(),
		logger:         d.logger,
		defaultTimeout: DefaultTimeout,
	}

	// Add logging handlers
	natsOpts = append(natsOpts,
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			d.logger.Warn("NATS disconnected",
				slogx.String("connection_id", saved.Id),
				slogx.String("url", logURL),
				slogx.Error(err))
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			client.permWatch.ResetReplies()
			d.logger.Info("NATS reconnected",
				slogx.String("connection_id", saved.Id),
				slogx.String("url", logURL))
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			d.logger.Warn("NATS connection closed",
				slogx.String("connection_id", saved.Id),
				slogx.String("url", logURL))
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			d.logger.Warn("NATS async error",
				slogx.String("url", logURL),
				slogx.Error(err))
			client.permWatch.HandleAsyncError(err)
		}),
	)

	// Connect without retries (fail fast); the client handles reconnection after
	// the initial connect succeeds.
	conn, err := nats.Connect(url, natsOpts...)
	if err != nil {
		d.logger.Error("failed to connect to NATS",
			slogx.String("url", logURL),
			slogx.Error(err))
		return nil, wrapErr(fmt.Errorf("%w: %w", errs.ErrNATSConnectionFailed, err))
	}

	domain, prefix := jetStreamTarget(saved.Connection)
	jsNew, err := newJetStream(conn, domain, prefix)
	if err != nil {
		conn.Close()
		return nil, wrapErr(coreerrs.WrapOperation(err, "get JetStream API"))
	}

	client.conn = conn
	client.permWatch.TrackReplies(client.inboxPrefix(), replySubscription(conn))
	client.api = apiPrefix(domain, prefix)
	client.jetStream = watchJetStream(jsNew, client.permWatch, client.api)

	d.logger.Info("connected to NATS",
		slogx.String("connection_id", saved.Id),
		slogx.String("url", logURL),
		slogx.String("server_id", conn.ConnectedServerId()),
		slogx.String("version", conn.ConnectedServerVersion()))

	return client, nil
}

// jetStreamTarget returns the JetStream domain and API prefix a connection is configured for.
func jetStreamTarget(cfg *entities.ConnectionConfig) (domain, prefix string) {
	if cfg == nil {
		return "", ""
	}
	return strings.TrimSpace(ptr.Unwrap(cfg.JetstreamDomain)), strings.TrimSpace(ptr.Unwrap(cfg.JetstreamAPIPrefix))
}

// newJetStream opens the JetStream API for a domain, an imported API prefix, or the default.
func newJetStream(conn *nats.Conn, domain, prefix string) (jetstream.JetStream, error) {
	switch {
	case domain != "":
		return jetstream.NewWithDomain(conn, domain)
	case prefix != "":
		return jetstream.NewWithAPIPrefix(conn, prefix)
	default:
		return jetstream.New(conn)
	}
}
