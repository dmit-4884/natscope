// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package e2e contains end-to-end integration tests that boot the real natscope
// application (real services + the unified Connect transport) against an
// embedded NATS JetStream server and drive every major flow through the
// generated Connect Go clients.
//
// The suite is intentionally hermetic:
//   - The OS keychain is replaced with go-keyring's in-memory backend so
//     cryptobox never prompts (and never touches the user's real keychain).
//   - NATS runs embedded with JetStream on a random port, storing state under
//     a t.TempDir() that is torn down with the test.
//   - All JSON storages point at a t.TempDir(), never ~/.natscope/data.
package e2e

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
	"go.uber.org/fx"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"

	fxmodules "github.com/dmit-4884/natscope/internal/fx"
	historyconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/history/v1/history/grpc_historyconnect"
	mappingsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings/grpc_mappingsconnect"
	connectionsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections/grpc_nats_connectionsconnect"
	liveconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live/grpc_nats_liveconnect"
	managementconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management/grpc_nats_managementconnect"
	messagesconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages/grpc_nats_messagesconnect"
	publishconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish/grpc_nats_publishconnect"
	statsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats/grpc_nats_statsconnect"
	streamsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams/grpc_nats_streamsconnect"
	codecconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec/grpc_proto_codecconnect"
	registryconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry/grpc_proto_registryconnect"
	selectionsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/selections/grpc_proto_selectionsconnect"
	sourcesconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources/grpc_proto_sourcesconnect"
	settingsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings/grpc_settingsconnect"
	templatesconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates/grpc_templatesconnect"
	workspaceconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/workspace/v1/workspace/grpc_workspaceconnect"
)

// e2eEnv bundles the running embedded NATS server, the booted natscope app, and
// the generated Connect clients pointed at the in-process HTTP listener.
type e2eEnv struct {
	natsURL string
	baseURL string

	connections connectionsconnect.ConnectionsServiceClient
	streams     streamsconnect.StreamsServiceClient
	messages    messagesconnect.MessagesServiceClient
	publish     publishconnect.PublishServiceClient
	management  managementconnect.ManagementServiceClient
	stats       statsconnect.StatsServiceClient
	live        liveconnect.LiveServiceClient
	mappings    mappingsconnect.MappingsServiceClient
	templates   templatesconnect.TemplatesServiceClient
	settings    settingsconnect.SettingsServiceClient
	history     historyconnect.HistoryServiceClient
	registry    registryconnect.RegistryServiceClient
	codec       codecconnect.CodecServiceClient
	sources     sourcesconnect.SourcesServiceClient
	selections  selectionsconnect.SelectionsServiceClient
	workspace   workspaceconnect.WorkspaceServiceClient
}

// setupE2E mocks the keychain, starts embedded NATS + JetStream, and boots
// the real natscope app on a free localhost port, returning the wired Connect clients.
func setupE2E(t *testing.T) *e2eEnv {
	t.Helper()

	// 1. In-memory keychain so cryptobox never prompts / touches the real OS keychain.
	keyring.MockInit()

	// 2. Embedded NATS server with JetStream on a random port.
	natsSrv := startEmbeddedNATS(t)
	natsURL := natsSrv.ClientURL()

	// 3. Boot the real app (same fx modules as the production entrypoint) on a
	//    free localhost port. The port pick races with other processes between
	//    release and rebind, so retry with a fresh port + data dir when Start
	//    fails to bind.
	const bootAttempts = 5
	var (
		app  *fx.App
		addr string
	)
	for attempt := 1; ; attempt++ {
		addr = freeLocalAddr(t)
		cfg := &appconfig.Config{
			GRPCWebAddress: addr,
			Storage: &appconfig.StorageConfig{
				Local: &appconfig.LocalStorageConfig{DataDir: t.TempDir()},
			},
		}

		candidate := fx.New(
			fx.NopLogger,
			fx.RecoverFromPanics(),
			fx.Supply(cfg),
			fx.Supply(slog.Default()),

			fxmodules.InfrastructureModule(),
			fxmodules.TransportsModule(),
			fxmodules.ServicesModule(),
		)
		require.NoError(t, candidate.Err(), "fx wiring must be valid")

		startCtx, startCancel := context.WithTimeout(t.Context(), 30*time.Second)
		err := candidate.Start(startCtx)
		startCancel()
		if err == nil {
			app = candidate
			break
		}
		if attempt >= bootAttempts {
			require.NoError(t, err, "fx app must start")
		}
	}

	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		_ = app.Stop(stopCtx)
	})

	baseURL := "http://" + addr
	waitForHTTP(t, baseURL)

	hc := http.DefaultClient
	return &e2eEnv{
		natsURL:     natsURL,
		baseURL:     baseURL,
		connections: connectionsconnect.NewConnectionsServiceClient(hc, baseURL),
		streams:     streamsconnect.NewStreamsServiceClient(hc, baseURL),
		messages:    messagesconnect.NewMessagesServiceClient(hc, baseURL),
		publish:     publishconnect.NewPublishServiceClient(hc, baseURL),
		management:  managementconnect.NewManagementServiceClient(hc, baseURL),
		stats:       statsconnect.NewStatsServiceClient(hc, baseURL),
		live:        liveconnect.NewLiveServiceClient(hc, baseURL),
		mappings:    mappingsconnect.NewMappingsServiceClient(hc, baseURL),
		templates:   templatesconnect.NewTemplatesServiceClient(hc, baseURL),
		settings:    settingsconnect.NewSettingsServiceClient(hc, baseURL),
		history:     historyconnect.NewHistoryServiceClient(hc, baseURL),
		registry:    registryconnect.NewRegistryServiceClient(hc, baseURL),
		codec:       codecconnect.NewCodecServiceClient(hc, baseURL),
		sources:     sourcesconnect.NewSourcesServiceClient(hc, baseURL),
		selections:  selectionsconnect.NewSelectionsServiceClient(hc, baseURL),
		workspace:   workspaceconnect.NewWorkspaceServiceClient(hc, baseURL),
	}
}

// startEmbeddedNATS boots a NATS server with JetStream enabled on a random
// port, storing state under a temp dir, and waits until it is ready.
func startEmbeddedNATS(t *testing.T) *server.Server {
	t.Helper()

	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1, // random free port
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoLog:     true,
		NoSigs:    true,
	}
	srv, err := server.NewServer(opts)
	require.NoError(t, err, "create embedded NATS server")

	go srv.Start()

	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("embedded NATS server not ready within 10s")
	}
	t.Cleanup(srv.Shutdown)
	return srv
}

// freeLocalAddr returns a 127.0.0.1:<port> address that was free when bound
// to :0 and released; a small race window remains before the transport rebinds.
func freeLocalAddr(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())
	return addr
}

// waitForHTTP polls the base URL until the listener accepts a TCP connection
// (the Connect transport serves on a goroutine after Start returns).
func waitForHTTP(t *testing.T, baseURL string) {
	t.Helper()
	host := baseURL[len("http://"):]
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", host, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("connect transport never accepted connections on %s", host)
}
