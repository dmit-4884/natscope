// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/altessa-s/go-atlas/core/errors"
	"github.com/altessa-s/go-atlas/core/runtime/appinfo"
	"github.com/altessa-s/go-atlas/observability/health"
	"github.com/altessa-s/go-atlas/service/id"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	slogfactory "github.com/altessa-s/go-atlas/observability/slog/factory"
	fxmodules "github.com/dmit-4884/natscope/internal/fx"
)

const (
	// startTimeout bounds dependency startup.
	startTimeout = 30 * time.Second
	// shutdownTimeout bounds graceful shutdown.
	shutdownTimeout = 30 * time.Second
)

type App struct {
	configPath string

	config *appconfig.Config

	healthCoordinator *health.Coordinator

	// Service instance ID.
	sid *id.Service
}

// NewApp creates a new server app.
func NewApp() *App {
	return &App{
		sid: id.MustNewWithFileProvider(
			path.Join(appinfo.LibDir(), strings.ToLower(appinfo.Name)+".sid"),
		),
		healthCoordinator: health.New(),
	}
}

// Run resolves the config path (flag or CONFIG_FILE env) and starts the server.
func (srv *App) Run(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config") //nolint:errcheck
	if cf := appinfo.GetEnvVar("CONFIG_FILE"); cf != "" {
		configFile = cf
	}

	if configFile != "" {
		if _, err := os.Stat(configFile); err != nil {
			return fmt.Errorf("configuration path does not exist: %s", configFile)
		}
		srv.configPath = configFile
	}

	return srv.run(context.Background())
}

// run performs the actual server startup.
func (srv *App) run(ctx context.Context) error {
	if err := srv.loadConfig(); err != nil {
		return errors.Wrapf(err, "failed to load config from '%s'", srv.configPath)
	}

	// Static service ID overrides the file provider when configured.
	if srv.config.Node != nil && srv.config.Node.Id != nil {
		srv.sid = id.MustNewStaticProvider(*srv.config.Node.Id)
	}

	logger, err := slogfactory.New(srv.config.Logger).
		WithServiceId(srv.sid.ID()).
		Build()
	if err != nil {
		return errors.WrapOperation(err, "create logger")
	}
	slog.SetDefault(logger)

	slog.Default().Info("starting",
		slog.String("service", appinfo.Name),
		slog.String("version", appinfo.Version),
		slog.String("sid", srv.sid.ID()),
		slog.Group("dirs",
			"bin", appinfo.BinDir(),
			"etc", appinfo.EtcDir(),
			"lib", appinfo.LibDir(),
			"var", appinfo.VarDir(),
		),
	)

	if err := appinfo.MakeAllDirs(); err != nil {
		return errors.WrapOperation(err, "create service directories")
	}

	startCtx, startCancel := context.WithTimeout(ctx, startTimeout)
	defer startCancel()

	di := fx.New(
		fx.NopLogger,
		fx.RecoverFromPanics(),
		fx.Supply(srv.config),
		fx.Supply(slog.Default()),

		fxmodules.InfrastructureModule(),
		fxmodules.TransportsModule(),
		fxmodules.ServicesModule(),
	)

	if di.Err() != nil {
		return errors.WrapOperation(di.Err(), "initialize dependencies")
	}

	if err := di.Start(startCtx); err != nil {
		return errors.WrapOperation(err, "start dependencies")
	}

	// Apply the soft Go heap cap (default 512 MiB) before serving.
	configureMemoryLimit()

	slog.Default().Info("started")

	return srv.gracefulShutdown(ctx, di)
}

// gracefulShutdown waits for a signal (or fx error) then stops cleanly.
func (srv *App) gracefulShutdown(ctx context.Context, di *fx.App) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		slog.Default().Info("received shutdown signal", slog.String("signal", sig.String()))
	case <-di.Wait():
		slog.Default().Info("application stopped unexpectedly")
	}

	srv.healthCoordinator.BroadcastStatus(health.StatusNotServing)

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, shutdownTimeout)
	defer shutdownCancel()

	slog.Default().Info("shutting down")

	if err := di.Stop(shutdownCtx); err != nil {
		slog.Default().Error("error during graceful shutdown", slogx.Error(err))
		return errors.Wrap(err, "graceful shutdown failed")
	}

	slog.Default().Info("stopped")
	return nil
}

func (srv *App) loadConfig() error {
	var err error
	srv.config, err = appconfig.Load(srv.configPath)
	if err != nil {
		return err
	}

	return nil
}

// defaultMemoryLimit is the soft GC cap used when GOMEMLIMIT is unset. 512 MiB
// bounds arena growth when a multi-MB message decode spikes the heap.
const defaultMemoryLimit int64 = 512 * 1024 * 1024

// configureMemoryLimit applies the soft heap cap, honoring GOMEMLIMIT or
// NATSCOPE_MEMORY_LIMIT_BYTES when set, otherwise defaultMemoryLimit.
func configureMemoryLimit() {
	if os.Getenv("GOMEMLIMIT") != "" {
		// Runtime already parsed it.
		slog.Default().Info("memory limit honored from GOMEMLIMIT", slog.Int64("bytes", debug.SetMemoryLimit(-1)))
		return
	}
	limit := defaultMemoryLimit
	if v := os.Getenv("NATSCOPE_MEMORY_LIMIT_BYTES"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}
	debug.SetMemoryLimit(limit)
	slog.Default().Info("memory limit applied", slog.Int64("bytes", limit))
}
