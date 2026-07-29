// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package templates

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
)

// --- Mocks ---

type mockTemplatesService struct {
	createResult *entities.MessageTemplate
	createErr    error
	getResult    *entities.MessageTemplate
	getErr       error
	updateResult *entities.MessageTemplate
	updateErr    error
	listResult   *entities.List[entities.MessageTemplates]
	listErr      error
	deleteErr    error
	bulkResult   int
	bulkErr      error
	deleteAllN   int64
	deleteAllErr error
}

func (m *mockTemplatesService) Create(_ context.Context, _ *entities.MessageTemplateCreate) (*entities.MessageTemplate, error) {
	return m.createResult, m.createErr
}

func (m *mockTemplatesService) Get(_ context.Context, _ string) (*entities.MessageTemplate, error) {
	return m.getResult, m.getErr
}

func (m *mockTemplatesService) Update(_ context.Context, _ *entities.MessageTemplateUpdate) (*entities.MessageTemplate, error) {
	return m.updateResult, m.updateErr
}

func (m *mockTemplatesService) List(_ context.Context, _ *entities.MessageTemplatesList) (*entities.List[entities.MessageTemplates], error) {
	return m.listResult, m.listErr
}

func (m *mockTemplatesService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockTemplatesService) BulkCreate(_ context.Context, _ []*entities.MessageTemplateCreate) (int, error) {
	return m.bulkResult, m.bulkErr
}

func (m *mockTemplatesService) DeleteAll(_ context.Context) (int64, error) {
	return m.deleteAllN, m.deleteAllErr
}

// --- Tests ---

func TestHandler_CreateTemplate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{createResult: entities.MessageTemplateNew(func(mt *entities.MessageTemplate) {
			mt.Name = "tmpl"
		})}
		handler := New(svc)

		resp, err := handler.CreateTemplate(t.Context(), connect.NewRequest(&templatespb.CreateTemplateRequest{Name: "tmpl"}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Template)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{createErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.CreateTemplate(t.Context(), connect.NewRequest(&templatespb.CreateTemplateRequest{Name: "x"}))
		assert.Error(t, err)
	})
}

func TestHandler_UpdateTemplate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{updateResult: entities.MessageTemplateNew()}
		handler := New(svc)

		name := "updated"
		resp, err := handler.UpdateTemplate(t.Context(), connect.NewRequest(&templatespb.UpdateTemplateRequest{
			Id:        "tmpl-1",
			Name:      &name,
			Headers:   &templatespb.HeadersPatch{Values: map[string]string{"k": "v"}},
			Wildcards: &templatespb.WildcardsPatch{Values: []string{"a"}},
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Template)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{updateErr: errs.ErrMessageTemplateNotFound}
		handler := New(svc)

		_, err := handler.UpdateTemplate(t.Context(), connect.NewRequest(&templatespb.UpdateTemplateRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrMessageTemplateNotFound)
	})
}

func TestHandler_GetTemplate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{getResult: entities.MessageTemplateNew()}
		handler := New(svc)

		resp, err := handler.GetTemplate(t.Context(), connect.NewRequest(&templatespb.GetTemplateRequest{Id: "tmpl-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Template)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{getErr: errs.ErrMessageTemplateNotFound}
		handler := New(svc)

		_, err := handler.GetTemplate(t.Context(), connect.NewRequest(&templatespb.GetTemplateRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrMessageTemplateNotFound)
	})
}

func TestHandler_ListTemplates(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		next := "next"
		svc := &mockTemplatesService{listResult: &entities.List[entities.MessageTemplates]{
			Items:      entities.MessageTemplates{entities.MessageTemplateNew()},
			NextCursor: &next,
		}}
		handler := New(svc)

		resp, err := handler.ListTemplates(t.Context(), connect.NewRequest(&templatespb.ListTemplatesRequest{
			PageSize:  5,
			PageToken: "c1",
		}))
		require.NoError(t, err)
		assert.Len(t, resp.Msg.Templates, 1)
		require.NotNil(t, resp.Msg.NextPageToken)
		assert.Equal(t, "next", *resp.Msg.NextPageToken)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{listErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.ListTemplates(t.Context(), connect.NewRequest(&templatespb.ListTemplatesRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_DeleteTemplate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{}
		handler := New(svc)

		resp, err := handler.DeleteTemplate(t.Context(), connect.NewRequest(&templatespb.DeleteTemplateRequest{Id: "tmpl-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{deleteErr: errs.ErrMessageTemplateNotFound}
		handler := New(svc)

		_, err := handler.DeleteTemplate(t.Context(), connect.NewRequest(&templatespb.DeleteTemplateRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrMessageTemplateNotFound)
	})
}

func TestHandler_BatchCreateTemplates(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{bulkResult: 2}
		handler := New(svc)

		resp, err := handler.BatchCreateTemplates(t.Context(), connect.NewRequest(&templatespb.BatchCreateTemplatesRequest{
			Templates: []*templatespb.TemplateBulkCreateItem{{Name: "a"}, {Name: "b"}},
		}))
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.Msg.Created)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{bulkErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.BatchCreateTemplates(t.Context(), connect.NewRequest(&templatespb.BatchCreateTemplatesRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_DeleteAllTemplates(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{deleteAllN: 3}
		handler := New(svc)

		resp, err := handler.DeleteAllTemplates(t.Context(), connect.NewRequest(&templatespb.DeleteAllTemplatesRequest{}))
		require.NoError(t, err)
		assert.Equal(t, int32(3), resp.Msg.Deleted)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockTemplatesService{deleteAllErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.DeleteAllTemplates(t.Context(), connect.NewRequest(&templatespb.DeleteAllTemplatesRequest{}))
		assert.Error(t, err)
	})
}
