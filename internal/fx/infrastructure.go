// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package fx provides Uber FX dependency injection modules.
package fx

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"go.uber.org/fx"

	"github.com/altessa-s/go-atlas/core/errors"
	"github.com/altessa-s/go-atlas/observability/metrics"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/natsclient"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	metricsfactory "github.com/altessa-s/go-atlas/observability/metrics/factory"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
	natsgo "github.com/dmit-4884/natscope/internal/pkg/natsclient/natsgo"
	fwsvc "github.com/dmit-4884/natscope/internal/services/filewatcher"
	fwimpl "github.com/dmit-4884/natscope/internal/services/filewatcher/filewatcher"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	natsimpl "github.com/dmit-4884/natscope/internal/services/nats/nats"
	connectionsStorage "github.com/dmit-4884/natscope/internal/storages/connections"
)

// keychainServiceName namespaces OS keychain secrets (NATS auth, TLS keys,
// git tokens). Keep stable — changing it orphans every stored secret.
const keychainServiceName = "natscope"

// dataDirPerm is the permission for the auto-created data directory (user-only).
const dataDirPerm = 0o700

// InfrastructureModule provides core infrastructure components.
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		fx.Provide(newMetricsCollector),
		fx.Provide(newSecretsVault),
		fx.Provide(newBboltDB),
		fx.Provide(fx.Annotate(natsgo.NewDialer, fx.As(new(natsclient.Dialer)))),
		fx.Provide(fx.Annotate(
			newNATSService,
			fx.As(new(natssvc.ConnectionManager)),
			fx.As(new(natssvc.StreamReader)),
			fx.As(new(natssvc.StreamManager)),
			fx.As(new(natssvc.ConsumerManager)),
			fx.As(new(natssvc.Publisher)),
			fx.As(new(natssvc.Subscriber)),
			fx.As(new(natssvc.StatsReader)),
			fx.As(new(natssvc.KVStore)),
			fx.As(new(natssvc.ObjectStore)),
		)),
		fx.Provide(fx.Annotate(newFileWatcher, fx.As(new(fwsvc.Service)))),
	)
}

// newSecretsVault picks the vault that keeps every secret out of the
// database: the OS keychain when usable, otherwise the encrypted file vault
// (headless Linux, containers).
func newSecretsVault(cfg *appconfig.Config) (secrets.Vault, error) {
	lgr := slog.Default().With(slogx.Module("infra:secrets"))

	switch cfg.SecretsBackend() {
	case appconfig.SecretsBackendKeyring:
		return secrets.NewKeyring(keychainServiceName), nil
	case appconfig.SecretsBackendFile:
		lgr.Info("using encrypted file vault for secrets",
			slog.String("dir", cfg.ResolveDataDir()))
		return secrets.NewFile(cfg.ResolveDataDir())
	default: // auto
		kr := secrets.NewKeyring(keychainServiceName)
		if kr.Available() {
			return kr, nil
		}
		lgr.Warn("OS keychain unavailable — falling back to the encrypted file vault; "+
			"keep secrets.vault and vault.key private (both are user-only)",
			slog.String("dir", cfg.ResolveDataDir()))
		return secrets.NewFile(cfg.ResolveDataDir())
	}
}

// newBboltDB opens the shared bbolt database and registers Close on shutdown.
// Secrets live in the OS keychain, not here — each storage creates its bucket.
func newBboltDB(cfg *appconfig.Config, lc fx.Lifecycle) (*bbstore.DB, error) {
	path := cfg.ResolveBboltPath()

	// bbolt opens but does not create the data directory; ensure it exists.
	if err := os.MkdirAll(filepath.Dir(path), dataDirPerm); err != nil {
		return nil, errors.WrapOperation(err, "create data directory")
	}

	// NewDB already returns actionable, path-qualified errors (lock/corruption).
	db, err := bbstore.NewDB(path)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.StopHook(db.Close))
	return db, nil
}

// newMetricsCollector creates a metrics collector from configuration.
// Returns a no-op collector when metrics are not configured.
func newMetricsCollector(cfg *appconfig.Config, lc fx.Lifecycle) (metrics.Collector, error) {
	collector, err := metricsfactory.New(cfg.Metrics).Build()
	if err != nil {
		return nil, errors.WrapOperation(err, "create metrics collector")
	}

	lc.Append(fx.StopHook(collector.Shutdown))

	return collector, nil
}

// newNATSService creates a new NATS service with lazy connection support.
func newNATSService(lc fx.Lifecycle, dialer natsclient.Dialer, connStore connectionsStorage.Storage) *natsimpl.Service {
	svc := natsimpl.New(dialer, connStore)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			svc.Close()
			return nil
		},
	})

	return svc
}

// newFileWatcher creates and starts the file watcher service.
func newFileWatcher(lc fx.Lifecycle) *fwimpl.Service {
	svc := fwimpl.New()

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return svc.Start()
		},
		OnStop: func(context.Context) error {
			svc.Stop()
			return nil
		},
	})

	return svc
}
