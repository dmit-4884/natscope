// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// --- Mocks ---

// mockProtoService embeds proto.SourceManager and overrides only the methods
// the sources handler calls.
type mockProtoService struct {
	protosvc.SourceManager
	createResult  *entities.ProtoSource
	createErr     error
	getResult     *entities.ProtoSource
	getErr        error
	listResult    *entities.List[entities.ProtoSources]
	listErr       error
	updateResult  *entities.ProtoSource
	updateErr     error
	deleteErr     error
	repoResult    *entities.RepositoryValidation
	repoErr       error
	localResult   *entities.LocalPathValidation
	localErr      error
	refsResult    []entities.ProtoRef
	refsErr       error
	refSource     *entities.ProtoSource
	refOutcome    *entities.CompileOutcome
	refErr        error
	revisions     []entities.SchemaRevision
	revisionsErr  error
	enabledResult *entities.ProtoSource
	enabledErr    error
	watcherResult *entities.ProtoSource
	watcherErr    error
	gotUpload     entities.SchemaUpload
	gotSourceType entities.SourceType
	uploadOutcome *entities.CompileOutcome
	uploadErr     error
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

func (m *mockProtoService) ValidateRepository(
	_ context.Context,
	sourceType entities.SourceType,
	_ string,
	_ *string,
) (*entities.RepositoryValidation, error) {
	m.gotSourceType = sourceType
	return m.repoResult, m.repoErr
}

func (m *mockProtoService) ValidateLocalPath(_ context.Context, _ string) (*entities.LocalPathValidation, error) {
	return m.localResult, m.localErr
}

func (m *mockProtoService) ListRefs(_ context.Context, _ string) ([]entities.ProtoRef, error) {
	return m.refsResult, m.refsErr
}

func (m *mockProtoService) SelectRef(_ context.Context, _, _ string) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	return m.refSource, m.refOutcome, m.refErr
}

func (m *mockProtoService) RefreshSource(_ context.Context, _ string) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	return m.refSource, m.refOutcome, m.refErr
}

func (m *mockProtoService) ListRevisions(_ context.Context, _ string) ([]entities.SchemaRevision, error) {
	return m.revisions, m.revisionsErr
}

func (m *mockProtoService) SetEnabled(_ context.Context, _ string, _ bool) (*entities.ProtoSource, error) {
	return m.enabledResult, m.enabledErr
}

func (m *mockProtoService) SetWatcher(_ context.Context, _ string, _ bool) (*entities.ProtoSource, error) {
	return m.watcherResult, m.watcherErr
}

func (m *mockProtoService) UploadSchema(
	_ context.Context,
	_ string,
	upload entities.SchemaUpload,
) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	m.gotUpload = upload
	return entities.ProtoSourceNew(), m.uploadOutcome, m.uploadErr
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

		resp, err := handler.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{
			Name:       "my-src",
			SourceType: protopb.SourceType_SOURCE_TYPE_GIT,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Source)
	})

	t.Run("NameAlreadyInUse", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{createErr: errs.ErrProtoSourceNameAlreadyInUse}
		handler := New(svc)

		_, err := handler.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{
			Name:       "dup",
			SourceType: protopb.SourceType_SOURCE_TYPE_GIT,
		}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNameAlreadyInUse)
	})

	t.Run("UnknownSourceType", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{}
		handler := New(svc)

		_, err := handler.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{
			Name:       "my-src",
			SourceType: protopb.SourceType(99),
		}))
		assert.ErrorIs(t, err, errs.ErrInvalidRequest)
	})

	t.Run("UnspecifiedSourceType", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{}
		handler := New(svc)

		_, err := handler.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{Name: "my-src"}))
		assert.ErrorIs(t, err, errs.ErrInvalidRequest)
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
		assert.Equal(t, entities.SourceTypeGit, svc.gotSourceType)
	})

	t.Run("BSR module", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{repoResult: &entities.RepositoryValidation{Valid: true}}

		_, err := New(svc).ValidateRepository(t.Context(), connect.NewRequest(&sourcespb.ValidateRepositoryRequest{
			Repository: "buf.build/acme/payments", SourceType: protopb.SourceType_SOURCE_TYPE_BSR,
		}))
		require.NoError(t, err)
		assert.Equal(t, entities.SourceTypeBSR, svc.gotSourceType)
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

func TestHandler_ListRefs(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{refsResult: []entities.ProtoRef{
			{Name: "v1.0.0", Kind: entities.RefKindTag, Revision: "aaa"},
			{Name: "main", Kind: entities.RefKindBranch, Revision: "bbb"},
		}}
		resp, err := New(svc).ListRefs(t.Context(), connect.NewRequest(&sourcespb.ListRefsRequest{SourceId: "src-1"}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Refs, 2)
		assert.Equal(t, protopb.RefKind_REF_KIND_TAG, resp.Msg.Refs[0].Kind)
		assert.Equal(t, protopb.RefKind_REF_KIND_BRANCH, resp.Msg.Refs[1].Kind)
		assert.Equal(t, "bbb", resp.Msg.Refs[1].Revision)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{refsErr: errs.ErrProtoSourceNotFound}
		_, err := New(svc).ListRefs(t.Context(), connect.NewRequest(&sourcespb.ListRefsRequest{SourceId: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}

func TestHandler_SelectRef(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		src := entities.ProtoSourceNew(func(s *entities.ProtoSource) {
			s.SelectedRef = &entities.ProtoRef{Name: "main", Kind: entities.RefKindBranch, Revision: "abc"}
			s.ActiveSchema = &entities.SchemaRevision{Revision: "abc", Fingerprint: "fp", MessageCount: 3, Active: true}
		})
		svc := &mockProtoService{refSource: src, refOutcome: &entities.CompileOutcome{Valid: true, MessageTypes: 3, FileDescriptors: 2}}
		resp, err := New(svc).SelectRef(t.Context(), connect.NewRequest(&sourcespb.SelectRefRequest{SourceId: "src-1", Ref: "main"}))
		require.NoError(t, err)
		assert.Equal(t, protopb.RefKind_REF_KIND_BRANCH, resp.Msg.Source.SelectedRef.Kind)
		assert.Equal(t, "fp", resp.Msg.Source.ActiveSchema.Fingerprint)
		assert.True(t, resp.Msg.Outcome.Valid)
		assert.Equal(t, int32(2), resp.Msg.Outcome.FileDescriptors)
	})

	t.Run("CompileErrorsComeBackAsOutcome", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{
			refSource:  entities.ProtoSourceNew(),
			refOutcome: &entities.CompileOutcome{Diagnostics: []entities.CompileDiagnostic{{Severity: entities.DiagnosticError, Message: "boom"}}},
		}
		resp, err := New(svc).SelectRef(t.Context(), connect.NewRequest(&sourcespb.SelectRefRequest{SourceId: "src-1", Ref: "v1"}))
		require.NoError(t, err)
		assert.False(t, resp.Msg.Outcome.Valid)
		require.Len(t, resp.Msg.Outcome.Diagnostics, 1)
		assert.Equal(t, "boom", resp.Msg.Outcome.Diagnostics[0].Message)
	})

	t.Run("RefNotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{refErr: errs.ErrProtoRefNotFound}
		_, err := New(svc).SelectRef(t.Context(), connect.NewRequest(&sourcespb.SelectRefRequest{SourceId: "src-1", Ref: "nope"}))
		assert.ErrorIs(t, err, errs.ErrProtoRefNotFound)
	})
}

func TestHandler_RefreshSource(t *testing.T) {
	t.Parallel()

	svc := &mockProtoService{refSource: entities.ProtoSourceNew(), refOutcome: &entities.CompileOutcome{Valid: true, MessageTypes: 6}}
	resp, err := New(svc).RefreshSource(t.Context(), connect.NewRequest(&sourcespb.RefreshSourceRequest{SourceId: "src-1"}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Outcome.Valid)
	assert.Equal(t, int32(6), resp.Msg.Outcome.MessageTypes)
}

func TestHandler_ListRevisions(t *testing.T) {
	t.Parallel()

	svc := &mockProtoService{revisions: []entities.SchemaRevision{
		{Revision: "b", Fingerprint: "fp-b", CompiledAt: 2, MessageCount: 4, Active: true},
		{Revision: "a", Fingerprint: "fp-a", CompiledAt: 1, MessageCount: 3},
	}}
	resp, err := New(svc).ListRevisions(t.Context(), connect.NewRequest(&sourcespb.ListRevisionsRequest{SourceId: "src-1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Revisions, 2)
	assert.Equal(t, "fp-b", resp.Msg.Revisions[0].Fingerprint)
	assert.True(t, resp.Msg.Revisions[0].Active)
	assert.Equal(t, int64(1), resp.Msg.Revisions[1].CompiledAt)
	assert.Equal(t, int32(3), resp.Msg.Revisions[1].MessageCount)
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

func TestHandler_UploadSchema(t *testing.T) {
	t.Parallel()

	t.Run("files", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{uploadOutcome: &entities.CompileOutcome{Valid: true, MessageTypes: 4, FileDescriptors: 1}}

		resp, err := New(svc).UploadSchema(t.Context(), connect.NewRequest(&sourcespb.UploadSchemaRequest{
			SourceId: "src-1",
			Content: &sourcespb.UploadSchemaRequest_Files{Files: &sourcespb.UploadedFiles{Files: []*sourcespb.UploadedFile{
				{Path: "shop/order.proto", Content: "syntax = \"proto3\";"},
			}}},
		}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Outcome.Valid)
		assert.Equal(t, int32(4), resp.Msg.Outcome.MessageTypes)
		assert.Equal(t, []entities.ProtoFileEntry{{Path: "shop/order.proto", Content: "syntax = \"proto3\";"}}, svc.gotUpload.Files)
		assert.Empty(t, svc.gotUpload.DescriptorSet)
	})

	t.Run("descriptor set", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{uploadOutcome: &entities.CompileOutcome{Valid: true}}

		_, err := New(svc).UploadSchema(t.Context(), connect.NewRequest(&sourcespb.UploadSchemaRequest{
			SourceId: "src-1",
			Content:  &sourcespb.UploadSchemaRequest_DescriptorSet{DescriptorSet: []byte{1, 2}},
		}))
		require.NoError(t, err)
		assert.Equal(t, []byte{1, 2}, svc.gotUpload.DescriptorSet)
		assert.Empty(t, svc.gotUpload.Files)
	})

	t.Run("service error", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{uploadErr: errs.ErrProtoSourceNotFound}

		_, err := New(svc).UploadSchema(t.Context(), connect.NewRequest(&sourcespb.UploadSchemaRequest{SourceId: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSourceNotFound)
	})
}
