// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package fx

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"connectrpc.com/connect"
	"go.uber.org/fx"

	"github.com/altessa-s/go-atlas/core/errors"
	"github.com/altessa-s/go-atlas/core/runtime/panics"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/codec"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/connections"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/history"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/live"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/management"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/mappings"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/messages"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/publish"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/registry"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/selections"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/settings"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/sources"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/stats"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/streams"
	"github.com/dmit-4884/natscope/internal/transports/grpc/handlers/templates"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	httpserver "github.com/altessa-s/go-atlas/transport/http/server"
	httpfactory "github.com/altessa-s/go-atlas/transport/http/server/factory"
	livesvc "github.com/dmit-4884/natscope/internal/services/live"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	protoService "github.com/dmit-4884/natscope/internal/services/proto/proto"
	grpctransport "github.com/dmit-4884/natscope/internal/transports/grpc"
	workspaceHandler "github.com/dmit-4884/natscope/internal/transports/grpc/handlers/workspace"
	grpchelpers "github.com/dmit-4884/natscope/internal/transports/grpc/helpers"
)

// maxRequestBytes caps a single decoded request message to protect the process
// from memory exhaustion; sized for workspace imports with a wide margin.
const maxRequestBytes = 32 << 20 // 32 MiB

// TransportsModule provides the unified Connect-RPC transport plus the
// optional HTTP server for internal endpoints (health, metrics, pprof).
func TransportsModule() fx.Option {
	return fx.Module("transports",
		// Connect transport (single port, all three RPC protocols + SPA).
		fx.Provide(newConnectTransport),

		// HTTP server for internal endpoints (health, metrics, pprof).
		fx.Provide(newHTTPServer),
		fx.Invoke(func(_ *httpserver.Server) {}),

		// Connect handlers — all 16 collected via the connect-handlers group.
		fx.Provide(AsConnectHandler(connections.New)),
		fx.Provide(AsConnectHandler(streams.New)),
		fx.Provide(AsConnectHandler(messages.New)),
		fx.Provide(AsConnectHandler(publish.New)),
		fx.Provide(AsConnectHandler(management.New)),
		fx.Provide(AsConnectHandler(stats.New)),
		fx.Provide(AsConnectHandler(live.New)),
		fx.Provide(AsConnectHandler(registry.New)),
		fx.Provide(AsConnectHandler(codec.New)),
		fx.Provide(AsConnectHandler(sources.New)),
		fx.Provide(AsConnectHandler(selections.New)),
		fx.Provide(AsConnectHandler(mappings.New)),
		fx.Provide(AsConnectHandler(history.New)),
		fx.Provide(AsConnectHandler(settings.New)),
		fx.Provide(AsConnectHandler(templates.New)),
		fx.Provide(AsConnectHandler(workspaceHandler.New)),

		// Wire grouped handlers into the transport before Start runs.
		fx.Invoke(fx.Annotate(func(handlers []grpctransport.Handler, t *grpctransport.Transport) {
			if t != nil {
				t.RegisterHandlers(handlers)
			}
		}, fx.ParamTags(`group:"connect-handlers"`))),

		// Proto reload flips each live session's dirty bit so it re-initializes
		// its decoder before the next message; also restores local-source watchers.
		fx.Invoke(func(lc fx.Lifecycle, protoSvc protosvc.Codec, liveSvc livesvc.Service) error {
			ps, ok := protoSvc.(*protoService.Service)
			if !ok {
				// Fail loudly: a silent skip would leave live sessions without
				// proto-reload broadcasts and never restore local-source watchers.
				return fmt.Errorf("proto reload wiring: Codec is %T, want *proto.Service", protoSvc)
			}
			ps.SetOnReloadCallback(func(_ int) {
				liveSvc.BroadcastProtoReload()
			})
			lc.Append(fx.StartHook(func(ctx context.Context) error {
				ps.RestoreWatchers(ctx)
				return nil
			}))
			return nil
		}),
	)
}

// AsConnectHandler annotates a handler constructor so its result lands in the
// connect-handlers group consumed by the transport.
func AsConnectHandler(f any) any {
	return fx.Annotate(f, fx.As(new(grpctransport.Handler)), fx.ResultTags(`group:"connect-handlers"`))
}

// newConnectTransport builds the unified Connect transport and wires
// lifecycle hooks; nil return means GRPCWebAddress is empty (disabled).
func newConnectTransport(cfg *appconfig.Config, lc fx.Lifecycle) (*grpctransport.Transport, error) {
	lgr := slog.Default().With(slogx.Module("transport:connect"))

	if cfg.GRPCWebAddress == "" {
		lgr.Warn("connect transport is not configured (empty GRPCWebAddress), skipping")
		return nil, nil //nolint:nilnil // intentional: transport is optional
	}

	// Fail-closed: a non-loopback bind without auth aborts startup unless the
	// risk is explicitly accepted (allowInsecure).
	if !cfg.BindsLoopback() {
		if !cfg.AllowRemote {
			return nil, fmt.Errorf(
				"refusing to bind %s: the API has no authentication "+
					"and returns connection secrets. Set allowRemote: true (env ALLOW_REMOTE=true) "+
					"to expose it beyond loopback — ideally together with webAuth credentials",
				cfg.GRPCWebAddress)
		}
		if !cfg.WebAuth.Enabled() && !cfg.AllowInsecure {
			return nil, fmt.Errorf(
				"refusing to bind %s without authentication: the API returns NATS "+
					"connection secrets, git tokens and TLS keys to any caller. Configure "+
					"webAuth (WEB_AUTH__USERNAME / WEB_AUTH__PASSWORD), or set "+
					"allowInsecure: true (env ALLOW_INSECURE=true) to accept the risk explicitly. "+
					"The listener speaks cleartext h2c — terminate TLS in a reverse proxy "+
					"in front of it for any remote access",
				cfg.GRPCWebAddress)
		}
		if !cfg.WebAuth.Enabled() {
			lgr.Warn("listener is exposed beyond loopback WITHOUT authentication "+
				"(allowInsecure=true) — anyone who can reach it can read your NATS "+
				"connection secrets, git tokens and TLS keys",
				slog.String("address", cfg.GRPCWebAddress))
		}
	}

	frontendFS, fsErr := fs.Sub(grpctransport.FrontendFS, "dist")
	if fsErr != nil {
		// Embedded build did not include the dist tree (dev or partial build).
		// Transport still serves RPC; the SPA fallback degrades to 404s.
		lgr.Warn("frontend dist not embedded; SPA fallback will 404",
			slogx.Error(fsErr))
		frontendFS = nil
	}

	connectOpts := []connect.HandlerOption{
		// vtprotobuf-aware codec: skips reflection for all codegen'd types
		// except google.rpc.Status (Connect's internal error type).
		connect.WithCodec(grpchelpers.VTCodec{}),
		// Bound decoded request size; workspace imports are the largest
		// legitimate payloads, everything else is far below this cap.
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(
			// Validation runs once globally for every RPC; per-handler error
			// conversion is registered separately by each Handler.HTTPHandler.
			grpchelpers.NewValidationConnectInterceptor(),
		),
	}

	t, err := grpctransport.New(cfg.GRPCWebAddress, frontendFS, lgr, cfg.BindsLoopback(), connectOpts...)
	if err != nil {
		return nil, errors.WrapOperation(err, "create connect transport")
	}

	if cfg.WebAuth.Enabled() {
		t.SetBasicAuth(cfg.WebAuth.Username, cfg.WebAuth.Password)
		lgr.Info("HTTP basic auth enabled for the listener")
	}

	lc.Append(fx.StartHook(func(ctx context.Context) error {
		// Bind synchronously so a port-in-use failure aborts startup; async
		// Serve would let the listen error surface after StartHook returns nil.
		if err := t.Listen(ctx); err != nil {
			return err
		}
		// Serve omits ctx on purpose: StartHook's ctx has a startup deadline
		// that would close the listener early. StopHook owns shutdown via GracefulStop.
		go func() {
			defer panics.Handle(ctx)
			// Serve omits ctx on purpose (see above); the guard's ctx is only for
			// log correlation, not propagated into Serve.
			if err := t.Serve(); err != nil { //nolint:contextcheck // intentional decoupling
				lgr.Error("connect transport stopped", slogx.Error(err))
			}
		}()
		return nil
	}))

	lc.Append(fx.StopHook(func(ctx context.Context) error {
		return t.GracefulStop(ctx)
	}))

	return t, nil
}

// newHTTPServer creates and initializes an HTTP server for internal endpoints
// (health checks, metrics, pprof). Returns nil if HTTP is not configured.
func newHTTPServer(cfg *appconfig.Config, lc fx.Lifecycle) (*httpserver.Server, error) {
	if !cfg.IsHttpConfigured() {
		return nil, nil //nolint:nilnil
	}

	// Fail-closed: the internal HTTP server exposes unauthenticated health,
	// metrics and pprof. Refuse a non-loopback bind unless explicitly accepted.
	if !cfg.HTTPBindsLoopback() && !cfg.AllowInsecure {
		return nil, fmt.Errorf(
			"refusing to bind internal HTTP server to %s: it serves unauthenticated "+
				"health/metrics/pprof. Bind a loopback address, or set allowInsecure: true "+
				"(env ALLOW_INSECURE=true) to accept the risk explicitly",
			cfg.Http.ListenAddress)
	}

	srv, err := httpfactory.New(cfg.Http).
		UseLogger(slog.Default().With(slogx.Module("transport:http"))).
		WithMiddlewares().
		Build()
	if err != nil {
		return nil, errors.WrapOperation(err, "create HTTP server")
	}

	lc.Append(fx.StartHook(srv.Start))
	lc.Append(fx.StopHook(srv.Shutdown))

	return srv, nil
}
