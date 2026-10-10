// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sections

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore/bbstoretest"

	connectionssvc "github.com/dmit-4884/natscope/internal/services/connections"
	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	mappingsService "github.com/dmit-4884/natscope/internal/services/mappings/mappings"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
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
	for kw := range strings.FieldsSeq(secretSubstrings) {
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
			Username:    new("admin"),
			Password:    new("super-secret-pw"),
			Token:       new("tok-123"),
			NkeySeed:    new("SU-nkey"),
			Credentials: new("creds-blob"),
			JWT:         new("jwt-blob"),
		},
		TLS: &entities.TlsConfig{
			CaCert:     new("-----CA-----"),
			ClientCert: new("-----CLIENT-CERT-----"),
			ClientKey:  new("-----PRIVATEKEY-----"),
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
		Token:      new("ghp_supersecret"),
	}
	payload, err := json.Marshal(newItemsPayload([]protoSourceItem{redactProtoSource(src)}))
	require.NoError(t, err)
	assertNoSecrets(t, payload, "ghp_supersecret")
	assert.Contains(t, string(payload), "git-src")
	assert.Contains(t, string(payload), "acme/proto.git")
}

type fakeSources struct {
	protosvc.SourceManager
	items    entities.ProtoSources
	created  []*entities.ProtoSourceCreate
	selected map[string]string
	failRef  string
	stored   map[string]*entities.SchemaUpload
	uploads  []entities.SchemaUpload
}

func (f *fakeSources) ListSources(context.Context, *entities.ProtoSourcesList) (*entities.List[entities.ProtoSources], error) {
	return &entities.List[entities.ProtoSources]{Items: f.items}, nil
}

func (f *fakeSources) UploadedSchema(_ context.Context, sourceID string) (*entities.SchemaUpload, error) {
	return f.stored[sourceID], nil
}

func (f *fakeSources) UploadSchema(
	_ context.Context,
	_ string,
	upload entities.SchemaUpload,
) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	f.uploads = append(f.uploads, upload)
	return nil, &entities.CompileOutcome{Valid: true}, nil
}

func (f *fakeSources) CreateSource(_ context.Context, in *entities.ProtoSourceCreate) (*entities.ProtoSource, error) {
	f.created = append(f.created, in)
	return entities.ProtoSourceNew(func(s *entities.ProtoSource) { s.Name = in.Name }), nil
}

func (f *fakeSources) SelectRef(_ context.Context, sourceID, ref string) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	if ref == f.failRef {
		return nil, nil, errs.ErrProtoRefNotFound
	}
	f.selected[sourceID] = ref
	return nil, &entities.CompileOutcome{Valid: true}, nil
}

func TestProtoSourcesSection_RoundTripsSelectedRef(t *testing.T) {
	t.Parallel()
	src := &entities.ProtoSource{
		Name: "git-src", SourceType: entities.SourceTypeGit, Repository: "https://github.com/acme/proto.git",
		SelectedRef: &entities.ProtoRef{Name: "main", Kind: entities.RefKindBranch, Revision: "abc"},
	}
	broken := &entities.ProtoSource{
		Name: "broken", SourceType: entities.SourceTypeGit, Repository: "https://github.com/acme/other.git",
		SelectedRef: &entities.ProtoRef{Name: "gone", Kind: entities.RefKindTag, Revision: "def"},
	}
	payload, err := json.Marshal(newItemsPayload([]protoSourceItem{redactProtoSource(src), redactProtoSource(broken)}))
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"ref":"main"`)
	assert.NotContains(t, string(payload), "abc", "revisions are not exported")

	fake := &fakeSources{selected: map[string]string{}, failRef: "gone"}
	res, err := NewProtoSourcesSection(fake).Import(t.Context(), payload, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	assert.Equal(t, int32(2), res.Created)
	require.Len(t, fake.created, 2)
	var picked []string
	for _, ref := range fake.selected {
		picked = append(picked, ref)
	}
	assert.Equal(t, []string{"main"}, picked)
	require.NotEmpty(t, res.Warnings)
	assert.Contains(t, strings.Join(res.Warnings, "\n"), `source "broken": ref "gone" not selected`)
}

func TestProtoSourcesSection_RoundTripsUploads(t *testing.T) {
	t.Parallel()
	files := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
		s.Name, s.SourceType = "uploaded files", entities.SourceTypeUpload
		s.ActiveSchema = &entities.SchemaRevision{Revision: "aaa"}
	})
	set := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
		s.Name, s.SourceType = "uploaded set", entities.SourceTypeUpload
		s.ActiveSchema = &entities.SchemaRevision{Revision: "bbb"}
	})
	empty := entities.ProtoSourceNew(func(s *entities.ProtoSource) { s.Name, s.SourceType = "nothing yet", entities.SourceTypeUpload })
	exporter := &fakeSources{
		items: entities.ProtoSources{files, set, empty},
		stored: map[string]*entities.SchemaUpload{
			files.Id: {Files: []entities.ProtoFileEntry{{Path: "shop/order.proto", Content: "syntax = \"proto3\";"}}},
			set.Id:   {DescriptorSet: []byte{1, 2, 3}},
		},
	}
	payload, err := NewProtoSourcesSection(exporter).Export(t.Context())
	require.NoError(t, err)

	importer := &fakeSources{selected: map[string]string{}}
	res, err := NewProtoSourcesSection(importer).Import(t.Context(), payload, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	assert.Equal(t, int32(3), res.Created)
	assert.Empty(t, res.Warnings)
	require.Len(t, importer.uploads, 2)
	assert.Equal(t, []entities.ProtoFileEntry{{Path: "shop/order.proto", Content: "syntax = \"proto3\";"}}, importer.uploads[0].Files)
	assert.Equal(t, []byte{1, 2, 3}, importer.uploads[1].DescriptorSet)
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
	confluent := entities.Framing{Kind: entities.FramingConfluent, SchemaID: 12}
	custom := entities.Framing{Kind: entities.FramingCustom, Prefix: []byte{0xca, 0xfe}}
	_, err := src.Create(ctx, &entities.SubjectMappingCreate{Pattern: "a.*", MessageType: "pkg.A", SourceID: "s1", Framing: confluent})
	require.NoError(t, err)
	_, err = src.Create(ctx, &entities.SubjectMappingCreate{Pattern: "b.*", MessageType: "pkg.B", SourceID: "s1", Framing: custom})
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
	framings := map[string]entities.Framing{}
	for _, m := range all {
		got[m.Pattern] = m.MessageType
		framings[m.Pattern] = m.Framing
	}
	assert.Equal(t, confluent, framings["a.*"], "the update carries the framing")
	assert.Equal(t, custom, framings["b.*"])
	assert.Equal(t, entities.Framing{}, framings["c.*"])
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

	listed, err := dst.List(ctx, &entities.MessageTemplatesList{Limit: new(int64(100))})
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
		Behavior: &entities.BehaviorSettings{ConfirmDeleteMessage: new(false)},
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
		Auth: &entities.AuthConfig{Method: entities.AuthMethodToken, Token: new("tok-secret")},
		Connection: &entities.ConnectionConfig{
			ConnectionName: new("my-client"),
			NoEcho:         true,
		},
		Reconnect: &entities.ReconnectConfig{MaxReconnects: new(int32(7))},
		Ping:      &entities.PingConfig{MaxPingsOutstanding: new(int32(3))},
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

func TestConnectionsExport_KeepsReadOnlyAndLabel(t *testing.T) {
	t.Parallel()
	item := redactConnection(&entities.SavedConnection{
		Name:     "prod",
		URLs:     []string{"nats://h:4222"},
		ReadOnly: true,
		Label:    &entities.ConnectionLabel{Text: "PROD", Color: entities.LabelColorRed},
	})

	create := converter.Convert(item, &entities.SavedConnectionCreate{})
	assert.True(t, create.ReadOnly)
	require.NotNil(t, create.Label)
	assert.Equal(t, entities.ConnectionLabel{Text: "PROD", Color: entities.LabelColorRed}, *create.Label)
}

// TestRedactConnection_ConverterRoundTrip pins what the converter actually
// copies. Structural redaction only holds if the non-secret fields survive and
// the secret ones do not, and a silent field-name mismatch would break either
// half without failing to compile.
func TestRedactConnection_ConverterRoundTrip(t *testing.T) {
	t.Parallel()

	conn := &entities.SavedConnection{
		Name:        "full",
		Description: new("desc"),
		URLs:        []string{"nats://bob:url-secret@h1:4222", "nats://h2:4222"},
		Auth: &entities.AuthConfig{
			Method:      entities.AuthMethodUserPass,
			Username:    new("bob"),
			Password:    new("pw-secret"),
			Token:       new("tok-secret"),
			NkeySeed:    new("seed-secret"),
			Credentials: new("creds-secret"),
			JWT:         new("jwt-secret"),
		},
		TLS: &entities.TlsConfig{
			CaCert:     new("ca-public"),
			ClientCert: new("clientcert-secret"),
			ClientKey:  new("clientkey-secret"),
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

type fakeConnections struct {
	connectionssvc.Service
	saved   entities.SavedConnections
	deleted []string
	created []string
}

func (f *fakeConnections) List(context.Context, *entities.SavedConnectionsList) (*entities.List[entities.SavedConnections], error) {
	return &entities.List[entities.SavedConnections]{Items: f.saved}, nil
}

func (f *fakeConnections) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeConnections) Create(_ context.Context, in *entities.SavedConnectionCreate) (*entities.SavedConnection, error) {
	f.created = append(f.created, in.Name)
	return &entities.SavedConnection{Name: in.Name}, nil
}

func (f *fakeConnections) ValidateCreate(in *entities.SavedConnectionCreate) error {
	if in.Label != nil && len(in.Label.Text) > 16 {
		return errs.ErrConnectionLabelInvalid
	}
	return nil
}

func TestConnectionsSection_RefusesABadFileBeforeChangingAnything(t *testing.T) {
	t.Parallel()
	long := &entities.ConnectionLabel{Text: "PRODUCTION-EU-WEST", Color: entities.LabelColorRed}
	tests := map[string]struct {
		items []connectionItem
		want  error
	}{
		"a malformed connection": {
			items: []connectionItem{{Name: "ok", URLs: []string{"nats://h:4222"}}, {Name: "bad", URLs: []string{"nats://h:4222"}, Label: long}},
			want:  errs.ErrConnectionLabelInvalid,
		},
		"a name used twice": {
			items: []connectionItem{{Name: "dup", URLs: []string{"nats://h:4222"}}, {Name: "dup", URLs: []string{"nats://g:4222"}}},
			want:  errs.ErrConnectionNameAlreadyInUse,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			svc := &fakeConnections{saved: entities.SavedConnections{entities.SavedConnectionNew(func(c *entities.SavedConnection) { c.Name = "old" })}}
			section := NewConnectionsSection(svc)
			raw, err := json.Marshal(newItemsPayload(tt.items))
			require.NoError(t, err)

			_, err = section.Validate(t.Context(), raw, entities.WorkspaceStrategyReplace)
			require.ErrorIs(t, err, tt.want)
			_, err = section.Import(t.Context(), raw, entities.WorkspaceStrategyReplace)
			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, svc.deleted)
			assert.Empty(t, svc.created)
		})
	}
}

func TestProtoSourcesSection_ImportPassesEveryField(t *testing.T) {
	t.Parallel()
	src := &entities.ProtoSource{
		Name: "local-src", SourceType: entities.SourceTypeLocal, LocalPath: new("/srv/proto"), WatcherEnabled: true,
		ImportRoots: []string{"a", "b"}, ExcludePrefixes: []string{"vendor/"}, Token: new("secret"),
	}
	payload, err := json.Marshal(newItemsPayload([]protoSourceItem{redactProtoSource(src)}))
	require.NoError(t, err)
	assertNoSecrets(t, payload, "secret")

	fake := &fakeSources{selected: map[string]string{}}
	_, err = NewProtoSourcesSection(fake).Import(t.Context(), payload, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	require.Len(t, fake.created, 1)
	got := fake.created[0]
	assert.Equal(t, "local-src", got.Name)
	assert.Equal(t, entities.SourceTypeLocal, got.SourceType)
	assert.Equal(t, "/srv/proto", *got.LocalPath)
	require.NotNil(t, got.WatcherEnabled)
	assert.True(t, *got.WatcherEnabled)
	assert.Equal(t, []string{"a", "b"}, got.ImportRoots)
	assert.Equal(t, []string{"vendor/"}, got.ExcludePrefixes)
	assert.Nil(t, got.Token)

	off := redactProtoSource(&entities.ProtoSource{Name: "n", SourceType: entities.SourceTypeGit})
	assert.Empty(t, off.Ref)
	assert.False(t, off.WatcherEnabled)
}

func TestMappingsSection_ExportOmitsNoneFraming(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	src := newMappingsSvc(t)
	_, err := src.Create(ctx, &entities.SubjectMappingCreate{Pattern: "a.*", MessageType: "pkg.A", SourceID: "s1"})
	require.NoError(t, err)

	raw, err := NewMappingsSection(src, nil).Export(ctx)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "framing")
}

func TestTemplatesSection_ImportKeepsHeadersAndWildcards(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	newSvc := func() *templatesService.Service {
		store, err := templatesBbolt.New(ctx, bbstoretest.NewMemoryDB(t))
		require.NoError(t, err)
		return templatesService.New(store)
	}
	src := newSvc()
	_, err := src.Create(ctx, &entities.MessageTemplateCreate{
		Name: "t1", Subject: "a.*.>", MessageType: "pkg.T", Data: "{}",
		Headers: map[string]string{"H": "1"}, Wildcards: []string{"x", "y"},
	})
	require.NoError(t, err)
	raw, err := NewTemplatesSection(src).Export(ctx)
	require.NoError(t, err)

	dst := newSvc()
	_, err = dst.Create(ctx, &entities.MessageTemplateCreate{Name: "t1", Subject: "old", Data: "old", Headers: map[string]string{"Z": "9"}})
	require.NoError(t, err)
	res, err := NewTemplatesSection(dst).Import(ctx, raw, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	assert.Equal(t, int32(1), res.Updated)

	listed, err := dst.List(ctx, &entities.MessageTemplatesList{Limit: new(int64(100))})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)
	got := listed.Items[0]
	assert.Equal(t, "a.*.>", got.Subject)
	assert.Equal(t, "pkg.T", got.MessageType)
	assert.Equal(t, "{}", got.Data)
	assert.Equal(t, map[string]string{"H": "1"}, got.Headers)
	assert.Equal(t, []string{"x", "y"}, got.Wildcards)

	fresh := newSvc()
	_, err = NewTemplatesSection(fresh).Import(ctx, raw, entities.WorkspaceStrategyReplace)
	require.NoError(t, err)
	listed, err = fresh.List(ctx, &entities.MessageTemplatesList{Limit: new(int64(100))})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)
	assert.Equal(t, map[string]string{"H": "1"}, listed.Items[0].Headers)
	assert.Equal(t, []string{"x", "y"}, listed.Items[0].Wildcards)
}
