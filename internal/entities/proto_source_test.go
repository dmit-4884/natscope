// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoSourceNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew()
		require.NotNil(t, s)
		assert.NotEmpty(t, s.Id)
		assert.Positive(t, s.CreatedAt)
		assert.Empty(t, s.Name)
		assert.Empty(t, s.Repository)
		assert.Nil(t, s.Token)
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew(func(s *ProtoSource) {
			s.Name = "core-proto"
			s.Repository = "https://gitlab.com/org/proto.git"
		})
		assert.Equal(t, "core-proto", s.Name)
		assert.Equal(t, "https://gitlab.com/org/proto.git", s.Repository)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew(nil)
		require.NotNil(t, s)
	})
}

func TestProtoSource_ApplyUpdate(t *testing.T) {
	t.Parallel()

	t.Run("UpdatesFields", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew(func(s *ProtoSource) {
			s.Name = "old"
			s.Repository = "https://old.git"
		})
		// Seed a known-old UpdatedAt instead of sleeping (deterministic, race-free).
		s.UpdatedAt = time.UnixMilli(1).UTC()
		s.ApplyUpdate(&ProtoSourceUpdate{
			Name:       ptrString("new"),
			Repository: ptrString("https://new.git"),
		})

		assert.Equal(t, "new", s.Name)
		assert.Equal(t, "https://new.git", s.Repository)
		assert.True(t, s.UpdatedAt.After(time.UnixMilli(1).UTC()))
	})

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()
		var s *ProtoSource
		s.ApplyUpdate(&ProtoSourceUpdate{Name: ptrString("x")})
	})

	t.Run("NilRequest", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew()
		oldUpdatedAt := s.UpdatedAt
		s.ApplyUpdate(nil)
		assert.Equal(t, oldUpdatedAt, s.UpdatedAt)
	})
}

func TestProtoSource_AuthenticatedURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		repository string
		token      *string
		wantURL    string
	}{
		{
			name:       "WithToken",
			repository: "https://gitlab.com/org/proto.git",
			token:      ptrString("my-secret-token"),
			wantURL:    "https://oauth2:my-secret-token@gitlab.com/org/proto.git",
		},
		{
			name:       "WithoutToken_Nil",
			repository: "https://gitlab.com/org/proto.git",
			token:      nil,
			wantURL:    "https://gitlab.com/org/proto.git",
		},
		{
			name:       "WithEmptyToken",
			repository: "https://gitlab.com/org/proto.git",
			token:      ptrString(""),
			wantURL:    "https://gitlab.com/org/proto.git",
		},
		{
			name:       "InvalidURL",
			repository: "://invalid",
			token:      ptrString("token"),
			wantURL:    "://invalid",
		},
		{
			name:       "EmptyRepository",
			repository: "",
			token:      ptrString("token"),
			wantURL:    "//oauth2:token@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &ProtoSource{
				Repository: tt.repository,
				Token:      tt.token,
			}
			got := s.AuthenticatedURL()
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestProtoSourceNew_Defaults(t *testing.T) {
	t.Parallel()

	s := ProtoSourceNew()

	assert.Equal(t, SourceTypeGit, s.SourceType, "default SourceType should be git")
	assert.True(t, s.Enabled, "default Enabled should be true")
	assert.NotEmpty(t, s.Id)
	assert.Positive(t, s.CreatedAt)
	assert.Positive(t, s.UpdatedAt)
	assert.Nil(t, s.DeletedAt)
	assert.Nil(t, s.Token)
	assert.Nil(t, s.LocalPath)
	assert.False(t, s.WatcherEnabled)
}

func TestProtoSource_ApplyUpdate_LocalPath(t *testing.T) {
	t.Parallel()

	t.Run("UpdateLocalPath", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew(func(s *ProtoSource) {
			s.SourceType = SourceTypeLocal
		})

		s.ApplyUpdate(&ProtoSourceUpdate{
			LocalPath: ptrString("/new/path"),
		})

		require.NotNil(t, s.LocalPath)
		assert.Equal(t, "/new/path", *s.LocalPath)
	})

	t.Run("UpdateEnabled", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew(func(s *ProtoSource) {
			s.Enabled = true
		})

		enabled := false
		s.ApplyUpdate(&ProtoSourceUpdate{
			Enabled: &enabled,
		})

		assert.False(t, s.Enabled)
	})

	t.Run("NilFieldsNotOverwritten", func(t *testing.T) {
		t.Parallel()
		s := ProtoSourceNew(func(s *ProtoSource) {
			s.Name = "original"
			s.Repository = "https://original.git"
		})

		// Update only Name, Repository should remain unchanged
		s.ApplyUpdate(&ProtoSourceUpdate{
			Name: ptrString("updated"),
		})

		assert.Equal(t, "updated", s.Name)
		assert.Equal(t, "https://original.git", s.Repository)
	})
}

func TestProtoSource_AuthenticatedURL_NonGitTypes(t *testing.T) {
	t.Parallel()

	t.Run("LocalType_ReturnsRepository", func(t *testing.T) {
		t.Parallel()
		s := &ProtoSource{
			SourceType: SourceTypeLocal,
			Repository: "https://example.com/proto.git",
			Token:      ptrString("secret"),
		}
		got := s.AuthenticatedURL()
		assert.Equal(t, "https://example.com/proto.git", got, "non-git types should return Repository as-is")
	})

	t.Run("EmptySourceType_TreatedAsGit", func(t *testing.T) {
		t.Parallel()
		s := &ProtoSource{
			SourceType: "",
			Repository: "https://gitlab.com/org/proto.git",
			Token:      ptrString("token"),
		}
		got := s.AuthenticatedURL()
		assert.Equal(t, "https://oauth2:token@gitlab.com/org/proto.git", got)
	})
}

func TestProtoSources_IDs(t *testing.T) {
	t.Parallel()

	t.Run("MultipleItems", func(t *testing.T) {
		t.Parallel()
		sources := ProtoSources{
			{BaseEntity: BaseEntity{Id: "a"}},
			{BaseEntity: BaseEntity{Id: "b"}},
			{BaseEntity: BaseEntity{Id: "c"}},
		}
		assert.Equal(t, []string{"a", "b", "c"}, sources.IDs())
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		sources := ProtoSources{}
		assert.Empty(t, sources.IDs())
	})

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()
		var sources ProtoSources
		assert.Empty(t, sources.IDs())
	})
}
