// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package run

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"path/filepath"
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
	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	stderrors "errors"
	corecontext "github.com/altessa-s/go-atlas/core/context"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
	slogfactory "github.com/altessa-s/go-atlas/observability/slog/factory"
	fxmodules "github.com/dmit-4884/natscope/internal/fx"
)

const (
	// startTimeout bounds dependency startup.
	startTimeout = 30 * time.Second
	// shutdownTimeout bounds graceful shutdown.
	shutdownTimeout = 30 * time.Second
	// sigChanBuffer holds the first signal plus a second one that forces exit.
	sigChanBuffer = 2
)

type App struct {
	configPath string

	config *appconfig.Config

	healthCoordinator *health.Coordinator

	// Service instance ID.
	sid *id.Service
	// sidFile backs sid; nil when a static Node.Id replaces it.
	sidFile *id.File

	dirsFallback string

	logLevel  string
	logFormat string

	sigChan chan os.Signal
}

// NewApp creates a new server app.
func NewApp() *App {
	return &App{
		healthCoordinator: health.New(),
	}
}

// Run resolves the config path (flag or CONFIG_FILE env) and starts the server.
func (srv *App) Run(cmd *cobra.Command, args []string) error {
	// Armed before startup so a signal during startup is queued, not lost.
	srv.sigChan = make(chan os.Signal, sigChanBuffer)
	signal.Notify(srv.sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if err := srv.ensureServiceDirs(); err != nil {
		return err
	}

	fileProvider, err := id.NewFile(path.Join(appinfo.LibDir(), strings.ToLower(appinfo.Name)+".sid"))
	if err != nil {
		return errors.WrapOperation(err, "create service id provider")
	}
	srv.sidFile = fileProvider
	if srv.sid, err = id.NewWithProvider(fileProvider); err != nil {
		return errors.WrapOperation(err, "create service id")
	}

	configFile, _ := cmd.Flags().GetString("config") //nolint:errcheck
	if configFile != "" {
		configFile = appinfo.ExpandPath(configFile)
	}
	if cf := appinfo.GetEnvVar("CONFIG_FILE"); cf != "" {
		configFile = cf
	}
	srv.logLevel, _ = cmd.Flags().GetString("log-level")   //nolint:errcheck
	srv.logFormat, _ = cmd.Flags().GetString("log-format") //nolint:errcheck

	if configFile != "" {
		if _, err := os.Stat(configFile); err != nil {
			return fmt.Errorf("configuration path does not exist: %s", configFile)
		}
		srv.configPath = configFile
	}

	return srv.run(context.Background())
}

func (srv *App) ensureServiceDirs() error {
	mkErr := appinfo.MakeAllDirs()
	if mkErr == nil {
		return nil
	}
	if pathErr, ok := stderrors.AsType[*fs.PathError](mkErr); ok {
		mkErr = pathErr
	}

	if appinfo.GetEnvVar("LIB_DIR") != "" || appinfo.GetEnvVar("VAR_DIR") != "" {
		return errors.Wrapf(mkErr, "failed to create service directories (check %s and %s)", envVarName("LIB_DIR"), envVarName("VAR_DIR"))
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return errors.Wrapf(mkErr, "failed to create service directories and no home directory to fall back to (set %s and %s)",
			envVarName("LIB_DIR"), envVarName("VAR_DIR"))
	}

	base := filepath.Join(home, "."+strings.ToLower(appinfo.Name))
	if err := os.Setenv(envVarName("VAR_DIR"), filepath.Join(base, "var")); err != nil {
		return errors.WrapOperation(err, "override service var dir")
	}
	if err := os.Setenv(envVarName("LIB_DIR"), filepath.Join(base, "lib")); err != nil {
		return errors.WrapOperation(err, "override service lib dir")
	}

	if err := appinfo.MakeAllDirs(); err != nil {
		return errors.WrapOperation(err, "create fallback service directories")
	}

	srv.dirsFallback = base

	return nil
}

func envVarName(key string) string {
	return strings.ToUpper(appconfig.EnvPrefix() + key)
}

// run performs the actual server startup.
func (srv *App) run(ctx context.Context) error {
	logconsole.Register()

	if err := srv.loadConfig(); err != nil {
		return errors.Wrapf(err, "failed to load config from '%s'", srv.configPath)
	}
	if err := applyLogFlags(srv.config.Logger, srv.logLevel, srv.logFormat); err != nil {
		return err
	}

	// Static service ID overrides the file provider when configured.
	if srv.config.Node != nil && srv.config.Node.Id != nil {
		srv.sid = id.MustNewStaticProvider(*srv.config.Node.Id)
		srv.sidFile = nil
	}

	logger, err := slogfactory.New(srv.config.Logger).
		WithPrefixKey(slogx.ModuleKey).
		WithServiceId(srv.sid.ID()).
		Build()
	if err != nil {
		return errors.WrapOperation(err, "create logger")
	}
	slog.SetDefault(logger)

	// PersistenceError is set only after ID() has attempted the lazy write.
	if srv.sidFile != nil {
		if perr := srv.sidFile.PersistenceError(); perr != nil {
			slog.Default().Warn("service id could not be persisted; a new id will be generated on next start",
				slogx.Error(perr))
		}
	}

	slog.Default().Info("starting",
		slog.Group("dirs",
			"lib", appinfo.LibDir(),
			"var", appinfo.VarDir(),
		),
	)

	if srv.dirsFallback != "" {
		slog.Default().Info("default service directories not writable, using home-local fallback", slog.String("base", srv.dirsFallback))
	}

	startCtx, startCancel := corecontext.WithMaxTimeout(ctx, startTimeout)
	defer startCancel()

	var vault secrets.Vault
	di := fx.New(
		fx.NopLogger,
		fx.RecoverFromPanics(),
		fx.Supply(srv.config),
		fx.Supply(slog.Default()),

		fxmodules.InfrastructureModule(),
		fxmodules.TransportsModule(),
		fxmodules.ServicesModule(),

		fx.Populate(&vault),
	)

	if di.Err() != nil {
		return rootCause(di.Err())
	}

	if err := di.Start(startCtx); err != nil {
		return rootCause(err)
	}

	// Apply the soft Go heap cap (default 512 MiB) before serving.
	configureMemoryLimit()

	if logconsole.IsTerminal(os.Stdout) {
		srv.printBanner(os.Stdout, vault)
	}

	slog.Default().Info("started")

	return srv.gracefulShutdown(ctx, di)
}

// gracefulShutdown waits for a signal or fx error, then stops; a second signal forces exit.
func (srv *App) gracefulShutdown(ctx context.Context, di *fx.App) error {
	select {
	case sig := <-srv.sigChan:
		slog.Default().Info("received shutdown signal", slog.String("signal", sig.String()))
	case <-di.Wait():
		slog.Default().Info("application stopped unexpectedly")
	}

	go func() {
		if sig, ok := <-srv.sigChan; ok {
			slog.Default().Warn("received a second shutdown signal, exiting immediately",
				slog.String("signal", sig.String()))
			os.Exit(1)
		}
	}()

	srv.healthCoordinator.BroadcastStatus(health.StatusNotServing)

	shutdownCtx, shutdownCancel := corecontext.WithMaxTimeout(ctx, shutdownTimeout)
	defer shutdownCancel()

	slog.Default().Debug("shutting down")

	if err := di.Stop(shutdownCtx); err != nil {
		slog.Default().Error("error during graceful shutdown", slogx.Error(err))
		return errors.Wrap(err, "graceful shutdown failed")
	}

	slog.Default().Debug("stopped")
	return nil
}

// rootCause returns the innermost error in err's chain.
func rootCause(err error) error {
	for {
		next := stderrors.Unwrap(err)
		if next == nil {
			return err
		}
		err = next
	}
}

func (srv *App) loadConfig() error {
	var err error
	srv.config, err = appconfig.Load(srv.configPath, appconfig.LoggerDefaults(logconsole.IsTerminal(os.Stderr)))
	if err != nil {
		return err
	}

	return nil
}

func (srv *App) printBanner(w io.Writer, vault secrets.Vault) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	renderBanner(w, banner{
		Version: appinfo.Version,
		Address: srv.config.GRPCWebAddress,
		DataDir: srv.config.ResolveDataDir(),
		Secrets: secretsLabel(vault),
		Home:    home,
	})
}

// defaultMemoryLimit is the soft GC cap used when GOMEMLIMIT is unset. 512 MiB
// bounds arena growth when a multi-MB message decode spikes the heap.
const defaultMemoryLimit int64 = 512 * 1024 * 1024

// minMemoryLimit is the lowest accepted NATSCOPE_MEMORY_LIMIT_BYTES value.
const minMemoryLimit int64 = 16 * 1024 * 1024

// configureMemoryLimit applies the soft heap cap from GOMEMLIMIT, NATSCOPE_MEMORY_LIMIT_BYTES
// or defaultMemoryLimit, logging a warning for an invalid NATSCOPE_MEMORY_LIMIT_BYTES.
func configureMemoryLimit() {
	if os.Getenv("GOMEMLIMIT") != "" {
		// Runtime already parsed it.
		slog.Default().Debug("memory limit honored from GOMEMLIMIT", slog.Int64("bytes", debug.SetMemoryLimit(-1)))
		return
	}
	limit := defaultMemoryLimit
	if v := os.Getenv("NATSCOPE_MEMORY_LIMIT_BYTES"); v != "" {
		switch parsed, err := parseMemoryLimit(v); {
		case err != nil:
			slog.Default().Warn("ignoring invalid NATSCOPE_MEMORY_LIMIT_BYTES, using the default",
				slog.String("value", v), slogx.Error(err), slog.Int64("default_bytes", defaultMemoryLimit))
		case parsed < minMemoryLimit:
			slog.Default().Warn("NATSCOPE_MEMORY_LIMIT_BYTES is below the minimum, using the default",
				slog.String("value", v), slog.Int64("min_bytes", minMemoryLimit), slog.Int64("default_bytes", defaultMemoryLimit))
		default:
			limit = parsed
		}
	}
	debug.SetMemoryLimit(limit)
	slog.Default().Debug("memory limit applied", slog.Int64("bytes", limit))
}

// memoryLimitUnits lists the GOMEMLIMIT suffixes, longest first so "B" can't shadow "MiB".
var memoryLimitUnits = []struct {
	suffix string
	mult   int64
}{
	{"GiB", 1 << 30},
	{"MiB", 1 << 20},
	{"KiB", 1 << 10},
	{"B", 1},
}

// parseMemoryLimit parses a byte count with an optional B/KiB/MiB/GiB suffix.
func parseMemoryLimit(v string) (int64, error) {
	v = strings.TrimSpace(v)
	for _, u := range memoryLimitUnits {
		rest, ok := strings.CutSuffix(v, u.suffix)
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid memory limit %q", v)
		}
		return n * u.mult, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid memory limit %q", v)
	}
	return n, nil
}
