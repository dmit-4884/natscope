// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore/bbstoretest"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	fwsvc "github.com/dmit-4884/natscope/internal/services/filewatcher"
	gitfetchersvc "github.com/dmit-4884/natscope/internal/services/gitfetcher"
	conflictsBbolt "github.com/dmit-4884/natscope/internal/storages/proto/conflicts/bbolt"
	descriptorsBbolt "github.com/dmit-4884/natscope/internal/storages/proto/descriptors/bbolt"
	filesetsBbolt "github.com/dmit-4884/natscope/internal/storages/proto/filesets/bbolt"
	sourcesBbolt "github.com/dmit-4884/natscope/internal/storages/proto/sources/bbolt"
)

const (
	orderV1 = `syntax = "proto3";
package shop;
message Order { string id = 1; }
`
	orderV2 = `syntax = "proto3";
package shop;
message Order { string id = 1; int64 total = 2; }
message Refund { string order_id = 1; }
`
	brokenProto = `syntax = "proto3";
package shop;
message Order { strin id = 1; }
`
)

type fakeGit struct {
	mu      sync.Mutex
	refs    []entities.ProtoRef
	trees   map[string]string
	fetches int
}

func newFakeGit() *fakeGit {
	return &fakeGit{trees: map[string]string{}}
}

func (g *fakeGit) commit(revision, content string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.trees[revision] = content
}

func (g *fakeGit) point(name string, kind entities.RefKind, revision string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.refs {
		if g.refs[i].Name == name {
			g.refs[i].Revision = revision
			return
		}
	}
	g.refs = append(g.refs, entities.ProtoRef{Name: name, Kind: kind, Revision: revision})
}

func (g *fakeGit) fetchCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.fetches
}

func (g *fakeGit) ListRefs(context.Context, string) ([]entities.ProtoRef, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]entities.ProtoRef(nil), g.refs...), nil
}

func (g *fakeGit) ResolveRef(_ context.Context, _ string, ref string) (entities.ProtoRef, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, r := range g.refs {
		if r.Name == ref {
			return r, nil
		}
	}
	if _, ok := g.trees[ref]; ok {
		return entities.ProtoRef{Name: ref, Kind: entities.RefKindCommit, Revision: ref}, nil
	}
	return entities.ProtoRef{}, fmt.Errorf("%w: %q", errs.ErrProtoRefNotFound, ref)
}

func (g *fakeGit) Fetch(_ context.Context, _ string, ref entities.ProtoRef) (*gitfetchersvc.FetchResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fetches++
	content, ok := g.trees[ref.Revision]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errs.ErrProtoRefNotFound, ref.Name)
	}
	return &gitfetchersvc.FetchResult{
		Revision: ref.Revision,
		Files:    []entities.ProtoFileEntry{{Path: "shop.proto", Content: content, Size: int64(len(content))}},
	}, nil
}

func (g *fakeGit) ValidateRepository(context.Context, string) error { return nil }

type fakeWatcher struct {
	mu      sync.Mutex
	watched map[string]string
}

func (w *fakeWatcher) Watch(sourceID, dirPath string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watched == nil {
		w.watched = map[string]string{}
	}
	w.watched[sourceID] = dirPath
	return nil
}

func (w *fakeWatcher) Unwatch(sourceID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.watched, sourceID)
}

func (w *fakeWatcher) isWatching(sourceID string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.watched[sourceID]
	return ok
}

func (w *fakeWatcher) SetCallback(fwsvc.ChangeCallback) {}
func (w *fakeWatcher) Start() error                     { return nil }
func (w *fakeWatcher) Stop()                            {}

type testEnv struct {
	svc         *Service
	git         *fakeGit
	watcher     *fakeWatcher
	descriptors *descriptorsBbolt.Storage
	fileSets    *filesetsBbolt.Storage
	sources     *sourcesBbolt.Storage
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := t.Context()
	db := bbstoretest.NewMemoryDB(t)
	sources, err := sourcesBbolt.New(ctx, db, secrets.NewMemory())
	require.NoError(t, err)
	fileSets, err := filesetsBbolt.New(ctx, db)
	require.NoError(t, err)
	descriptors, err := descriptorsBbolt.New(ctx, db)
	require.NoError(t, err)
	conflicts, err := conflictsBbolt.New(ctx, db)
	require.NoError(t, err)

	env := &testEnv{git: newFakeGit(), watcher: &fakeWatcher{}, descriptors: descriptors, fileSets: fileSets, sources: sources}
	env.svc = New(sources, fileSets, descriptors, conflicts, env.git, nil, env.watcher)
	return env
}

func newBareService() *Service {
	return New(nil, nil, nil, nil, nil, nil, nil)
}

func (e *testEnv) createGit(t *testing.T) *entities.ProtoSource {
	t.Helper()
	src, err := e.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
		Name: "git-" + t.Name(), SourceType: entities.SourceTypeGit, Repository: "https://example.com/shop.git",
	})
	require.NoError(t, err)
	return src
}

func (e *testEnv) createLocal(t *testing.T, files map[string]string) (*entities.ProtoSource, string) {
	t.Helper()
	dir := t.TempDir()
	writeTree(t, dir, files)
	src, err := e.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
		Name: "local-" + t.Name(), SourceType: entities.SourceTypeLocal, LocalPath: &dir,
	})
	require.NoError(t, err)
	return src, dir
}

func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
}

func messageNames(svc *Service, t *testing.T, sourceID string) []string {
	t.Helper()
	types, err := svc.ListTypes(t.Context(), "")
	require.NoError(t, err)
	var out []string
	for _, m := range types {
		if m.SourceID == sourceID && m.Kind == entities.SchemaTypeMessage {
			out = append(out, m.FullName)
		}
	}
	return out
}

func TestSelectRef_CompilesAndActivates(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.git.commit("aaa", orderV1)
	env.git.point("v1.0.0", entities.RefKindTag, "aaa")
	src := env.createGit(t)

	got, outcome, err := env.svc.SelectRef(t.Context(), src.Id, "v1.0.0")
	require.NoError(t, err)
	require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
	assert.Equal(t, 1, outcome.MessageTypes)
	assert.Equal(t, &entities.ProtoRef{Name: "v1.0.0", Kind: entities.RefKindTag, Revision: "aaa"}, got.SelectedRef)
	require.NotNil(t, got.ActiveSchema)
	assert.Equal(t, "aaa", got.ActiveSchema.Revision)
	assert.NotEmpty(t, got.ActiveSchema.Fingerprint)
	assert.True(t, got.LastCompile.Ok)
	assert.Equal(t, []string{"shop.Order"}, messageNames(env.svc, t, src.Id))

	stored, err := env.sources.Get(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Equal(t, got.SelectedRef, stored.SelectedRef, "selected ref is persisted")
	assert.Equal(t, got.ActiveSchema, stored.ActiveSchema, "active schema is persisted")
}

func TestSelectRef_ReusesCompiledRevision(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.git.commit("aaa", orderV1)
	env.git.point("v1.0.0", entities.RefKindTag, "aaa")
	env.git.point("main", entities.RefKindBranch, "aaa")
	src := env.createGit(t)

	_, _, err := env.svc.SelectRef(t.Context(), src.Id, "v1.0.0")
	require.NoError(t, err)
	got, outcome, err := env.svc.SelectRef(t.Context(), src.Id, "main")
	require.NoError(t, err)
	assert.True(t, outcome.Valid)
	assert.Equal(t, "main", got.SelectedRef.Name)
	assert.Equal(t, 1, env.git.fetchCount(), "a revision compiled once is not fetched again")
}

func TestRefreshSource_FollowsMovedBranch(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.git.commit("aaa", orderV1)
	env.git.commit("bbb", orderV2)
	env.git.point("main", entities.RefKindBranch, "aaa")
	src := env.createGit(t)

	first, _, err := env.svc.SelectRef(t.Context(), src.Id, "main")
	require.NoError(t, err)
	oldFingerprint := first.ActiveSchema.Fingerprint

	env.git.point("main", entities.RefKindBranch, "bbb")
	got, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	require.True(t, outcome.Valid)
	assert.Equal(t, "bbb", got.SelectedRef.Revision)
	assert.Equal(t, "bbb", got.ActiveSchema.Revision)
	assert.NotEqual(t, oldFingerprint, got.ActiveSchema.Fingerprint)
	assert.ElementsMatch(t, []string{"shop.Order", "shop.Refund"}, messageNames(env.svc, t, src.Id))

	revisions, err := env.svc.ListRevisions(t.Context(), src.Id)
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	assert.Equal(t, "bbb", revisions[0].Revision, "newest first")
	assert.True(t, revisions[0].Active)
	assert.False(t, revisions[1].Active)
	assert.Equal(t, oldFingerprint, revisions[1].Fingerprint)

	pinned, err := env.svc.Decode(t.Context(), entities.CodecRequest{
		SourceID: src.Id, Fingerprint: oldFingerprint, MessageType: "shop.Refund", Data: []byte{0x0a, 0x01, 'x'},
	})
	require.NoError(t, err)
	assert.False(t, pinned.Success, "the pinned old schema has no Refund")
	assert.Contains(t, pinned.Error, "not found")
}

func TestSelectRef_CompileErrorKeepsPreviousSchema(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.git.commit("aaa", orderV1)
	env.git.commit("bad", brokenProto)
	env.git.point("v1", entities.RefKindTag, "aaa")
	env.git.point("v2", entities.RefKindTag, "bad")
	src := env.createGit(t)

	_, _, err := env.svc.SelectRef(t.Context(), src.Id, "v1")
	require.NoError(t, err)

	got, outcome, err := env.svc.SelectRef(t.Context(), src.Id, "v2")
	require.NoError(t, err, "compile errors come back in the outcome")
	assert.False(t, outcome.Valid)
	require.NotEmpty(t, outcome.Diagnostics)
	assert.Equal(t, "v1", got.SelectedRef.Name)
	assert.Equal(t, "aaa", got.ActiveSchema.Revision)
	assert.False(t, got.LastCompile.Ok)

	_, err = env.fileSets.GetBySourceRevision(t.Context(), src.Id, "bad")
	assert.ErrorIs(t, err, errs.ErrProtoFileSetNotFound, "files of a broken revision are not kept")
}

func TestSelectRef_Errors(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	gitSrc := env.createGit(t)
	localSrc, _ := env.createLocal(t, map[string]string{"a.proto": orderV1})

	_, _, err := env.svc.SelectRef(t.Context(), gitSrc.Id, "nope")
	require.ErrorIs(t, err, errs.ErrProtoRefNotFound)

	_, _, err = env.svc.SelectRef(t.Context(), localSrc.Id, "main")
	require.ErrorIs(t, err, errs.ErrInvalidRequest)

	_, err = env.svc.ListRefs(t.Context(), localSrc.Id)
	require.ErrorIs(t, err, errs.ErrInvalidRequest)

	_, _, err = env.svc.RefreshSource(t.Context(), gitSrc.Id)
	require.ErrorIs(t, err, errs.ErrInvalidRequest, "refresh needs a selected ref")

	_, _, err = env.svc.SelectRef(t.Context(), "missing", "main")
	require.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
}

func TestRefreshSource_LocalRecompiles(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src, dir := env.createLocal(t, map[string]string{"shop.proto": orderV1})

	first, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	require.True(t, outcome.Valid)
	assert.Equal(t, LocalRevision, first.ActiveSchema.Revision)

	writeTree(t, dir, map[string]string{"shop.proto": orderV2})
	second, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	require.True(t, outcome.Valid)
	assert.NotEqual(t, first.ActiveSchema.Fingerprint, second.ActiveSchema.Fingerprint)
	assert.ElementsMatch(t, []string{"shop.Order", "shop.Refund"}, messageNames(env.svc, t, src.Id))

	revisions, err := env.svc.ListRevisions(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Len(t, revisions, 1, "a local source keeps only its latest schema")
}

func TestRefreshSource_LocalDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("broken import returns diagnostics with a hint", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		src, _ := env.createLocal(t, map[string]string{"a.proto": "syntax = \"proto3\";\nimport \"missing.proto\";\nmessage A {}\n"})

		got, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
		require.NoError(t, err)
		assert.False(t, outcome.Valid)
		var hint string
		for _, d := range outcome.Diagnostics {
			if d.MissingImport == "missing.proto" {
				hint = d.Hint
			}
		}
		assert.NotEmpty(t, hint)
		assert.Nil(t, got.ActiveSchema)
		assert.False(t, got.LastCompile.Ok)
	})

	t.Run("inferred roots are recorded", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		src, _ := env.createLocal(t, map[string]string{
			"proto/common/t.proto": "syntax = \"proto3\";\npackage c;\nmessage T {}\n",
			"proto/api/s.proto":    "syntax = \"proto3\";\npackage a;\nimport \"common/t.proto\";\nmessage S { c.T t = 1; }\n",
		})

		got, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
		require.NoError(t, err)
		require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
		assert.Equal(t, []string{"proto"}, got.LastCompile.Roots)
		assert.Equal(t, "inferred", got.LastCompile.RootsOrigin)
	})

	t.Run("missing local path", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		src, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{Name: "nopath", SourceType: entities.SourceTypeLocal})
		require.NoError(t, err)
		_, _, err = env.svc.RefreshSource(t.Context(), src.Id)
		require.ErrorIs(t, err, errs.ErrInvalidRequest)
	})
}

func TestDeleteSource_PurgesSchemasAndFiles(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.git.commit("aaa", orderV1)
	env.git.point("v1", entities.RefKindTag, "aaa")
	src := env.createGit(t)
	_, _, err := env.svc.SelectRef(t.Context(), src.Id, "v1")
	require.NoError(t, err)

	require.NoError(t, env.svc.DeleteSource(t.Context(), src.Id))

	schemas, err := env.descriptors.ListBySource(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Empty(t, schemas)
	_, err = env.fileSets.GetBySourceRevision(t.Context(), src.Id, "aaa")
	require.ErrorIs(t, err, errs.ErrProtoFileSetNotFound)
	assert.Empty(t, messageNames(env.svc, t, src.Id))
}

func TestStart_BuildsMissingSchemasAndRefreshesBranches(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.git.commit("aaa", orderV1)
	env.git.commit("bbb", orderV2)
	env.git.point("v1", entities.RefKindTag, "aaa")
	env.git.point("main", entities.RefKindBranch, "aaa")

	tagSrc := env.createGit(t)
	tagSrc.SelectedRef = &entities.ProtoRef{Name: "v1", Kind: entities.RefKindTag, Revision: "aaa"}
	require.NoError(t, env.sources.Update(t.Context(), tagSrc))

	branchSrc, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
		Name: "branch", SourceType: entities.SourceTypeGit, Repository: "https://example.com/shop.git",
	})
	require.NoError(t, err)
	_, _, err = env.svc.SelectRef(t.Context(), branchSrc.Id, "main")
	require.NoError(t, err)
	env.git.point("main", entities.RefKindBranch, "bbb")

	localSrc, _ := env.createLocal(t, map[string]string{"shop.proto": orderV1})
	_, err = env.svc.SetWatcher(t.Context(), localSrc.Id, true)
	require.NoError(t, err)
	env.watcher.Unwatch(localSrc.Id)

	env.svc.Start(t.Context())

	require.Eventually(t, func() bool {
		tagOK, _ := env.svc.hasSchema(t.Context(), tagSrc.Id, "aaa")
		localOK, _ := env.svc.hasSchema(t.Context(), localSrc.Id, LocalRevision)
		branch, err := env.sources.Get(t.Context(), branchSrc.Id)
		return tagOK && localOK && err == nil && branch.SelectedRef.Revision == "bbb"
	}, 10*time.Second, 20*time.Millisecond)
	assert.True(t, env.watcher.isWatching(localSrc.Id))
}

func TestCreateSource(t *testing.T) {
	t.Parallel()

	t.Run("git", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		src, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name: "core-proto", SourceType: entities.SourceTypeGit,
			Repository: "https://gitlab.com/org/proto.git", Token: new("my-token"),
		})
		require.NoError(t, err)
		stored, err := env.sources.Get(t.Context(), src.Id)
		require.NoError(t, err)
		assert.Equal(t, "core-proto", stored.Name)
		assert.True(t, stored.Enabled)
		assert.Equal(t, "my-token", *stored.Token)
		assert.Nil(t, stored.SelectedRef)
	})

	t.Run("local starts the watcher only when asked", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		dir := t.TempDir()
		watched, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name: "watched", SourceType: entities.SourceTypeLocal, LocalPath: &dir, WatcherEnabled: new(true),
		})
		require.NoError(t, err)
		plain, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name: "plain", SourceType: entities.SourceTypeLocal, LocalPath: &dir,
		})
		require.NoError(t, err)
		assert.True(t, env.watcher.isWatching(watched.Id))
		assert.False(t, env.watcher.isWatching(plain.Id))
	})

	t.Run("duplicate name", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		in := &entities.ProtoSourceCreate{Name: "dup", SourceType: entities.SourceTypeGit}
		_, err := env.svc.CreateSource(t.Context(), in)
		require.NoError(t, err)
		_, err = env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{Name: "dup", SourceType: entities.SourceTypeGit})
		require.ErrorIs(t, err, errs.ErrProtoSourceNameAlreadyInUse)
	})
}

func TestUpdateSource(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
		Name: "old", SourceType: entities.SourceTypeGit, Token: new("secret"),
	})
	require.NoError(t, err)

	got, err := env.svc.UpdateSource(t.Context(), &entities.ProtoSourceUpdate{Id: src.Id, Name: new("new"), Token: new("")})
	require.NoError(t, err)
	assert.Equal(t, "new", got.Name)
	assert.Nil(t, got.Token, "an empty token removes it")

	_, err = env.svc.UpdateSource(t.Context(), &entities.ProtoSourceUpdate{Id: "missing", Name: new("x")})
	require.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
}

func TestSetEnabled(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src, _ := env.createLocal(t, map[string]string{"shop.proto": orderV1})
	_, _, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	_, err = env.svc.SetWatcher(t.Context(), src.Id, true)
	require.NoError(t, err)

	off, err := env.svc.SetEnabled(t.Context(), src.Id, false)
	require.NoError(t, err)
	assert.False(t, off.Enabled)
	assert.False(t, env.watcher.isWatching(src.Id))
	assert.Empty(t, messageNames(env.svc, t, src.Id), "a disabled source decodes nothing")

	on, err := env.svc.SetEnabled(t.Context(), src.Id, true)
	require.NoError(t, err)
	assert.True(t, on.Enabled)
	assert.True(t, env.watcher.isWatching(src.Id))
	assert.Equal(t, []string{"shop.Order"}, messageNames(env.svc, t, src.Id))

	_, err = env.svc.SetEnabled(t.Context(), "missing", true)
	require.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
}

func TestSetWatcher(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	local, _ := env.createLocal(t, nil)
	gitSrc := env.createGit(t)

	got, err := env.svc.SetWatcher(t.Context(), local.Id, true)
	require.NoError(t, err)
	assert.True(t, got.WatcherEnabled)
	assert.True(t, env.watcher.isWatching(local.Id))

	_, err = env.svc.SetWatcher(t.Context(), local.Id, false)
	require.NoError(t, err)
	assert.False(t, env.watcher.isWatching(local.Id))

	_, err = env.svc.SetWatcher(t.Context(), gitSrc.Id, true)
	require.ErrorIs(t, err, errs.ErrInvalidRequest)

	_, err = env.svc.SetWatcher(t.Context(), "missing", true)
	require.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
}

func TestValidateLocalPath(t *testing.T) {
	t.Parallel()
	svc := newBareService()

	valid := t.TempDir()
	writeTree(t, valid, map[string]string{
		"test.proto":            `syntax = "proto3";`,
		"sub/nested.proto":      `syntax = "proto3";`,
		"readme.md":             "# readme",
		".git/hidden.proto":     "x",
		"node_modules/d/x.proto": "x",
	})
	got, err := svc.ValidateLocalPath(t.Context(), valid)
	require.NoError(t, err)
	assert.True(t, got.Valid)
	assert.Equal(t, 2, got.ProtoFileCount, ".git and node_modules are not counted")

	filePath := filepath.Join(t.TempDir(), "not-a-dir.proto")
	require.NoError(t, os.WriteFile(filePath, []byte(`syntax = "proto3";`), 0o644))
	for _, path := range []string{"/nonexistent/path/xyz", t.TempDir(), filePath} {
		got, err := svc.ValidateLocalPath(t.Context(), path)
		require.NoError(t, err)
		assert.False(t, got.Valid)
		require.NotNil(t, got.Error)
		assert.Equal(t, localPathInvalidMessage, *got.Error, "failure modes are indistinguishable")
	}
}

func TestValidateRepository(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	got, err := env.svc.ValidateRepository(t.Context(), "https://example.com/x.git", nil)
	require.NoError(t, err)
	assert.True(t, got.Valid)
}

func TestListRevisions_UnknownSource(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	_, err := env.svc.ListRevisions(t.Context(), "missing")
	require.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
}

func TestListTypes_MarksDependenciesAndKeepsComments(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src, _ := env.createLocal(t, map[string]string{"shop.proto": `syntax = "proto3";
package shop;
import "google/protobuf/timestamp.proto";
// A placed order.
message Order {
  google.protobuf.Timestamp at = 1; // When it was placed.
}
`})
	_, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)

	types, err := env.svc.ListTypes(t.Context(), src.Id)
	require.NoError(t, err)
	byName := map[string]entities.SchemaType{}
	for _, ty := range types {
		byName[ty.FullName] = ty
	}
	assert.False(t, byName["shop.Order"].Dependency)
	assert.Equal(t, "A placed order.", byName["shop.Order"].Comment)
	assert.Equal(t, LocalRevision, byName["shop.Order"].SourceRevision)
	assert.True(t, byName["google.protobuf.Timestamp"].Dependency)

	desc, err := env.svc.DescribeType(t.Context(), src.Id, "shop.Order", true)
	require.NoError(t, err)
	assert.Equal(t, "When it was placed.", desc.Messages[0].Fields[0].Comment)
	assert.Equal(t, "google.protobuf.Timestamp", desc.Messages[1].FullName)

	_, err = env.svc.DescribeType(t.Context(), src.Id, "shop.Missing", false)
	require.ErrorIs(t, err, errs.ErrProtoTypeNotFound)
	_, err = env.svc.ListTypes(t.Context(), "missing")
	require.ErrorIs(t, err, errs.ErrMappingSourceNotFound)
}
