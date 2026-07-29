// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
)

// --- Mocks ---

// mockProtoService embeds proto.SourceManager and overrides only the methods
// the sources handler calls.
type mockProtoService struct {
	protosvc.SourceManager
	createResult   *entities.ProtoSource
	createErr      error
	getResult      *entities.ProtoSource
	getErr         error
	listResult     *entities.List[entities.ProtoSources]
	listErr        error
	updateResult   *entities.ProtoSource
	updateErr      error
	deleteErr      error
	repoResult     *entities.RepositoryValidation
	repoErr        error
	localResult    *entities.LocalPathValidation
	localErr       error
	tagsResult     []string
	tagsErr        error
	versionResult  *entities.ProtoVersion
	versionErr     error
	enabledResult  *entities.ProtoSource
	enabledErr     error
	watcherResult  *entities.ProtoSource
	watcherErr     error
	compileResult  *entities.CompileResult
	compileDiags   []entities.CompileDiagnostic
	compileErr     error
	validateResult *entities.CompileResult
	validateDiags  []entities.CompileDiagnostic
	validateErr    error
}

func (m *mockProtoService) CreateSource(_ context.Context, _ *entities.ProtoSourceCreate) (*entities.ProtoSource, error) {
	return m.createResult, m.createErr
}

func (m *mockProtoService) GetSource(_ context.Context, _ string) (*entities.ProtoSource, error) {
	return m.getResult, m.getErr
}

func (m *mockProtoService) ListSources(_ context.Context, _ *entities.ProtoSourcesList) (*entities.List[entities.ProtoSources], error) {
	return m.listResult, m.listErr
}

func (m *mockProtoService) UpdateSource(_ context.Context, _ *entities.ProtoSourceUpdate) (*entities.ProtoSource, error) {
	return m.updateResult, m.updateErr
}

func (m *mockProtoService) DeleteSource(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockProtoService) ValidateRepository(_ context.Context, _ string, _ *string) (*entities.RepositoryValidation, error) {
	return m.repoResult, m.repoErr
}

func (m *mockProtoService) ValidateLocalPath(_ context.Context, _ string) (*entities.LocalPathValidation, error) {
	return m.localResult, m.localErr
}

func (m *mockProtoService) ListTags(_ context.Context, _ string) ([]string, error) {
	return m.tagsResult, m.tagsErr
}

func (m *mockProtoService) FetchVersion(_ context.Context, _, _ string) (*entities.ProtoVersion, error) {
	return m.versionResult, m.versionErr
}

func (m *mockProtoService) SetEnabled(_ context.Context, _ string, _ bool) (*entities.ProtoSource, error) {
	return m.enabledResult, m.enabledErr
}

func (m *mockProtoService) SetWatcher(_ context.Context, _ string, _ bool) (*entities.ProtoSource, error) {
	return m.watcherResult, m.watcherErr
}

func (m *mockProtoService) CompileLocal(_ context.Context, _ string) (*entities.CompileResult, []entities.CompileDiagnostic, error) {
	return m.compileResult, m.compileDiags, m.compileErr
}

func (m *mockProtoService) ValidateFiles(_ context.Context, _ *string, _, _ []string) (*entities.CompileResult, []entities.CompileDiagnostic, error) {
	return m.validateResult, m.validateDiags, m.validateErr
}

func (m *mockProtoService) CompileFiles(_ context.Context, _ string) (*entities.CompileResult, []entities.CompileDiagnostic, error) {
	return m.compileResult, m.compileDiags, m.compileErr
}

// --- Tests ---

func TestHandler_CreateSource(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{createResult: entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.Name = "my-src"
			s.SourceType = entities.SourceTypeGit
		})}
		handler := New(svc)

		resp, err := handler.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{Name: "my-src"}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Source)
	})

	t.Run("NameAlreadyInUse", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{createErr: errs.ErrProtoSourceNameAlreadyInUse}
		handler := New(svc)

		_, err := handler.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{Name: "dup"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNameAlreadyInUse)
	})
}

func TestHandler_GetSource(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{getResult: entities.ProtoSourceNew()}
		handler := New(svc)

		resp, err := handler.GetSource(t.Context(), connect.NewRequest(&sourcespb.GetSourceRequest{Id: "src-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Source)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{getErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.GetSource(t.Context(), connect.NewRequest(&sourcespb.GetSourceRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_ListSources(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		next := "next-cursor"
		svc := &mockProtoService{listResult: &entities.List[entities.ProtoSources]{
			Items:      entities.ProtoSources{entities.ProtoSourceNew()},
			NextCursor: &next,
		}}
		handler := New(svc)

		resp, err := handler.ListSources(t.Context(), connect.NewRequest(&sourcespb.ListSourcesRequest{
			PageSize:  10,
			PageToken: "c1",
		}))
		require.NoError(t, err)
		assert.Len(t, resp.Msg.Sources, 1)
		require.NotNil(t, resp.Msg.NextPageToken)
		assert.Equal(t, "next-cursor", *resp.Msg.NextPageToken)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{listErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.ListSources(t.Context(), connect.NewRequest(&sourcespb.ListSourcesRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_UpdateSource(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{updateResult: entities.ProtoSourceNew()}
		handler := New(svc)

		resp, err := handler.UpdateSource(t.Context(), connect.NewRequest(&sourcespb.UpdateSourceRequest{Id: "src-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Source)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{updateErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.UpdateSource(t.Context(), connect.NewRequest(&sourcespb.UpdateSourceRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_DeleteSource(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{}
		handler := New(svc)

		resp, err := handler.DeleteSource(t.Context(), connect.NewRequest(&sourcespb.DeleteSourceRequest{Id: "src-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{deleteErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.DeleteSource(t.Context(), connect.NewRequest(&sourcespb.DeleteSourceRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_ValidateRepository(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{repoResult: &entities.RepositoryValidation{Valid: true}}
		handler := New(svc)

		resp, err := handler.ValidateRepository(t.Context(), connect.NewRequest(&sourcespb.ValidateRepositoryRequest{
			Repository: "https://example.com/repo.git",
		}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Valid)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{repoErr: errors.New("network error")}
		handler := New(svc)

		_, err := handler.ValidateRepository(t.Context(), connect.NewRequest(&sourcespb.ValidateRepositoryRequest{
			Repository: "bad",
		}))
		assert.Error(t, err)
	})
}

func TestHandler_ValidateLocalPath(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{localResult: &entities.LocalPathValidation{Valid: true, ProtoFileCount: 3}}
		handler := New(svc)

		resp, err := handler.ValidateLocalPath(t.Context(), connect.NewRequest(&sourcespb.ValidateLocalPathRequest{Path: "/tmp/protos"}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Valid)
		assert.Equal(t, int32(3), resp.Msg.ProtoFileCount)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{localErr: errors.New("io error")}
		handler := New(svc)

		_, err := handler.ValidateLocalPath(t.Context(), connect.NewRequest(&sourcespb.ValidateLocalPathRequest{Path: "/bad"}))
		assert.Error(t, err)
	})
}

func TestHandler_ListTags(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{tagsResult: []string{"v1.0.0", "v1.1.0"}}
		handler := New(svc)

		resp, err := handler.ListTags(t.Context(), connect.NewRequest(&sourcespb.ListTagsRequest{SourceId: "src-1"}))
		require.NoError(t, err)
		assert.Equal(t, []string{"v1.0.0", "v1.1.0"}, resp.Msg.Tags)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{tagsErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.ListTags(t.Context(), connect.NewRequest(&sourcespb.ListTagsRequest{SourceId: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_FetchVersion(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{versionResult: entities.ProtoVersionNew()}
		handler := New(svc)

		resp, err := handler.FetchVersion(t.Context(), connect.NewRequest(&sourcespb.FetchVersionRequest{
			SourceId: "src-1",
			Tag:      "v1.0.0",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Version)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{versionErr: errs.ErrProtoVersionNotFound}
		handler := New(svc)

		_, err := handler.FetchVersion(t.Context(), connect.NewRequest(&sourcespb.FetchVersionRequest{
			SourceId: "src-1",
			Tag:      "missing",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoVersionNotFound)
	})
}

func TestHandler_SetEnabled(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{enabledResult: entities.ProtoSourceNew()}
		handler := New(svc)

		resp, err := handler.SetEnabled(t.Context(), connect.NewRequest(&sourcespb.SetEnabledRequest{
			SourceId: "src-1",
			Enabled:  true,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Source)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{enabledErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.SetEnabled(t.Context(), connect.NewRequest(&sourcespb.SetEnabledRequest{SourceId: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_SetWatcher(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{watcherResult: entities.ProtoSourceNew()}
		handler := New(svc)

		resp, err := handler.SetWatcher(t.Context(), connect.NewRequest(&sourcespb.SetWatcherRequest{
			SourceId: "src-1",
			Enabled:  true,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Source)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{watcherErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.SetWatcher(t.Context(), connect.NewRequest(&sourcespb.SetWatcherRequest{SourceId: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_CompileLocal(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{compileResult: &entities.CompileResult{MessageTypes: 5, FileDescriptors: 2}}
		handler := New(svc)

		resp, err := handler.CompileLocal(t.Context(), connect.NewRequest(&sourcespb.CompileLocalRequest{SourceId: "src-1"}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Valid)
		assert.Equal(t, int32(5), resp.Msg.MessageTypes)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{compileErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.CompileLocal(t.Context(), connect.NewRequest(&sourcespb.CompileLocalRequest{SourceId: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_ValidateFiles(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{validateResult: &entities.CompileResult{MessageTypes: 4, FileDescriptors: 1}}
		handler := New(svc)

		resp, err := handler.ValidateFiles(t.Context(), connect.NewRequest(&sourcespb.ValidateFilesRequest{
			Files: []string{"/tmp/a.proto"},
		}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Valid)
		assert.Equal(t, int32(4), resp.Msg.MessageTypes)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{validateErr: errors.New("compile error")}
		handler := New(svc)

		_, err := handler.ValidateFiles(t.Context(), connect.NewRequest(&sourcespb.ValidateFilesRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_CompileFiles(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{compileResult: &entities.CompileResult{MessageTypes: 6, FileDescriptors: 3}}
		handler := New(svc)

		resp, err := handler.CompileFiles(t.Context(), connect.NewRequest(&sourcespb.CompileFilesRequest{SourceId: "src-1"}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Valid)
		assert.Equal(t, int32(6), resp.Msg.MessageTypes)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{compileErr: errs.ErrProtoSourceNotFound}
		handler := New(svc)

		_, err := handler.CompileFiles(t.Context(), connect.NewRequest(&sourcespb.CompileFilesRequest{SourceId: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}
