// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	fwsvc "github.com/dmit-4884/natscope/internal/services/filewatcher"
	gitfetchersvc "github.com/dmit-4884/natscope/internal/services/gitfetcher"
	descriptorsstorage "github.com/dmit-4884/natscope/internal/storages/proto/descriptors"
	sourcesstorage "github.com/dmit-4884/natscope/internal/storages/proto/sources"
	versionsstorage "github.com/dmit-4884/natscope/internal/storages/proto/versions"
)

// --- Mocks ---

type mockSourcesStorage struct {
	saveErr    error
	getResult  *entities.ProtoSource
	getErr     error
	listResult *entities.List[entities.ProtoSources]
	listErr    error
	updateErr  error
	deleteErr  error

	saveCalled   bool
	saveInput    *entities.ProtoSource
	getCalled    bool
	getID        string
	updateCalled bool
	updateInput  *entities.ProtoSource
	deleteCalled bool
	deleteInput  *entities.SoftDelete
}

func (m *mockSourcesStorage) Save(_ context.Context, in *entities.ProtoSource) error {
	m.saveCalled = true
	m.saveInput = in
	return m.saveErr
}
func (m *mockSourcesStorage) Get(_ context.Context, id string, _ ...bool) (*entities.ProtoSource, error) {
	m.getCalled = true
	m.getID = id
	return m.getResult, m.getErr
}
func (m *mockSourcesStorage) List(_ context.Context, _ *entities.ProtoSourcesList) (*entities.List[entities.ProtoSources], error) {
	return m.listResult, m.listErr
}
func (m *mockSourcesStorage) Update(_ context.Context, in *entities.ProtoSource) error {
	m.updateCalled = true
	m.updateInput = in
	return m.updateErr
}
func (m *mockSourcesStorage) SoftDelete(_ context.Context, in *entities.SoftDelete) error {
	m.deleteCalled = true
	m.deleteInput = in
	return m.deleteErr
}

// Stubs for the rest of the storage interface — not exercised by the proto
// service today, but required so the mock satisfies sources.Storage.
func (m *mockSourcesStorage) GetByName(_ context.Context, _ string, _ ...bool) (*entities.ProtoSource, error) {
	return nil, nil
}
func (m *mockSourcesStorage) Exists(_ context.Context, _ string, _ ...bool) (bool, error) {
	return false, nil
}

type mockVersionsStorage struct {
	saveErr           error
	getResult         *entities.ProtoVersion
	getErr            error
	getBySourceTagErr error
	getBySourceTag    *entities.ProtoVersion

	saveCalled bool
}

func (m *mockVersionsStorage) Save(_ context.Context, _ *entities.ProtoVersion) error {
	m.saveCalled = true
	return m.saveErr
}
func (m *mockVersionsStorage) Get(_ context.Context, _ string) (*entities.ProtoVersion, error) {
	return m.getResult, m.getErr
}
func (m *mockVersionsStorage) GetBySourceAndTag(_ context.Context, _, _ string) (*entities.ProtoVersion, error) {
	return m.getBySourceTag, m.getBySourceTagErr
}

// Stubs for the rest of the storage interface.
func (m *mockVersionsStorage) List(_ context.Context, _ *entities.ProtoVersionsList) (*entities.List[entities.ProtoVersions], error) {
	return nil, nil
}
func (m *mockVersionsStorage) GetTagsBySource(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockVersionsStorage) Exists(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockVersionsStorage) ExistsBySourceAndTag(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockVersionsStorage) Delete(_ context.Context, _ string) error { return nil }
func (m *mockVersionsStorage) DeleteBySource(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type mockDescriptorsStorage struct {
	saveErr           error
	getBySourceTag    *entities.ProtoDescriptor
	getBySourceTagErr error
	getAll            entities.ProtoDescriptors
	getAllErr         error
	updateErr         error

	saveCalled bool
	saveInput  *entities.ProtoDescriptor
}

func (m *mockDescriptorsStorage) Save(_ context.Context, in *entities.ProtoDescriptor) error {
	m.saveCalled = true
	m.saveInput = in
	return m.saveErr
}
func (m *mockDescriptorsStorage) GetBySourceTag(_ context.Context, _, _ string) (*entities.ProtoDescriptor, error) {
	return m.getBySourceTag, m.getBySourceTagErr
}
func (m *mockDescriptorsStorage) FindByFingerprint(_ context.Context, _ string) (*entities.ProtoDescriptor, error) {
	return m.getBySourceTag, m.getBySourceTagErr
}
func (m *mockDescriptorsStorage) GetAll(_ context.Context) (entities.ProtoDescriptors, error) {
	return m.getAll, m.getAllErr
}
func (m *mockDescriptorsStorage) Update(_ context.Context, _ *entities.ProtoDescriptor) error {
	return m.updateErr
}

// Stubs for the rest of the storage interface.
func (m *mockDescriptorsStorage) GetById(_ context.Context, _ string) (*entities.ProtoDescriptor, error) {
	return nil, nil
}
func (m *mockDescriptorsStorage) List(_ context.Context, _ *entities.ProtoDescriptorsList) (*entities.List[entities.ProtoDescriptors], error) {
	return nil, nil
}
func (m *mockDescriptorsStorage) Delete(_ context.Context, _ string) error { return nil }
func (m *mockDescriptorsStorage) DeleteBySource(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockDescriptorsStorage) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type mockGitFetcher struct {
	validateErr error
	listTags    []string
	listTagsErr error
	fetchFiles  []entities.ProtoFileEntry
	fetchErr    error
}

func (m *mockGitFetcher) ValidateRepository(_ context.Context, _ string) error {
	return m.validateErr
}
func (m *mockGitFetcher) ListTags(_ context.Context, _ string) ([]string, error) {
	return m.listTags, m.listTagsErr
}
func (m *mockGitFetcher) FetchVersion(_ context.Context, _, _ string) (*gitfetchersvc.FetchResult, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return &gitfetchersvc.FetchResult{Files: m.fetchFiles}, nil
}

type mockFileWatcher struct {
	watchCalled   bool
	watchSourceID string
	watchDirPath  string
	watchErr      error

	unwatchCalled   bool
	unwatchSourceID string

	setCallbackCalled bool
	callback          fwsvc.ChangeCallback
}

func (m *mockFileWatcher) Watch(sourceID string, dirPath string) error {
	m.watchCalled = true
	m.watchSourceID = sourceID
	m.watchDirPath = dirPath
	return m.watchErr
}
func (m *mockFileWatcher) Unwatch(sourceID string) {
	m.unwatchCalled = true
	m.unwatchSourceID = sourceID
}
func (m *mockFileWatcher) SetCallback(cb fwsvc.ChangeCallback) {
	m.setCallbackCalled = true
	m.callback = cb
}

// Stubs for the rest of the filewatcher.Service interface.
func (m *mockFileWatcher) Start() error { return nil }
func (m *mockFileWatcher) Stop()        {}

// --- Helpers ---

func ptrStr(s string) *string { return &s }
func ptrBool(b bool) *bool    { return &b }

func newTestService(
	sources *mockSourcesStorage,
	versions *mockVersionsStorage,
	descriptors *mockDescriptorsStorage,
	gitFetcher *mockGitFetcher,
	fileWatcher *mockFileWatcher,
) *Service {
	var ss sourcesstorage.Storage
	if sources != nil {
		ss = sources
	}
	var vs versionsstorage.Storage
	if versions != nil {
		vs = versions
	}
	var ds descriptorsstorage.Storage
	if descriptors != nil {
		ds = descriptors
	}
	var gf gitfetchersvc.Service
	if gitFetcher != nil {
		gf = gitFetcher
	}
	var fw fwsvc.Service
	if fileWatcher != nil {
		fw = fileWatcher
	}
	return New(ss, vs, ds, nil /* selections */, nil /* conflicts */, gf, nil /* mappings */, fw)
}

// --- Tests ---

func TestCreateSource(t *testing.T) {
	t.Parallel()

	t.Run("Git_Success", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name:       "core-proto",
			SourceType: entities.SourceTypeGit,
			Repository: "https://gitlab.com/org/proto.git",
			Token:      ptrStr("my-token"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Id)
		assert.Equal(t, "core-proto", result.Name)
		assert.Equal(t, entities.SourceTypeGit, result.SourceType)
		assert.True(t, result.Enabled)
		assert.True(t, store.saveCalled)
	})

	t.Run("Local_WithWatcher", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{}
		fw := &mockFileWatcher{}
		svc := newTestService(store, nil, nil, nil, fw)

		localPath := "/tmp/protos"
		result, err := svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name:           "local-proto",
			SourceType:     entities.SourceTypeLocal,
			LocalPath:      ptrStr(localPath),
			WatcherEnabled: ptrBool(true),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, entities.SourceTypeLocal, result.SourceType)
		assert.True(t, fw.watchCalled, "fileWatcher.Watch should be called")
		assert.Equal(t, result.Id, fw.watchSourceID)
		assert.Equal(t, localPath, fw.watchDirPath)
	})

	t.Run("Local_WithoutWatcher", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{}
		fw := &mockFileWatcher{}
		svc := newTestService(store, nil, nil, nil, fw)

		result, err := svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name:       "local-proto",
			SourceType: entities.SourceTypeLocal,
			LocalPath:  ptrStr("/tmp/protos"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, fw.watchCalled, "fileWatcher.Watch should NOT be called when WatcherEnabled is nil/false")
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{saveErr: errors.New("db error")}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name:       "fail-proto",
			SourceType: entities.SourceTypeGit,
		})

		require.Error(t, err)
		assert.Nil(t, result)
	})

}

func TestUpdateSource(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.Name = "old-name"
			s.Repository = "https://old.git"

		})
		store := &mockSourcesStorage{getResult: existing}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.UpdateSource(t.Context(), &entities.ProtoSourceUpdate{
			Id:   existing.Id,
			Name: ptrStr("new-name"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "new-name", result.Name)
		assert.True(t, store.updateCalled)
	})

	t.Run("TokenRemoval_EmptyString", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.Token = ptrStr("old-token")

		})
		store := &mockSourcesStorage{getResult: existing}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.UpdateSource(t.Context(), &entities.ProtoSourceUpdate{
			Id:    existing.Id,
			Token: ptrStr(""),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.Token, "token should be nil after sending empty string")
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{getErr: errors.New("not found")}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.UpdateSource(t.Context(), &entities.ProtoSourceUpdate{
			Id: "nonexistent",
		})

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("UpdateStorageError", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew()
		store := &mockSourcesStorage{
			getResult: existing,
			updateErr: errors.New("update failed"),
		}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.UpdateSource(t.Context(), &entities.ProtoSourceUpdate{
			Id:   existing.Id,
			Name: ptrStr("new"),
		})

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDeleteSource(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{}
		fw := &mockFileWatcher{}
		svc := newTestService(store, nil, nil, nil, fw)

		err := svc.DeleteSource(t.Context(), "source-123")

		require.NoError(t, err)
		assert.True(t, store.deleteCalled)
		assert.True(t, fw.unwatchCalled, "fileWatcher.Unwatch should be called")
		assert.Equal(t, "source-123", fw.unwatchSourceID)
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{deleteErr: errors.New("not found")}
		svc := newTestService(store, nil, nil, nil, nil)

		err := svc.DeleteSource(t.Context(), "nonexistent")

		require.Error(t, err)
	})

}

func TestSetEnabled(t *testing.T) {
	t.Parallel()

	t.Run("Enable", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.Enabled = false

		})
		store := &mockSourcesStorage{getResult: existing}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.SetEnabled(t.Context(), existing.Id, true)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Enabled)
		assert.True(t, store.updateCalled)
	})

	t.Run("Disable", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.Enabled = true

		})
		store := &mockSourcesStorage{getResult: existing}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.SetEnabled(t.Context(), existing.Id, false)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Enabled)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{getErr: errors.New("not found")}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.SetEnabled(t.Context(), "nonexistent", true)

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("UpdateError", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {

		})
		store := &mockSourcesStorage{
			getResult: existing,
			updateErr: errors.New("update failed"),
		}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.SetEnabled(t.Context(), existing.Id, true)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestSetWatcher(t *testing.T) {
	t.Parallel()

	t.Run("Enable_LocalSource", func(t *testing.T) {
		t.Parallel()
		localPath := "/tmp/protos"
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeLocal
			s.LocalPath = &localPath
			s.WatcherEnabled = false
		})
		store := &mockSourcesStorage{getResult: existing}
		fw := &mockFileWatcher{}
		svc := newTestService(store, nil, nil, nil, fw)

		result, err := svc.SetWatcher(t.Context(), existing.Id, true)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.WatcherEnabled)
		assert.True(t, fw.watchCalled, "fileWatcher.Watch should be called")
		assert.Equal(t, existing.Id, fw.watchSourceID)
		assert.Equal(t, localPath, fw.watchDirPath)
	})

	t.Run("Disable_LocalSource", func(t *testing.T) {
		t.Parallel()
		localPath := "/tmp/protos"
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeLocal
			s.LocalPath = &localPath
			s.WatcherEnabled = true
		})
		store := &mockSourcesStorage{getResult: existing}
		fw := &mockFileWatcher{}
		svc := newTestService(store, nil, nil, nil, fw)

		result, err := svc.SetWatcher(t.Context(), existing.Id, false)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.WatcherEnabled)
		assert.True(t, fw.unwatchCalled, "fileWatcher.Unwatch should be called")
		assert.Equal(t, existing.Id, fw.unwatchSourceID)
	})

	t.Run("Error_NonLocalSource", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeGit
		})
		store := &mockSourcesStorage{getResult: existing}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.SetWatcher(t.Context(), existing.Id, true)

		require.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidRequest)
		assert.Nil(t, result)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{getErr: errors.New("not found")}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.SetWatcher(t.Context(), "nonexistent", true)

		require.Error(t, err)
		assert.Nil(t, result)
	})

}

func TestValidateLocalPath(t *testing.T) {
	t.Parallel()

	t.Run("ValidDirectory_WithProtoFiles", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "test.proto"), []byte(`syntax = "proto3";`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "other.proto"), []byte(`syntax = "proto3";`), 0o644))

		svc := newTestService(nil, nil, nil, nil, nil)

		result, err := svc.ValidateLocalPath(t.Context(), dir)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Valid)
		assert.Equal(t, 2, result.ProtoFileCount)
	})

	t.Run("NonExistentDirectory", func(t *testing.T) {
		t.Parallel()
		svc := newTestService(nil, nil, nil, nil, nil)

		result, err := svc.ValidateLocalPath(t.Context(), "/nonexistent/path/xyz")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Valid)
		require.NotNil(t, result.Error)
		assert.Contains(t, *result.Error, "not accessible")
	})

	t.Run("EmptyDirectory_NoProtoFiles", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		svc := newTestService(nil, nil, nil, nil, nil)

		result, err := svc.ValidateLocalPath(t.Context(), dir)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Valid)
		require.NotNil(t, result.Error)
		assert.Contains(t, *result.Error, "no .proto files")
	})

	t.Run("FileInsteadOfDirectory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "not-a-dir.proto")
		require.NoError(t, os.WriteFile(filePath, []byte(`syntax = "proto3";`), 0o644))

		svc := newTestService(nil, nil, nil, nil, nil)

		result, err := svc.ValidateLocalPath(t.Context(), filePath)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Valid)
		require.NotNil(t, result.Error)
		assert.Contains(t, *result.Error, "not a directory")
	})

	t.Run("NestedProtoFiles", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		subDir := filepath.Join(dir, "subpkg")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "root.proto"), []byte(`syntax = "proto3";`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.proto"), []byte(`syntax = "proto3";`), 0o644))
		// Non-proto files should be ignored
		require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0o644))

		svc := newTestService(nil, nil, nil, nil, nil)

		result, err := svc.ValidateLocalPath(t.Context(), dir)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Valid)
		assert.Equal(t, 2, result.ProtoFileCount)
	})
}

func TestCompileLocal(t *testing.T) {
	t.Parallel()

	t.Run("Error_NonLocalSource", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeGit
		})
		store := &mockSourcesStorage{getResult: existing}
		desc := &mockDescriptorsStorage{}
		svc := newTestService(store, nil, desc, nil, nil)

		ctx := t.Context()
		result, _, err := svc.CompileLocal(ctx, existing.Id)

		require.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidRequest)
		assert.Nil(t, result)
	})

	t.Run("Error_LocalPathNotSet", func(t *testing.T) {
		t.Parallel()
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeLocal
			s.LocalPath = nil
		})
		store := &mockSourcesStorage{getResult: existing}
		desc := &mockDescriptorsStorage{}
		svc := newTestService(store, nil, desc, nil, nil)

		ctx := t.Context()
		result, _, err := svc.CompileLocal(ctx, existing.Id)

		require.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidRequest)
		assert.Nil(t, result)
	})

	t.Run("Success_WithRealProtoFiles", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		protoContent := `syntax = "proto3";
package test;
message TestMessage {
  string name = 1;
  int32 value = 2;
}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "test.proto"), []byte(protoContent), 0o644))

		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeLocal
			s.LocalPath = ptrStr(dir)
		})
		store := &mockSourcesStorage{getResult: existing}
		desc := &mockDescriptorsStorage{}
		svc := newTestService(store, nil, desc, nil, nil)

		ctx := t.Context()
		result, _, err := svc.CompileLocal(ctx, existing.Id)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.MessageTypes)
		assert.Equal(t, 1, result.FileDescriptors)
		assert.True(t, desc.saveCalled, "descriptor should be saved")
		assert.Equal(t, "local", desc.saveInput.Tag)

	})
}

func errorDiagsEnt(diags []entities.CompileDiagnostic) []entities.CompileDiagnostic {
	var out []entities.CompileDiagnostic
	for _, d := range diags {
		if d.Severity == entities.DiagnosticError {
			out = append(out, d)
		}
	}
	return out
}

func TestCompileLocalDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("broken import returns diagnostics not opaque error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.proto"),
			[]byte("syntax = \"proto3\";\nimport \"missing.proto\";\nmessage A {}\n"), 0o644))
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeLocal
			s.LocalPath = ptrStr(dir)
		})
		store := &mockSourcesStorage{getResult: existing}
		desc := &mockDescriptorsStorage{}
		svc := newTestService(store, nil, desc, nil, nil)

		result, diags, err := svc.CompileLocal(t.Context(), existing.Id)
		require.NoError(t, err, "compile errors are diagnostics, not a transport error")
		assert.Nil(t, result)
		require.NotEmpty(t, diags)
		var found bool
		for _, d := range diags {
			if d.MissingImport == "missing.proto" {
				found = true
				assert.NotEmpty(t, d.Hint)
			}
		}
		assert.True(t, found, "missing-import diagnostic with hint expected")
	})

	t.Run("success returns result plus roots persisted in telemetry", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "proto/common"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "proto/api"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "proto/common/t.proto"),
			[]byte("syntax = \"proto3\";\npackage c;\nmessage T {}\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "proto/api/s.proto"),
			[]byte("syntax = \"proto3\";\npackage a;\nimport \"common/t.proto\";\nmessage S { c.T t = 1; }\n"), 0o644))
		existing := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeLocal
			s.LocalPath = ptrStr(dir)
		})
		store := &mockSourcesStorage{getResult: existing}
		desc := &mockDescriptorsStorage{}
		svc := newTestService(store, nil, desc, nil, nil)

		result, diags, err := svc.CompileLocal(t.Context(), existing.Id)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, errorDiagsEnt(diags))
		require.NotNil(t, store.updateInput)
		require.NotNil(t, store.updateInput.LastCompile)
		assert.Equal(t, []string{"proto"}, store.updateInput.LastCompile.Roots)
		assert.Equal(t, "inferred", store.updateInput.LastCompile.RootsOrigin)
	})
}

func TestValidateLocalPathHonest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.proto"), []byte("syntax = \"proto3\";"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "hidden.proto"), []byte("x"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "dep"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "dep", "x.proto"), []byte("x"), 0o644))

	svc := newTestService(nil, nil, nil, nil, nil)
	v, err := svc.ValidateLocalPath(t.Context(), dir)
	require.NoError(t, err)
	assert.True(t, v.Valid)
	assert.Equal(t, 1, v.ProtoFileCount, ".git and node_modules must not be counted")
}

func TestFetchAndCompile(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsCachedDescriptor", func(t *testing.T) {
		t.Parallel()
		cachedDesc := entities.ProtoDescriptorNew(func(d *entities.ProtoDescriptor) {
			d.SourceID = "source-1"
			d.Tag = "v1.0.0"
			d.MessageTypes = []string{"test.Message"}
		})
		desc := &mockDescriptorsStorage{getBySourceTag: cachedDesc}
		svc := newTestService(nil, nil, desc, nil, nil)

		ctx := t.Context()
		result, err := svc.FetchAndCompile(ctx, "source-1", "v1.0.0")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, cachedDesc.Id, result.Id)
		assert.Equal(t, []string{"test.Message"}, result.MessageTypes)
	})

	t.Run("FetchesFromGit_WhenNotCached", func(t *testing.T) {
		t.Parallel()
		source := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeGit
			s.Repository = "https://example.com/proto.git"
		})
		store := &mockSourcesStorage{getResult: source}
		versions := &mockVersionsStorage{
			getBySourceTagErr: errors.New("not found"),
		}
		desc := &mockDescriptorsStorage{
			getBySourceTagErr: errors.New("not found"),
		}
		gitFetcher := &mockGitFetcher{
			fetchFiles: []entities.ProtoFileEntry{
				{
					Path: "test.proto",
					Content: `syntax = "proto3";
package test;
message GitMessage {
  string id = 1;
}`,
					Size: 80,
				},
			},
		}
		svc := newTestService(store, versions, desc, gitFetcher, nil)

		ctx := t.Context()
		result, err := svc.FetchAndCompile(ctx, source.Id, "v1.0.0")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "v1.0.0", result.Tag)
		assert.Contains(t, result.MessageTypes, "test.GitMessage")
		assert.True(t, desc.saveCalled, "descriptor should be saved")
		assert.True(t, versions.saveCalled, "version should be saved")
	})
}

func TestGetSource(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		source := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.Name = "test-source"
		})
		store := &mockSourcesStorage{getResult: source}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.GetSource(t.Context(), source.Id)

		require.NoError(t, err)
		assert.Equal(t, "test-source", result.Name)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{getErr: errors.New("not found")}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.GetSource(t.Context(), "nonexistent")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestListSources(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		expected := &entities.List[entities.ProtoSources]{
			Items: entities.ProtoSources{
				{BaseEntity: entities.BaseEntity{Id: "1"}, Name: "source1"},
			},
		}
		store := &mockSourcesStorage{listResult: expected}
		svc := newTestService(store, nil, nil, nil, nil)

		result, err := svc.ListSources(t.Context(), &entities.ProtoSourcesList{})

		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
	})

}

func TestValidateRepository(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		gf := &mockGitFetcher{}
		svc := newTestService(nil, nil, nil, gf, nil)

		result, err := svc.ValidateRepository(t.Context(), "https://example.com/proto.git", nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Valid)
		assert.Nil(t, result.Error)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		gf := &mockGitFetcher{validateErr: errors.New("auth failed")}
		svc := newTestService(nil, nil, nil, gf, nil)

		result, err := svc.ValidateRepository(t.Context(), "https://example.com/proto.git", ptrStr("bad-token"))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Valid)
		require.NotNil(t, result.Error)
		assert.Contains(t, *result.Error, "auth failed")
	})
}

func TestListTags(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		source := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SourceType = entities.SourceTypeGit
			s.Repository = "https://example.com/proto.git"
		})
		store := &mockSourcesStorage{getResult: source}
		gf := &mockGitFetcher{listTags: []string{"v1.0.0", "v2.0.0"}}
		svc := newTestService(store, nil, nil, gf, nil)

		tags, err := svc.ListTags(t.Context(), source.Id)

		require.NoError(t, err)
		assert.Equal(t, []string{"v1.0.0", "v2.0.0"}, tags)
	})

	t.Run("SourceNotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockSourcesStorage{getErr: errors.New("not found")}
		gf := &mockGitFetcher{}
		svc := newTestService(store, nil, nil, gf, nil)

		tags, err := svc.ListTags(t.Context(), "nonexistent")

		require.Error(t, err)
		assert.Nil(t, tags)
	})
}

func TestFetchVersion(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsCachedVersion", func(t *testing.T) {
		t.Parallel()
		cachedVersion := entities.ProtoVersionNew(func(v *entities.ProtoVersion) {
			v.SourceID = "source-1"
			v.Tag = "v1.0.0"
		})
		versions := &mockVersionsStorage{getBySourceTag: cachedVersion}
		svc := newTestService(nil, versions, nil, nil, nil)

		result, err := svc.FetchVersion(t.Context(), "source-1", "v1.0.0")

		require.NoError(t, err)
		assert.Equal(t, cachedVersion.Id, result.Id)
		assert.False(t, versions.saveCalled, "should not save when returning cached")
	})

	t.Run("FetchesFromGit_WhenNotCached", func(t *testing.T) {
		t.Parallel()
		source := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.Repository = "https://example.com/proto.git"
		})
		store := &mockSourcesStorage{getResult: source}
		versions := &mockVersionsStorage{getBySourceTagErr: errors.New("not found")}
		gf := &mockGitFetcher{
			fetchFiles: []entities.ProtoFileEntry{
				{Path: "test.proto", Content: "syntax = \"proto3\";", Size: 18},
			},
		}
		svc := newTestService(store, versions, nil, gf, nil)

		result, err := svc.FetchVersion(t.Context(), source.Id, "v1.0.0")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "v1.0.0", result.Tag)
		assert.Len(t, result.Files, 1)
		assert.True(t, versions.saveCalled, "version should be saved")
	})
}
