// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package gitfetcher

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
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

func newFileService() *Service {
	s := New()
	s.allowFileScheme = true
	return s
}

type testRepo struct {
	t    *testing.T
	dir  string
	repo *git.Repository
	wt   *git.Worktree
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)
	return &testRepo{t: t, dir: dir, repo: repo, wt: wt}
}

func (r *testRepo) url() string { return "file://" + r.dir }

func (r *testRepo) commit(files map[string]string) plumbing.Hash {
	r.t.Helper()
	for p, content := range files {
		full := filepath.Join(r.dir, filepath.FromSlash(p))
		require.NoError(r.t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(r.t, os.WriteFile(full, []byte(content), 0o644))
		_, err := r.wt.Add(p)
		require.NoError(r.t, err)
	}
	hash, err := r.wt.Commit("change", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@x", When: time.Now()},
	})
	require.NoError(r.t, err)
	return hash
}

func (r *testRepo) tag(name string, hash plumbing.Hash, annotated bool) {
	r.t.Helper()
	var opts *git.CreateTagOptions
	if annotated {
		opts = &git.CreateTagOptions{Message: name, Tagger: &object.Signature{Name: "t", Email: "t@x", When: time.Now()}}
	}
	_, err := r.repo.CreateTag(name, hash, opts)
	require.NoError(r.t, err)
}

func (r *testRepo) branch(name string, hash plumbing.Hash) {
	r.t.Helper()
	require.NoError(r.t, r.repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName(name), hash)))
}

const schemaV1 = `syntax = "proto3"; message X { string a = 1; }`

const schemaV2 = `syntax = "proto3"; message X { string a = 1; int32 b = 2; }`

func refNames(refs []entities.ProtoRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = string(r.Kind) + ":" + r.Name
	}
	return out
}

func TestListRefs_TagsNewestFirstThenBranches(t *testing.T) {
	t.Parallel()
	r := newTestRepo(t)
	c := r.commit(map[string]string{"schema.proto": schemaV1})
	for _, tag := range []string{"v0.1.0", "v0.2.0", "v0.10.0", "v1.0.0-alpha", "v1.0.0"} {
		r.tag(tag, c, false)
	}
	r.branch("develop", c)
	r.branch("main", c)

	refs, err := newFileService().ListRefs(t.Context(), r.url())
	require.NoError(t, err)
	assert.Equal(t, []string{
		"tag:v1.0.0", "tag:v1.0.0-alpha", "tag:v0.10.0", "tag:v0.2.0", "tag:v0.1.0",
		"branch:main", "branch:master", "branch:develop",
	}, refNames(refs))
	for _, ref := range refs {
		assert.Equal(t, c.String(), ref.Revision, ref.Name)
	}
}

func TestListRefs_AnnotatedTagPointsAtCommit(t *testing.T) {
	t.Parallel()
	r := newTestRepo(t)
	c := r.commit(map[string]string{"schema.proto": schemaV1})
	r.tag("v1.0.0", c, true)

	refs, err := newFileService().ListRefs(t.Context(), r.url())
	require.NoError(t, err)
	require.Equal(t, "tag:v1.0.0", refNames(refs)[0])
	assert.Equal(t, c.String(), refs[0].Revision, "annotated tags resolve to the commit, not the tag object")
}

func TestResolveRef(t *testing.T) {
	t.Parallel()
	r := newTestRepo(t)
	first := r.commit(map[string]string{"schema.proto": schemaV1})
	r.tag("v1.0.0", first, false)
	second := r.commit(map[string]string{"schema.proto": schemaV2})
	svc := newFileService()

	tests := []struct {
		name    string
		ref     string
		want    entities.ProtoRef
		wantErr error
	}{
		{name: "tag", ref: "v1.0.0", want: entities.ProtoRef{Name: "v1.0.0", Kind: entities.RefKindTag, Revision: first.String()}},
		{name: "full tag ref", ref: "refs/tags/v1.0.0", want: entities.ProtoRef{Name: "v1.0.0", Kind: entities.RefKindTag, Revision: first.String()}},
		{name: "branch", ref: " master ", want: entities.ProtoRef{Name: "master", Kind: entities.RefKindBranch, Revision: second.String()}},
		{name: "full commit sha", ref: strings.ToUpper(first.String()), want: entities.ProtoRef{Name: first.String(), Kind: entities.RefKindCommit, Revision: first.String()}},
		{name: "short sha", ref: first.String()[:7], wantErr: errs.ErrProtoRefNotFound},
		{name: "unknown", ref: "v9.9.9", wantErr: errs.ErrProtoRefNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := svc.ResolveRef(t.Context(), r.url(), tt.ref)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFetch_TagBranchAndCommit(t *testing.T) {
	t.Parallel()
	r := newTestRepo(t)
	first := r.commit(map[string]string{"schema.proto": schemaV1})
	r.tag("v1.0.0", first, true)
	second := r.commit(map[string]string{"schema.proto": schemaV2})
	svc := newFileService()

	tests := []struct {
		name     string
		ref      entities.ProtoRef
		revision plumbing.Hash
		content  string
	}{
		{name: "annotated tag", ref: entities.ProtoRef{Name: "v1.0.0", Kind: entities.RefKindTag}, revision: first, content: schemaV1},
		{name: "branch", ref: entities.ProtoRef{Name: "master", Kind: entities.RefKindBranch}, revision: second, content: schemaV2},
		{name: "commit", ref: entities.ProtoRef{Name: first.String(), Kind: entities.RefKindCommit, Revision: first.String()}, revision: first, content: schemaV1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := svc.Fetch(t.Context(), r.url(), tt.ref)
			require.NoError(t, err)
			assert.Equal(t, tt.revision.String(), res.Revision)
			require.Len(t, res.Files, 1)
			assert.Equal(t, "schema.proto", res.Files[0].Path)
			assert.Equal(t, tt.content, res.Files[0].Content)
		})
	}
}

func TestFetch_MissingRefs(t *testing.T) {
	t.Parallel()
	r := newTestRepo(t)
	r.commit(map[string]string{"schema.proto": schemaV1})
	svc := newFileService()

	_, err := svc.Fetch(t.Context(), r.url(), entities.ProtoRef{Name: "v2.0.0", Kind: entities.RefKindTag})
	require.ErrorIs(t, err, errs.ErrProtoRefNotFound)

	missing := strings.Repeat("a", 40)
	_, err = svc.Fetch(t.Context(), r.url(), entities.ProtoRef{Name: missing, Kind: entities.RefKindCommit, Revision: missing})
	require.ErrorIs(t, err, errs.ErrProtoRefNotFound)
}

func TestValidateRepository_Valid(t *testing.T) {
	t.Parallel()
	r := newTestRepo(t)
	r.commit(map[string]string{"schema.proto": schemaV1})

	require.NoError(t, newFileService().ValidateRepository(t.Context(), r.url()))
}

func TestValidateRepository_NonExistent(t *testing.T) {
	t.Parallel()
	require.Error(t, newFileService().ValidateRepository(t.Context(), "file:///nonexistent/path/to/repo"))
}

func TestRemoteErrors_DoNotLeakResponseBody(t *testing.T) {
	t.Parallel()
	const secretBanner = "server_id=SECRET-BANNER-7f3a xkey=leaked-key"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(secretBanner))
	}))
	defer srv.Close()
	svc := New()

	err := svc.ValidateRepository(t.Context(), srv.URL)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-BANNER")

	_, err = svc.ListRefs(t.Context(), srv.URL)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-BANNER")
}

func TestFetch_SkipsOversizeAndCollectsConfigs(t *testing.T) {
	t.Parallel()
	r := newTestRepo(t)
	c := r.commit(map[string]string{
		"a.proto":     `syntax = "proto3";`,
		"buf.yaml":    "version: v2",
		"buf.lock":    "version: v2\ndeps: []",
		"sub/b.proto": `syntax = "proto3";`,
		"big.proto":   strings.Repeat("x", MaxFileSize+1),
	})
	r.tag("v1.0.0", c, false)

	res, err := newFileService().Fetch(t.Context(), r.url(), entities.ProtoRef{Name: "v1.0.0", Kind: entities.RefKindTag})
	require.NoError(t, err, "oversize files do not abort the fetch")
	assert.ElementsMatch(t, []string{"a.proto", "sub/b.proto"}, entryPathsGF(res.Files))
	assert.ElementsMatch(t, []string{"buf.yaml", "buf.lock"}, entryPathsGF(res.Configs))
	require.Len(t, res.Skipped, 1)
	assert.Equal(t, "big.proto", res.Skipped[0].Path)
	assert.Contains(t, res.Skipped[0].Reason, "exceeds")
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
