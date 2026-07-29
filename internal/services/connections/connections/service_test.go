// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	ptr "github.com/altessa-s/go-atlas/core/types/ptr"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	storage "github.com/dmit-4884/natscope/internal/storages/connections"
)

// --- Mocks ---

type mockStorage struct {
	saveErr    error
	getResult  *entities.SavedConnection
	getErr     error
	listResult *entities.List[entities.SavedConnections]
	listErr    error
	updateErr  error
	deleteErr  error

	saveCalled   bool
	saveInput    *entities.SavedConnection
	getCalled    bool
	getID        string
	updateCalled bool
	updateInput  *entities.SavedConnection
	deleteCalled bool
	deleteID     string
}

func (m *mockStorage) Save(_ context.Context, in *entities.SavedConnection) error {
	m.saveCalled = true
	m.saveInput = in
	return m.saveErr
}
func (m *mockStorage) Get(_ context.Context, id string, _ ...bool) (*entities.SavedConnection, error) {
	m.getCalled = true
	m.getID = id
	return m.getResult, m.getErr
}
func (m *mockStorage) List(_ context.Context, _ *entities.SavedConnectionsList) (*entities.List[entities.SavedConnections], error) {
	return m.listResult, m.listErr
}
func (m *mockStorage) Update(_ context.Context, in *entities.SavedConnection) error {
	m.updateCalled = true
	m.updateInput = in
	return m.updateErr
}
func (m *mockStorage) SoftDelete(_ context.Context, _ *entities.SoftDelete) error { return nil }
func (m *mockStorage) Delete(_ context.Context, id string) error {
	m.deleteCalled = true
	m.deleteID = id
	return m.deleteErr
}
func (m *mockStorage) Exists(_ context.Context, _ string, _ ...bool) (bool, error) {
	return false, nil
}
func (m *mockStorage) WithTransaction(ctx context.Context, handler func(context.Context, storage.Storage) error) error {
	return handler(ctx, m)
}

// mockNATSService embeds natssvc.ConnectionManager — the only role
// connections.Service depends on — and overrides the two methods it exercises.
type mockNATSService struct {
	natssvc.ConnectionManager

	disconnectCalled bool
	disconnectID     string

	testResult *entities.TestConnectionResult
	testErr    error
	testInput  *entities.TestConnectionRequest
}

func (m *mockNATSService) DisconnectFromPool(id string) {
	m.disconnectCalled = true
	m.disconnectID = id
}

func (m *mockNATSService) TestConnection(_ context.Context, in *entities.TestConnectionRequest) (*entities.TestConnectionResult, error) {
	m.testInput = in
	return m.testResult, m.testErr
}

// Compile-time interface check
var _ natssvc.ConnectionManager = (*mockNATSService)(nil)

// --- Tests ---

func TestService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *entities.SavedConnectionCreate
		saveErr error
		wantErr error
	}{
		{
			name: "Success",
			input: &entities.SavedConnectionCreate{
				Name: "test-conn",
				URLs: []string{"nats://localhost:4222"},
			},
		},
		{
			name: "StorageError_DuplicateName",
			input: &entities.SavedConnectionCreate{
				Name: "dup-name",
				URLs: []string{"nats://localhost:4222"},
			},
			saveErr: errs.ErrConnectionNameAlreadyInUse,
			wantErr: errs.ErrConnectionNameAlreadyInUse,
		},
		{
			name: "StorageError_Generic",
			input: &entities.SavedConnectionCreate{
				Name: "test",
				URLs: []string{"nats://localhost:4222"},
			},
			saveErr: errors.New("db error"),
			wantErr: errors.New("db error"),
		},
		{
			// Regression: buf.validate min_len=1 passes on the raw request, but
			// the normalizer trims "   " to "". The service must reject it
			// instead of persisting an empty name.
			name: "WhitespaceOnlyName_Rejected",
			input: &entities.SavedConnectionCreate{
				Name: "   ",
				URLs: []string{"nats://localhost:4222"},
			},
			wantErr: errs.ErrConnectionNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &mockStorage{saveErr: tt.saveErr}
			svc := New(store, nil)

			result, err := svc.Create(t.Context(), tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, errs.ErrConnectionNameAlreadyInUse) {
					assert.ErrorIs(t, err, errs.ErrConnectionNameAlreadyInUse)
				}
				if errors.Is(tt.wantErr, errs.ErrConnectionNameRequired) {
					assert.ErrorIs(t, err, errs.ErrConnectionNameRequired)
					assert.False(t, store.saveCalled, "storage.Save must not be called for an invalid name")
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.Id)
				assert.True(t, store.saveCalled, "storage.Save should be called")
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		conn := entities.SavedConnectionNew(func(c *entities.SavedConnection) {
			c.Name = "found"
		})
		store := &mockStorage{getResult: conn}
		svc := New(store, nil)

		result, err := svc.Get(t.Context(), conn.Id)
		require.NoError(t, err)
		assert.Equal(t, "found", result.Name)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{getErr: errs.ErrSavedConnectionNotFound}
		svc := New(store, nil)

		result, err := svc.Get(t.Context(), "nonexistent")
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
		assert.Nil(t, result)
	})
}

func TestService_List(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		expected := &entities.List[entities.SavedConnections]{
			Items:      entities.SavedConnections{{BaseEntity: entities.BaseEntity{Id: "1"}}},
			NextCursor: ptr.Wrap("next"),
		}
		store := &mockStorage{listResult: expected}
		svc := New(store, nil)

		result, err := svc.List(t.Context(), &entities.SavedConnectionsList{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		require.NotNil(t, result.NextCursor)
		assert.Equal(t, "next", *result.NextCursor)
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{listErr: errors.New("db error")}
		svc := New(store, nil)

		_, err := svc.List(t.Context(), &entities.SavedConnectionsList{})
		assert.Error(t, err)
	})
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		existing := entities.SavedConnectionNew(func(c *entities.SavedConnection) {
			c.Name = "old"
			c.URLs = []string{"nats://old:4222"}
		})
		store := &mockStorage{getResult: existing}
		natsSvc := &mockNATSService{}
		svc := New(store, natsSvc)

		result, err := svc.Update(t.Context(), &entities.SavedConnectionUpdate{
			Id:   existing.Id,
			Name: ptrStr("new"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "new", result.Name)
		assert.True(t, store.updateCalled)
		assert.True(t, natsSvc.disconnectCalled, "NATS pool should be disconnected")
		assert.Equal(t, existing.Id, natsSvc.disconnectID)
	})

	t.Run("GetFails_NotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{getErr: errs.ErrSavedConnectionNotFound}
		svc := New(store, nil)

		_, err := svc.Update(t.Context(), &entities.SavedConnectionUpdate{Id: "x"})
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})

	t.Run("UpdateFails", func(t *testing.T) {
		t.Parallel()
		existing := entities.SavedConnectionNew()
		store := &mockStorage{
			getResult: existing,
			updateErr: errs.ErrSavedConnectionNotFound,
		}
		svc := New(store, nil)

		_, err := svc.Update(t.Context(), &entities.SavedConnectionUpdate{Id: existing.Id})
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		existing := entities.SavedConnectionNew()
		store := &mockStorage{getResult: existing}
		natsSvc := &mockNATSService{}
		svc := New(store, natsSvc)

		err := svc.Delete(t.Context(), existing.Id)
		require.NoError(t, err)
		assert.True(t, store.deleteCalled)
		assert.Equal(t, existing.Id, store.deleteID)
		assert.True(t, natsSvc.disconnectCalled)
		assert.Equal(t, existing.Id, natsSvc.disconnectID)
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{deleteErr: errors.New("db error")}
		svc := New(store, nil)

		err := svc.Delete(t.Context(), "conn-1")
		assert.Error(t, err)
	})

}

func TestService_Duplicate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		existing := entities.SavedConnectionNew(func(c *entities.SavedConnection) {
			c.Name = "original"
			c.URLs = []string{"nats://localhost:4222"}
			c.Auth = &entities.AuthConfig{
				Method: entities.AuthMethodToken,
				Token:  ptrStr("secret"),
			}
		})
		store := &mockStorage{getResult: existing}
		svc := New(store, nil)

		result, err := svc.Duplicate(t.Context(), existing.Id, "copy")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "copy", result.Name)
		assert.Equal(t, existing.URLs, result.URLs)
		assert.Equal(t, existing.Auth, result.Auth)
		assert.NotEqual(t, existing.Id, result.Id)
		assert.True(t, store.saveCalled)
	})

	t.Run("SourceNotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{getErr: errs.ErrSavedConnectionNotFound}
		svc := New(store, nil)

		_, err := svc.Duplicate(t.Context(), "nonexistent", "copy")
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})
}

func TestService_TestConnection(t *testing.T) {
	t.Parallel()

	t.Run("SavedConfigOverridesRequest", func(t *testing.T) {
		t.Parallel()
		saved := entities.SavedConnectionNew(func(c *entities.SavedConnection) {
			c.Name = "stored"
			c.URLs = []string{"nats://trusted:4222"}
			c.Auth = &entities.AuthConfig{
				Method: entities.AuthMethodToken,
				Token:  ptrStr("stored-secret"),
			}
		})
		store := &mockStorage{getResult: saved}
		natsSvc := &mockNATSService{
			testResult: &entities.TestConnectionResult{Success: true, ServerVersion: "2.10.0"},
		}
		svc := New(store, natsSvc)

		result, err := svc.TestConnection(t.Context(), &entities.TestConnectionRequest{
			ConnectionID: saved.Id,
			URLs:         []string{"nats://attacker:4222"},
			Auth: &entities.AuthConfig{
				Method: entities.AuthMethodToken,
				Token:  ptrStr("attacker-token"),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		require.NotNil(t, natsSvc.testInput, "natService.TestConnection should be called")
		assert.Equal(t, saved.URLs, natsSvc.testInput.URLs, "stored URLs should override request")
		require.NotNil(t, natsSvc.testInput.Auth)
		require.NotNil(t, natsSvc.testInput.Auth.Token)
		assert.Equal(t, "stored-secret", *natsSvc.testInput.Auth.Token, "stored auth should override request")

		assert.True(t, store.updateCalled, "test result should be persisted")
		require.NotNil(t, store.updateInput)
		require.NotNil(t, store.updateInput.Meta, "recordTestResult should write Meta")
		assert.True(t, store.updateInput.Meta.LastSuccess)
	})

	t.Run("NoConnectionID_SkipsOverlayAndRecord", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		natsSvc := &mockNATSService{
			testResult: &entities.TestConnectionResult{Success: true},
		}
		svc := New(store, natsSvc)

		_, err := svc.TestConnection(t.Context(), &entities.TestConnectionRequest{
			URLs: []string{"nats://ad-hoc:4222"},
		})

		require.NoError(t, err)
		require.NotNil(t, natsSvc.testInput)
		assert.Equal(t, []string{"nats://ad-hoc:4222"}, natsSvc.testInput.URLs)
		assert.False(t, store.getCalled, "no stored config lookup without ConnectionID")
		assert.False(t, store.updateCalled, "no Meta write without ConnectionID")
	})
}

func ptrStr(s string) *string { return &s }

// TestService_LiftsURLCredentials pins the security invariant: credentials
// embedded in a server URL never reach the document store. They are moved into
// the auth config, which secretsplit routes to the vault.
func TestService_LiftsURLCredentials(t *testing.T) {
	t.Parallel()

	t.Run("Create_UserPassword", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store, &mockNATSService{})

		conn, err := svc.Create(t.Context(), &entities.SavedConnectionCreate{
			Name: "c",
			URLs: []string{"nats://bob:s3cr3t@h1:4222"},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"nats://h1:4222"}, conn.URLs)
		require.NotNil(t, conn.Auth)
		assert.Equal(t, entities.AuthMethodUserPass, conn.Auth.Method)
		assert.Equal(t, "bob", *conn.Auth.Username)
		assert.Equal(t, "s3cr3t", *conn.Auth.Password)

		require.NotNil(t, store.saveInput)
		assert.NotContains(t, store.saveInput.URLsString(), "s3cr3t",
			"the persisted document must not carry the secret in its URLs")
	})

	t.Run("Create_BareTokenBecomesTokenAuth", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store, &mockNATSService{})

		conn, err := svc.Create(t.Context(), &entities.SavedConnectionCreate{
			Name: "c",
			URLs: []string{"nats://tok3n@h1:4222"},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"nats://h1:4222"}, conn.URLs)
		require.NotNil(t, conn.Auth)
		assert.Equal(t, entities.AuthMethodToken, conn.Auth.Method)
		assert.Equal(t, "tok3n", *conn.Auth.Token)
	})

	t.Run("Create_CleanURLLeavesAuthUntouched", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store, &mockNATSService{})

		conn, err := svc.Create(t.Context(), &entities.SavedConnectionCreate{
			Name: "c",
			URLs: []string{"nats://h1:4222"},
			Auth: &entities.AuthConfig{Method: entities.AuthMethodToken, Token: ptrStr("explicit")},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"nats://h1:4222"}, conn.URLs)
		require.NotNil(t, conn.Auth)
		assert.Equal(t, "explicit", *conn.Auth.Token)
	})

	t.Run("Create_ConflictWithExplicitAuth", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store, &mockNATSService{})

		_, err := svc.Create(t.Context(), &entities.SavedConnectionCreate{
			Name: "c",
			URLs: []string{"nats://bob:s3cr3t@h1:4222"},
			Auth: &entities.AuthConfig{Method: entities.AuthMethodToken, Token: ptrStr("explicit")},
		})

		require.ErrorIs(t, err, errs.ErrConnectionURLCredentialsConflict)
		assert.False(t, store.saveCalled, "nothing may be persisted on conflict")
	})

	t.Run("Create_MixedCredentialsAcrossURLs", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store, &mockNATSService{})

		_, err := svc.Create(t.Context(), &entities.SavedConnectionCreate{
			Name: "c",
			URLs: []string{"nats://bob:s3cr3t@h1:4222", "nats://eve:hunter2@h2:4222"},
		})

		require.ErrorIs(t, err, errs.ErrConnectionURLCredentialsMixed)
		assert.False(t, store.saveCalled, "nothing may be persisted on mixed credentials")
	})

	t.Run("Update_LiftsIntoAuth", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{
			getResult: connWith(func(c *entities.SavedConnection) {
				c.Name = "c"
				c.URLs = []string{"nats://old:4222"}
			}),
		}
		svc := New(store, &mockNATSService{})

		conn, err := svc.Update(t.Context(), &entities.SavedConnectionUpdate{
			Id:   "id",
			URLs: []string{"nats://bob:s3cr3t@h1:4222"},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"nats://h1:4222"}, conn.URLs)
		require.NotNil(t, conn.Auth)
		assert.Equal(t, "s3cr3t", *conn.Auth.Password)

		require.NotNil(t, store.updateInput)
		assert.NotContains(t, store.updateInput.URLsString(), "s3cr3t")
	})

	t.Run("Update_NilURLsIsStillNoChange", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{
			getResult: connWith(func(c *entities.SavedConnection) {
				c.Name = "c"
				c.URLs = []string{"nats://kept:4222"}
			}),
		}
		svc := New(store, &mockNATSService{})

		conn, err := svc.Update(t.Context(), &entities.SavedConnectionUpdate{Id: "id"})

		require.NoError(t, err)
		assert.Equal(t, []string{"nats://kept:4222"}, conn.URLs)
	})

	t.Run("TestConnection_AdHocURLIsCleaned", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		natsSvc := &mockNATSService{testResult: &entities.TestConnectionResult{Success: true}}
		svc := New(store, natsSvc)

		_, err := svc.TestConnection(t.Context(), &entities.TestConnectionRequest{
			URLs: []string{"nats://bob:s3cr3t@h1:4222"},
		})

		require.NoError(t, err)
		require.NotNil(t, natsSvc.testInput)
		assert.Equal(t, []string{"nats://h1:4222"}, natsSvc.testInput.URLs)
		require.NotNil(t, natsSvc.testInput.Auth)
		assert.Equal(t, "s3cr3t", *natsSvc.testInput.Auth.Password)
	})
}

// connWith builds a saved connection for storage mocks.
func connWith(init func(*entities.SavedConnection)) *entities.SavedConnection {
	return entities.SavedConnectionNew(init)
}
