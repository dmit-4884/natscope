// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package gitfetcher

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func entryPathsGF(in []entities.ProtoFileEntry) []string {
	out := make([]string, len(in))
	for i, f := range in {
		out[i] = f.Path
	}
	return out
}

func TestCompareSemVer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b string
		want int
		name string
	}{
		{"v1.0.0", "v1.0.0", 0, "equal_with_prefix"},
		{"1.0.0", "1.0.0", 0, "equal_no_prefix"},
		{"v1.0.0", "1.0.0", 0, "mixed_prefix"},
		{"v1.2.3", "v1.2.4", -1, "patch_lower"},
		{"v1.3.0", "v1.2.5", 1, "minor_higher"},
		{"v2.0.0", "v1.99.99", 1, "major_higher"},
		// Pre-release ordering: 1.0.0-alpha < 1.0.0
		{"v1.0.0-alpha", "v1.0.0", -1, "prerelease_less_than_release"},
		{"v1.0.0", "v1.0.0-alpha", 1, "release_greater_than_prerelease"},
		{"v1.0.0-alpha", "v1.0.0-beta", -1, "alpha_less_than_beta"},
		{"v1.0.0-rc.1", "v1.0.0-rc.2", -1, "rc_ordering"},
		// Build metadata is ignored by semver
		{"v1.0.0+build.1", "v1.0.0+build.2", 0, "build_metadata_ignored"},
		// Invalid versions fall back to lexical
		{"main", "develop", strings.Compare("main", "develop"), "lexical_fallback_branches"},
		{"foo", "v1.0.0", strings.Compare("foo", "v1.0.0"), "lexical_fallback_one_invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compareSemVer(tt.a, tt.b)
			// Normalize result to -1/0/+1
			switch {
			case got < 0:
				got = -1
			case got > 0:
				got = 1
			}
			want := tt.want
			switch {
			case want < 0:
				want = -1
			case want > 0:
				want = 1
			}
			assert.Equal(t, want, got, "compareSemVer(%q, %q)", tt.a, tt.b)
		})
	}
}

func TestExtractAuth(t *testing.T) {
	t.Parallel()

	t.Run("oauth2_token_extracted", func(t *testing.T) {
		t.Parallel()
		auth := extractAuth("https://oauth2:secret-token@github.com/foo/bar.git")
		require.NotNil(t, auth)
		assert.Equal(t, "oauth2", auth.Username)
		assert.Equal(t, "secret-token", auth.Password)
	})

	t.Run("no_auth_in_url", func(t *testing.T) {
		t.Parallel()
		auth := extractAuth("https://github.com/foo/bar.git")
		assert.Nil(t, auth)
	})

	t.Run("malformed_no_at_symbol", func(t *testing.T) {
		t.Parallel()
		auth := extractAuth("https://oauth2:tokenwithoutat")
		assert.Nil(t, auth)
	})
}

func TestMaskURL(t *testing.T) {
	t.Parallel()

	t.Run("masks_oauth_token", func(t *testing.T) {
		t.Parallel()
		got := maskURL("https://oauth2:secret-token@github.com/foo/bar.git")
		assert.NotContains(t, got, "secret-token")
		assert.Contains(t, got, "***@github.com")
		assert.Contains(t, got, "github.com/foo/bar.git")
	})

	t.Run("plain_url_unchanged", func(t *testing.T) {
		t.Parallel()
		const u = "https://github.com/foo/bar.git"
		assert.Equal(t, u, maskURL(u))
	})

	t.Run("masks_token_in_middle_of_error", func(t *testing.T) {
		t.Parallel()
		got := maskURL("authentication required: https://oauth2:ghp_secret@github.com/a/b.git")
		assert.NotContains(t, got, "ghp_secret")
		assert.Contains(t, got, "***@github.com")
	})

	t.Run("masks_token_containing_at_sign_without_tail_leak", func(t *testing.T) {
		t.Parallel()
		got := maskURL("https://oauth2:tok@en@github.com/a/b.git")
		assert.NotContains(t, got, "tok")
		assert.NotContains(t, got, "@en@")
		assert.Contains(t, got, "***@github.com")
	})

	t.Run("masks_token_in_query_parameter", func(t *testing.T) {
		t.Parallel()
		got := maskURL("clone failed: https://gitlab/x?private_token=abc123&ref=main")
		assert.NotContains(t, got, "abc123")
		assert.Contains(t, got, "private_token=***")
		assert.Contains(t, got, "ref=main")
	})

	t.Run("masks_multiple_urls", func(t *testing.T) {
		t.Parallel()
		got := maskURL("https://oauth2:AAA@h1/r and https://oauth2:BBB@h2/r")
		assert.NotContains(t, got, "AAA")
		assert.NotContains(t, got, "BBB")
	})
}

func TestEnsureSemverPrefix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "v1.0.0", ensureSemverPrefix("1.0.0"))
	assert.Equal(t, "v1.0.0", ensureSemverPrefix("v1.0.0"))
	assert.Equal(t, "vmain", ensureSemverPrefix("main"))
}

// --- Integration tests with a real local git repo ---

// newFileService returns a Service that accepts file:// remotes for local
// test repositories.
func newFileService() *Service {
	s := New()
	s.allowFileScheme = true
	return s
}

// makeRepo creates a local git repo with one .proto file and tags it.
// Returns the repo's filesystem URL suitable for go-git operations.
func makeRepo(t *testing.T, tags []string) string {
	t.Helper()

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// One proto file, committed once and tagged for each requested tag.
	protoPath := filepath.Join(dir, "schema.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3"; message X { string a = 1; }`), 0o644))

	_, err = wt.Add("schema.proto")
	require.NoError(t, err)

	commit, err := wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)

	for _, tag := range tags {
		_, err := repo.CreateTag(tag, commit, nil)
		require.NoError(t, err)
	}

	return "file://" + dir
}

func TestListTags_Integration(t *testing.T) {
	t.Parallel()

	repoURL := makeRepo(t, []string{"v0.1.0", "v0.2.0", "v0.10.0", "v1.0.0-alpha", "v1.0.0"})
	svc := newFileService()

	tags, err := svc.ListTags(t.Context(), repoURL)
	require.NoError(t, err)

	// Sorted newest-first by semver (prerelease sorts below its release).
	require.Equal(t, []string{"v1.0.0", "v1.0.0-alpha", "v0.10.0", "v0.2.0", "v0.1.0"}, tags)
}

func TestListTags_EmptyRepo(t *testing.T) {
	t.Parallel()
	repoURL := makeRepo(t, nil)
	svc := newFileService()

	tags, err := svc.ListTags(t.Context(), repoURL)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestValidateRepository_Valid(t *testing.T) {
	t.Parallel()
	repoURL := makeRepo(t, []string{"v1.0.0"})
	svc := newFileService()

	err := svc.ValidateRepository(t.Context(), repoURL)
	require.NoError(t, err)
}

func TestValidateRepository_NonExistent(t *testing.T) {
	t.Parallel()
	svc := newFileService()

	err := svc.ValidateRepository(t.Context(), "file:///nonexistent/path/to/repo")
	require.Error(t, err)
}

func TestFetchVersion_LoadsProtoFile(t *testing.T) {
	t.Parallel()
	repoURL := makeRepo(t, []string{"v1.0.0"})
	svc := newFileService()

	res, err := svc.FetchVersion(t.Context(), repoURL, "v1.0.0")
	require.NoError(t, err)
	require.Len(t, res.Files, 1)
	assert.Equal(t, "schema.proto", res.Files[0].Path)
	assert.Contains(t, res.Files[0].Content, "message X")
	assert.Greater(t, res.Files[0].Size, int64(0))
}

func TestFetchVersion_UnknownTag(t *testing.T) {
	t.Parallel()
	repoURL := makeRepo(t, []string{"v1.0.0"})
	svc := newFileService()

	_, err := svc.FetchVersion(t.Context(), repoURL, "v2.0.0")
	require.Error(t, err)
}

func TestFetchVersion_SkipsLargeProtoFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)

	// File > MaxFileSize
	bigContent := make([]byte, MaxFileSize+1)
	for i := range bigContent {
		bigContent[i] = 'x'
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.proto"), bigContent, 0o644))

	_, err = wt.Add("big.proto")
	require.NoError(t, err)
	commit, err := wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@x", When: time.Now()},
	})
	require.NoError(t, err)
	_, err = repo.CreateTag("v1", commit, nil)
	require.NoError(t, err)

	repoURL := "file://" + dir

	svc := newFileService()
	res, err := svc.FetchVersion(t.Context(), repoURL, "v1")
	require.NoError(t, err, "oversize no longer aborts the fetch")
	assert.Empty(t, res.Files)
	require.Len(t, res.Skipped, 1)
	assert.Equal(t, "big.proto", res.Skipped[0].Path)
	assert.Contains(t, res.Skipped[0].Reason, "exceeds")
}

func TestFetchVersionCollectsConfigsAndSkipsOversize(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)

	files := map[string]string{
		"a.proto":     `syntax = "proto3";`,
		"buf.yaml":    "version: v2",
		"buf.lock":    "version: v2\ndeps: []",
		"sub/b.proto": `syntax = "proto3";`,
	}
	for p, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	big := make([]byte, MaxFileSize+1)
	for i := range big {
		big[i] = 'x'
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.proto"), big, 0o644))

	// Stage every file (use slash paths; go-git uses forward slashes in-repo).
	for p := range files {
		_, addErr := wt.Add(p)
		require.NoError(t, addErr)
	}
	_, err = wt.Add("big.proto")
	require.NoError(t, err)

	commit, err := wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@x", When: time.Now()},
	})
	require.NoError(t, err)
	_, err = repo.CreateTag("v1.0.0", commit, nil)
	require.NoError(t, err)

	svc := newFileService()
	res, err := svc.FetchVersion(t.Context(), "file://"+dir, "v1.0.0")
	require.NoError(t, err, "oversize no longer aborts the fetch")
	assert.ElementsMatch(t, []string{"a.proto", "sub/b.proto"}, entryPathsGF(res.Files))
	assert.ElementsMatch(t, []string{"buf.yaml", "buf.lock"}, entryPathsGF(res.Configs))
	require.Len(t, res.Skipped, 1)
	assert.Equal(t, "big.proto", res.Skipped[0].Path)
}

func TestExtractAuth_TokenContainingAtSign(t *testing.T) {
	t.Parallel()

	auth := extractAuth("https://oauth2:tok%40en@github.com/foo/bar.git")

	require.NotNil(t, auth)
	assert.Equal(t, "oauth2", auth.Username)
	assert.Equal(t, "tok@en", auth.Password, "the whole token must survive, not the part before the first @")
}

func TestExtractAuth_GenericUserinfo(t *testing.T) {
	t.Parallel()

	auth := extractAuth("https://alice:pw@github.com/foo/bar.git")

	require.NotNil(t, auth)
	assert.Equal(t, "alice", auth.Username)
	assert.Equal(t, "pw", auth.Password)
}

func TestExtractAuth_SSHIsKeyBased(t *testing.T) {
	t.Parallel()

	assert.Nil(t, extractAuth("ssh://git@github.com/foo/bar.git"))
	assert.Nil(t, extractAuth("git@github.com:foo/bar.git"))
}

// TestMaskQueryValue_NonASCIIDoesNotPanic covers the offset hazard: lowercasing
// can change a string's byte length, so indices taken from a lowered copy would
// slice the original out of range.
func TestMaskQueryValue_NonASCIIDoesNotPanic(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"https://gitlab/İ?private_token=abc123",
		"https://gitlab/K?private_token=abc123",
		strings.Repeat("İ", 50) + "?access_token=abc123",
		"İ?private_token=",
		"KK?x-access-token=abc123&ref=main",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			got := maskURL(in)
			assert.NotContains(t, got, "abc123")
		})
	}
}

func TestCompareSemVer_IsTransitive(t *testing.T) {
	t.Parallel()

	// A mix of semver and non-semver tags: sorting must not depend on input order.
	tags := []string{"v2.0.0", "nightly", "v1.0.0", "latest", "v10.0.0", "alpha"}

	first := append([]string(nil), tags...)
	slices.SortFunc(first, func(a, b string) int { return -compareSemVer(a, b) })

	reversed := append([]string(nil), tags...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	slices.SortFunc(reversed, func(a, b string) int { return -compareSemVer(a, b) })

	assert.Equal(t, first, reversed, "sort order must not depend on the input permutation")
	assert.Equal(t, []string{"v10.0.0", "v2.0.0", "v1.0.0", "nightly", "latest", "alpha"}, first)
}
