// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sections

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore/bbstoretest"

	ptr "github.com/altessa-s/go-atlas/core/types/ptr"
	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	mappingsService "github.com/dmit-4884/natscope/internal/services/mappings/mappings"
	settingsService "github.com/dmit-4884/natscope/internal/services/settings/settings"
	templatesService "github.com/dmit-4884/natscope/internal/services/templates/templates"
	mappingsBbolt "github.com/dmit-4884/natscope/internal/storages/mappings/bbolt"
	settingsBbolt "github.com/dmit-4884/natscope/internal/storages/settings/bbolt"
	templatesBbolt "github.com/dmit-4884/natscope/internal/storages/templates/bbolt"
)

const secretSubstrings = "password token nkey credentials jwt clientKey clientCert privatekey"

// assertNoSecrets fails if the payload contains any secret-looking key or supplied secret value.
func assertNoSecrets(t *testing.T, payload []byte, values ...string) {
	t.Helper()
	lower := strings.ToLower(string(payload))
	for _, kw := range strings.Fields(secretSubstrings) {
		assert.NotContainsf(t, lower, kw, "export must not contain %q", kw)
	}
	for _, v := range values {
		assert.NotContainsf(t, string(payload), v, "export must not contain secret value %q", v)
	}
}

func TestConnectionsExport_HasNoSecrets(t *testing.T) {
	t.Parallel()
	conn := &entities.SavedConnection{
		Name: "prod",
		URLs: []string{"nats://prod:4222"},
		Auth: &entities.AuthConfig{
			Method:      entities.AuthMethodUserPass,
			Username:    ptr.Wrap("admin"),
			Password:    ptr.Wrap("super-secret-pw"),
			Token:       ptr.Wrap("tok-123"),
			NkeySeed:    ptr.Wrap("SU-nkey"),
			Credentials: ptr.Wrap("creds-blob"),
			JWT:         ptr.Wrap("jwt-blob"),
		},
		TLS: &entities.TlsConfig{
			CaCert:     ptr.Wrap("-----CA-----"),
			ClientCert: ptr.Wrap("-----CLIENT-CERT-----"),
			ClientKey:  ptr.Wrap("-----PRIVATEKEY-----"),
			SkipVerify: true,
		},
	}
	payload, err := json.Marshal(newItemsPayload([]connectionItem{redactConnection(conn)}))
	require.NoError(t, err)

	assertNoSecrets(t, payload,
		"super-secret-pw", "tok-123", "SU-nkey", "creds-blob", "jwt-blob", "-----CLIENT-CERT-----", "-----PRIVATEKEY-----")

	// Non-secret fields survive.
	s := string(payload)
	assert.Contains(t, s, "prod")
	assert.Contains(t, s, "admin")        // username is not secret
	assert.Contains(t, s, "-----CA-----") // public CA cert is shareable
}

func TestProtoSourcesExport_HasNoToken(t *testing.T) {
	t.Parallel()
	src := &entities.ProtoSource{
		Name:       "git-src",
		SourceType: entities.SourceTypeGit,
		Repository: "https://github.com/acme/proto.git",
		Token:      ptr.Wrap("ghp_supersecret"),
	}
	payload, err := json.Marshal(newItemsPayload([]protoSourceItem{redactProtoSource(src)}))
	require.NoError(t, err)
	assertNoSecrets(t, payload, "ghp_supersecret")
	assert.Contains(t, string(payload), "git-src")
	assert.Contains(t, string(payload), "acme/proto.git")
}

// --- mappings section roundtrip (real SQLite-backed service) ---

func newMappingsSvc(t *testing.T) mappingssvc.Service {
	t.Helper()
	store, err := mappingsBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	require.NoError(t, err)
	return mappingsService.New(store)
}

func TestMappingsSection_ExportThenImportMerge(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	src := newMappingsSvc(t)
	_, err := src.Create(ctx, &entities.SubjectMappingCreate{Pattern: "a.*", MessageType: "pkg.A", SourceID: "s1"})
	require.NoError(t, err)
	_, err = src.Create(ctx, &entities.SubjectMappingCreate{Pattern: "b.*", MessageType: "pkg.B", SourceID: "s1"})
	require.NoError(t, err)

	raw, err := NewMappingsSection(src, nil).Export(ctx)
	require.NoError(t, err)

	// Fresh target with one overlapping pattern (a.*) and one unique (c.*).
	dst := newMappingsSvc(t)
	_, err = dst.Create(ctx, &entities.SubjectMappingCreate{Pattern: "a.*", MessageType: "old.A", SourceID: "s1"})
	require.NoError(t, err)
	_, err = dst.Create(ctx, &entities.SubjectMappingCreate{Pattern: "c.*", MessageType: "pkg.C", SourceID: "s1"})
	require.NoError(t, err)

	section := NewMappingsSection(dst, nil)

	rep, err := section.Validate(ctx, raw, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	assert.Equal(t, int32(1), rep.Created, "b.* is new")
	assert.Equal(t, int32(1), rep.Updated, "a.* overlaps")
	assert.Equal(t, []string{"a.*"}, rep.Conflicts)

	res, err := section.Import(ctx, raw, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	assert.Equal(t, int32(1), res.Created)
	assert.Equal(t, int32(1), res.Updated)

	all, err := dst.GetAll(ctx)
	require.NoError(t, err)
	got := map[string]string{}
	for _, m := range all {
		got[m.Pattern] = m.MessageType
	}
	assert.Equal(t, "pkg.A", got["a.*"], "a.* updated from import")
	assert.Equal(t, "pkg.B", got["b.*"], "b.* created")
	assert.Equal(t, "pkg.C", got["c.*"], "c.* untouched by merge")
}

func TestMappingsSection_ImportReplace(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	src := newMappingsSvc(t)
	_, _ = src.Create(ctx, &entities.SubjectMappingCreate{Pattern: "x.*", MessageType: "pkg.X", SourceID: "s1"})
	raw, err := NewMappingsSection(src, nil).Export(ctx)
	require.NoError(t, err)

	dst := newMappingsSvc(t)
	_, _ = dst.Create(ctx, &entities.SubjectMappingCreate{Pattern: "old.*", MessageType: "old", SourceID: "s1"})
	_, _ = dst.Create(ctx, &entities.SubjectMappingCreate{Pattern: "stale.*", MessageType: "stale", SourceID: "s1"})

	section := NewMappingsSection(dst, nil)
	res, err := section.Import(ctx, raw, entities.WorkspaceStrategyReplace)
	require.NoError(t, err)
	assert.Equal(t, int32(2), res.Deleted)
	assert.Equal(t, int32(1), res.Created)

	all, err := dst.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "x.*", all[0].Pattern)
}

// --- templates section roundtrip ---

func TestTemplatesSection_Roundtrip(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	srcStore, err := templatesBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	require.NoError(t, err)
	src := templatesService.New(srcStore)
	_, err = src.Create(ctx, &entities.MessageTemplateCreate{Name: "t1", Subject: "a.b", Data: "{}"})
	require.NoError(t, err)

	raw, err := NewTemplatesSection(src).Export(ctx)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "t1")

	dstStore, err := templatesBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	require.NoError(t, err)
	dst := templatesService.New(dstStore)
	res, err := NewTemplatesSection(dst).Import(ctx, raw, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	assert.Equal(t, int32(1), res.Created)

	listed, err := dst.List(ctx, &entities.MessageTemplatesList{ListBase: entities.ListBase{Limit: ptr.Wrap(int64(100))}})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)
	assert.Equal(t, "t1", listed.Items[0].Name)
}

// --- settings section roundtrip ---

func TestSettingsSection_Roundtrip(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	srcStore, err := settingsBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	require.NoError(t, err)
	src := settingsService.New(srcStore)
	_, err = src.Update(ctx, &entities.UserSettingsUpdate{
		Behavior: &entities.BehaviorSettings{ConfirmDeleteMessage: ptr.Wrap(false)},
	})
	require.NoError(t, err)

	raw, err := NewSettingsSection(src).Export(ctx)
	require.NoError(t, err)

	dstStore, err := settingsBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	require.NoError(t, err)
	dst := settingsService.New(dstStore)
	res, err := NewSettingsSection(dst).Import(ctx, raw, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	assert.Equal(t, int32(1), res.Updated)

	got, err := dst.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, got.Behavior)
	require.NotNil(t, got.Behavior.ConfirmDeleteMessage)
	assert.False(t, *got.Behavior.ConfirmDeleteMessage)
}

func TestPlans(t *testing.T) {
	t.Parallel()
	existing := map[string]struct{}{"a": {}, "b": {}}

	c, u, d, conf := upsertPlan(existing, []string{"a", "x"}, entities.WorkspaceStrategyMerge)
	assert.Equal(t, int32(1), c) // x
	assert.Equal(t, int32(1), u) // a
	assert.Equal(t, int32(0), d) // merge never deletes
	assert.Equal(t, []string{"a"}, conf)

	c, u, d, _ = upsertPlan(existing, []string{"a", "x"}, entities.WorkspaceStrategyReplace)
	assert.Equal(t, int32(2), c)
	assert.Equal(t, int32(0), u)
	assert.Equal(t, int32(2), d) // replace clears both existing

	// create-only merge keeps existing untouched (only new names created).
	c, d, conf = createOnlyPlan(existing, []string{"a", "x"}, entities.WorkspaceStrategyMerge)
	assert.Equal(t, int32(1), c) // only x created
	assert.Equal(t, int32(0), d)
	assert.Equal(t, []string{"a"}, conf)

	// create-only replace deletes every existing item before recreating.
	c, d, _ = createOnlyPlan(existing, []string{"a", "x"}, entities.WorkspaceStrategyReplace)
	assert.Equal(t, int32(2), c)
	assert.Equal(t, int32(2), d)
}

// TestMappingsResolveSourceID guards the M3 cross-machine remap: portable source NAME wins over the file's machine-local id, falling back when unknown.
func TestMappingsResolveSourceID(t *testing.T) {
	t.Parallel()
	s := &MappingsSection{}
	nameToID := map[string]string{"prod-src": "local-id-123"}

	// Name resolves locally → use the LOCAL id, not the file's remote id.
	assert.Equal(t, "local-id-123",
		s.resolveSourceID(mappingItem{SourceName: "prod-src", SourceID: "old-remote-id"}, nameToID))
	// Name present but unknown locally → fall back to the file's raw id.
	assert.Equal(t, "old-remote-id",
		s.resolveSourceID(mappingItem{SourceName: "missing", SourceID: "old-remote-id"}, nameToID))
	// No name (legacy/same-machine file) → raw id round-trips.
	assert.Equal(t, "raw-id",
		s.resolveSourceID(mappingItem{SourceID: "raw-id"}, nameToID))
}

// TestConnectionsExport_KeepsNonSecretConfig guards m10: Connection/Reconnect/Ping settings survive redaction while secrets never appear.
func TestConnectionsExport_KeepsNonSecretConfig(t *testing.T) {
	t.Parallel()
	conn := &entities.SavedConnection{
		Name: "tuned",
		URLs: []string{"nats://h:4222"},
		Auth: &entities.AuthConfig{Method: entities.AuthMethodToken, Token: ptr.Wrap("tok-secret")},
		Connection: &entities.ConnectionConfig{
			ConnectionName: ptr.Wrap("my-client"),
			NoEcho:         true,
		},
		Reconnect: &entities.ReconnectConfig{MaxReconnects: ptr.Wrap(int32(7))},
		Ping:      &entities.PingConfig{MaxPingsOutstanding: ptr.Wrap(int32(3))},
	}
	item := redactConnection(conn)
	require.NotNil(t, item.Connection)
	assert.Equal(t, "my-client", *item.Connection.ConnectionName)
	assert.True(t, item.Connection.NoEcho)
	require.NotNil(t, item.Reconnect)
	assert.Equal(t, int32(7), *item.Reconnect.MaxReconnects)
	require.NotNil(t, item.Ping)
	assert.Equal(t, int32(3), *item.Ping.MaxPingsOutstanding)

	payload, err := json.Marshal(newItemsPayload([]connectionItem{item}))
	require.NoError(t, err)
	assertNoSecrets(t, payload, "tok-secret")
}

// TestRedactConnection_ConverterRoundTrip pins what the converter actually
// copies. Structural redaction only holds if the non-secret fields survive and
// the secret ones do not, and a silent field-name mismatch would break either
// half without failing to compile.
func TestRedactConnection_ConverterRoundTrip(t *testing.T) {
	t.Parallel()

	conn := &entities.SavedConnection{
		Name:        "full",
		Description: ptr.Wrap("desc"),
		URLs:        []string{"nats://bob:url-secret@h1:4222", "nats://h2:4222"},
		Auth: &entities.AuthConfig{
			Method:      entities.AuthMethodUserPass,
			Username:    ptr.Wrap("bob"),
			Password:    ptr.Wrap("pw-secret"),
			Token:       ptr.Wrap("tok-secret"),
			NkeySeed:    ptr.Wrap("seed-secret"),
			Credentials: ptr.Wrap("creds-secret"),
			JWT:         ptr.Wrap("jwt-secret"),
		},
		TLS: &entities.TlsConfig{
			CaCert:     ptr.Wrap("ca-public"),
			ClientCert: ptr.Wrap("clientcert-secret"),
			ClientKey:  ptr.Wrap("clientkey-secret"),
			SkipVerify: true,
			TlsFirst:   true,
		},
	}

	item := redactConnection(conn)

	assert.Equal(t, "full", item.Name)
	require.NotNil(t, item.Description)
	assert.Equal(t, "desc", *item.Description)

	require.NotNil(t, item.Auth, "auth projection must be populated")
	assert.Equal(t, entities.AuthMethodUserPass, item.Auth.Method)
	require.NotNil(t, item.Auth.Username)
	assert.Equal(t, "bob", *item.Auth.Username)

	require.NotNil(t, item.TLS, "tls projection must be populated")
	require.NotNil(t, item.TLS.CaCert)
	assert.Equal(t, "ca-public", *item.TLS.CaCert)
	assert.True(t, item.TLS.SkipVerify)
	assert.True(t, item.TLS.TlsFirst)

	assert.Equal(t, []string{"nats://h1:4222", "nats://h2:4222"}, item.URLs)

	payload, err := json.Marshal(newItemsPayload([]connectionItem{item}))
	require.NoError(t, err)
	assertNoSecrets(t, payload,
		"url-secret", "pw-secret", "tok-secret", "seed-secret",
		"creds-secret", "jwt-secret", "clientcert-secret", "clientkey-secret")

	// Import side: the same shape must rebuild the create DTO without inventing
	// credentials the export never carried.
	create := converter.Convert(item, &entities.SavedConnectionCreate{})
	assert.Equal(t, "full", create.Name)
	assert.Equal(t, []string{"nats://h1:4222", "nats://h2:4222"}, create.URLs)
	require.NotNil(t, create.Auth)
	assert.Equal(t, entities.AuthMethodUserPass, create.Auth.Method)
	require.NotNil(t, create.Auth.Username)
	assert.Equal(t, "bob", *create.Auth.Username)
	assert.Nil(t, create.Auth.Password)
	assert.Nil(t, create.Auth.Token)
	assert.Nil(t, create.Auth.NkeySeed)
	assert.Nil(t, create.Auth.Credentials)
	assert.Nil(t, create.Auth.JWT)
	require.NotNil(t, create.TLS)
	assert.Nil(t, create.TLS.ClientCert)
	assert.Nil(t, create.TLS.ClientKey)
	assert.True(t, create.TLS.SkipVerify)
}

func TestRedactConnection_NoAuthOrTLS(t *testing.T) {
	t.Parallel()

	item := redactConnection(&entities.SavedConnection{
		Name: "bare",
		URLs: []string{"nats://h:4222"},
	})

	assert.Nil(t, item.Auth)
	assert.Nil(t, item.TLS)
	assert.Nil(t, converter.Convert(item, &entities.SavedConnectionCreate{}).Auth)
}
