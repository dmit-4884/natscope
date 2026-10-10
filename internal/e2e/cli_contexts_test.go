// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestImportCliContexts lists and imports nats CLI contexts from the host and from uploaded files.
func TestImportCliContexts(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "nats", "context")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	write("local.json", `{"description": "laptop", "url": "`+env.natsURL+`", "inbox_prefix": "_INBOX_me"}`)
	write("prod.json", `{"url": "tls://prod:4222", "token": "s3cret", "nsc": "nsc://op/acc/u"}`)
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "nats", "context.txt"), []byte("local"), 0o600))

	listed, err := env.connections.ListCliContexts(ctx, connect.NewRequest(&connectionspb.ListCliContextsRequest{}))
	require.NoError(t, err)
	assert.Equal(t, dir, listed.Msg.GetDirectory())
	require.Len(t, listed.Msg.GetContexts(), 2)
	local := listed.Msg.GetContexts()[0]
	assert.Equal(t, "local", local.GetName())
	assert.True(t, local.GetSelected())
	assert.True(t, local.GetImportable())
	assert.Equal(t, []string{env.natsURL}, local.GetUrls())
	assert.Equal(t, "_INBOX_me", local.GetInboxPrefix())
	prod := listed.Msg.GetContexts()[1]
	assert.Equal(t, natstypes.AuthMethod_AUTH_METHOD_TOKEN, prod.GetAuthMethod())
	assert.Len(t, prod.GetWarnings(), 1)

	imported, err := env.connections.ImportCliContexts(ctx, connect.NewRequest(&connectionspb.ImportCliContextsRequest{
		Names: []string{"local", "prod", "missing"},
	}))
	require.NoError(t, err)
	require.Len(t, imported.Msg.GetConnections(), 2)
	conn := imported.Msg.GetConnections()[0]
	assert.Equal(t, "local", conn.GetName())
	assert.Equal(t, "laptop", conn.GetDescription())
	assert.True(t, imported.Msg.GetConnections()[1].GetAuth().GetHasToken(), "the token went to the vault")
	assert.Empty(t, imported.Msg.GetConnections()[1].GetAuth().GetToken(), "secrets are never echoed")
	require.Len(t, imported.Msg.GetSkipped(), 1)
	assert.Equal(t, "missing", imported.Msg.GetSkipped()[0].GetName())

	_, err = env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{ConnectionId: conn.GetId()}))
	require.NoError(t, err, "the imported connection works")

	again, err := env.connections.ListCliContexts(ctx, connect.NewRequest(&connectionspb.ListCliContextsRequest{}))
	require.NoError(t, err)
	assert.True(t, again.Msg.GetContexts()[0].GetExists())

	t.Run("an import never overwrites a connection of the same name", func(t *testing.T) {
		res, err := env.connections.ImportCliContexts(ctx, connect.NewRequest(&connectionspb.ImportCliContextsRequest{Names: []string{"local"}}))
		require.NoError(t, err)
		assert.Empty(t, res.Msg.GetConnections())
		require.Len(t, res.Msg.GetSkipped(), 1)
	})

	t.Run("uploaded contexts are imported without reading this host's files", func(t *testing.T) {
		secret := filepath.Join(t.TempDir(), "host.creds")
		require.NoError(t, os.WriteFile(secret, []byte("host secret"), 0o600))
		files := []*connectionspb.CliContextFile{{
			Name:    "remote.json",
			Content: []byte(`{"url": "nats://remote:4222", "creds": "` + secret + `"}`),
		}}

		preview, err := env.connections.ListCliContexts(ctx, connect.NewRequest(&connectionspb.ListCliContextsRequest{Files: files}))
		require.NoError(t, err)
		assert.Empty(t, preview.Msg.GetDirectory())
		require.Len(t, preview.Msg.GetContexts(), 1)
		assert.Contains(t, preview.Msg.GetContexts()[0].GetWarnings()[0], "host.creds")

		res, err := env.connections.ImportCliContexts(ctx, connect.NewRequest(&connectionspb.ImportCliContextsRequest{
			Names: []string{"remote"}, Files: files,
		}))
		require.NoError(t, err)
		require.Len(t, res.Msg.GetConnections(), 1)
		assert.Nil(t, res.Msg.GetConnections()[0].GetAuth())
	})
}
