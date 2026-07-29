// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package gitfetcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"golang.org/x/mod/semver"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
	gitfetchersvc "github.com/dmit-4884/natscope/internal/services/gitfetcher"
	gitconfig "github.com/go-git/go-git/v5/config"
)

const (
	// DefaultTimeout is the default timeout for git operations.
	DefaultTimeout = 5 * time.Minute

	// TempDirPrefix is the prefix for temporary directories.
	TempDirPrefix = "natscope-proto-"

	// MaxFileSize is the maximum file size to read (1MB).
	MaxFileSize = 1024 * 1024
)

// Service implements gitfetchersvc.Service.
type Service struct {
	logger  *slog.Logger
	timeout time.Duration
	tempDir string
	// allowFileScheme permits file:// remotes; off in production, tests enable
	// it for local repositories.
	allowFileScheme bool
}

// New creates a new git fetcher service.
func New() *Service {
	return &Service{
		logger:  slog.Default().With(slogx.Module("service:gitfetcher")),
		timeout: DefaultTimeout,
		tempDir: os.TempDir(),
	}
}

// ListTags returns available tags from a Git repository using ls-remote (no
// clone needed).
func (s *Service) ListTags(ctx context.Context, repositoryURL string) ([]string, error) {
	if err := s.checkRepoURL(repositoryURL); err != nil {
		return nil, err
	}
	ctx, cancel := corecontext.ApplyTimeout(ctx, s.timeout)
	defer cancel()

	s.logger.Debug("fetching tags via ls-remote", slog.String("url", maskURL(repositoryURL)))

	// Use ls-remote to get tags without cloning
	auth := extractAuth(repositoryURL)
	remote := git.NewRemote(nil, &gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repositoryURL},
	})

	s.logger.Debug("starting ls-remote", slog.Bool("has_auth", auth != nil))

	refs, err := remote.ListContext(ctx, &git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		err = scrubURLErr(err)
		s.logger.Error("ls-remote failed", slog.String("error", err.Error()))
		return nil, errors.WrapOperation(err, "list remote refs")
	}
	s.logger.Debug("ls-remote completed", slog.Int("refs_count", len(refs)))

	// Extract tags from refs
	var tags []string
	for _, ref := range refs {
		refName := ref.Name().String()
		if strings.HasPrefix(refName, "refs/tags/") {
			tagName := strings.TrimPrefix(refName, "refs/tags/")
			// Skip annotated tag objects (the ^{} ones are handled by git internally)
			if !strings.HasSuffix(tagName, "^{}") {
				tags = append(tags, tagName)
			}
		}
	}

	// Sort tags by semantic version in reverse order (newest first)
	slices.SortFunc(tags, func(a, b string) int {
		return -compareSemVer(a, b)
	})

	s.logger.Info("fetched tags", slog.String("url", maskURL(repositoryURL)), slog.Int("count", len(tags)))

	return tags, nil
}

// FetchVersion fetches proto files and buf configs from a specific tag.
func (s *Service) FetchVersion(ctx context.Context, repositoryURL, tag string) (*gitfetchersvc.FetchResult, error) {
	if err := s.checkRepoURL(repositoryURL); err != nil {
		return nil, err
	}
	ctx, cancel := corecontext.ApplyTimeout(ctx, s.timeout)
	defer cancel()

	// Create temp directory for cloning
	tempDir, err := os.MkdirTemp(s.tempDir, TempDirPrefix)
	if err != nil {
		return nil, errors.WrapOperation(err, "create temp dir")
	}
	defer os.RemoveAll(tempDir)

	s.logger.Debug("cloning repository for version fetch",
		slog.String("url", maskURL(repositoryURL)),
		slog.String("tag", tag))

	// Clone at specific tag
	auth := extractAuth(repositoryURL)
	_, err = git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:           repositoryURL,
		Auth:          auth,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.NewTagReferenceName(tag),
		Tags:          git.NoTags,
	})
	if err != nil {
		return nil, errors.Wrapf(scrubURLErr(err), "failed to clone repository at tag %s", tag)
	}

	walk, err := protoutils.WalkProtoTree(tempDir, protoutils.WalkOptions{CollectConfigs: true, MaxFileSize: MaxFileSize})
	if err != nil {
		return nil, errors.WrapOperation(err, "walk repository")
	}
	for _, sk := range walk.Skipped {
		s.logger.Warn("fetch: skipped repository entry",
			slog.String("path", sk.Path), slog.String("reason", sk.Reason))
	}

	s.logger.Info("fetched proto files",
		slog.String("url", maskURL(repositoryURL)),
		slog.String("tag", tag),
		slog.Int("files", len(walk.Files)),
		slog.Int("configs", len(walk.Configs)))

	return &gitfetchersvc.FetchResult{
		Files:   walk.Files,
		Configs: walk.Configs,
		Skipped: walk.Skipped,
	}, nil
}

// ValidateRepository checks if a repository URL is valid and accessible.
func (s *Service) ValidateRepository(ctx context.Context, repositoryURL string) error {
	if err := s.checkRepoURL(repositoryURL); err != nil {
		return err
	}
	const validateTimeout = 30 * time.Second
	ctx, cancel := corecontext.ApplyTimeout(ctx, validateTimeout)
	defer cancel()

	// Create temp directory
	tempDir, err := os.MkdirTemp(s.tempDir, TempDirPrefix)
	if err != nil {
		return errors.WrapOperation(err, "create temp dir")
	}
	defer os.RemoveAll(tempDir)

	// Try to clone with minimal data
	auth := extractAuth(repositoryURL)
	_, err = git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:        repositoryURL,
		Auth:       auth,
		Depth:      1,
		NoCheckout: true,
		Tags:       git.NoTags,
	})
	if err != nil {
		return errors.Wrap(scrubURLErr(err), "repository not accessible")
	}

	return nil
}

// allowedGitSchemes lists the URL schemes accepted for remote git operations.
// The plaintext, unauthenticated git:// protocol is intentionally excluded.
var allowedGitSchemes = []string{"http", "https", "ssh"}

// checkRepoURL rejects unsupported schemes (defense-in-depth): only
// http/https/ssh and SCP-style ssh pass; file:// and empty/relative fail.
func (s *Service) checkRepoURL(repositoryURL string) error {
	u := strings.TrimSpace(repositoryURL)
	if u == "" {
		return fmt.Errorf("%w: repository URL is empty", errs.ErrInvalidRequest)
	}
	if i := strings.Index(u, "://"); i >= 0 {
		scheme := strings.ToLower(u[:i])
		switch {
		case slices.Contains(allowedGitSchemes, scheme):
			return nil
		case scheme == "file" && s.allowFileScheme:
			return nil
		default:
			return fmt.Errorf("%w: unsupported repository URL scheme: %s", errs.ErrInvalidRequest, scheme)
		}
	}
	// Scheme-less: accept SCP-style ssh ("user@host:path"), reject local paths.
	if at := strings.IndexByte(u, '@'); at > 0 && strings.IndexByte(u[at+1:], ':') >= 0 {
		return nil
	}
	return fmt.Errorf("%w: unsupported or relative repository URL", errs.ErrInvalidRequest)
}

// extractAuth lifts HTTP basic credentials out of an http(s) URL's userinfo.
// net/url rather than a first-"@" scan, so a token containing "@" survives.
// ssh URLs are left alone: their auth is key-based.
func extractAuth(repositoryURL string) *http.BasicAuth {
	u, err := url.Parse(repositoryURL)
	if err != nil || u.User == nil {
		return nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}
	username := u.User.Username()
	password, _ := u.User.Password()
	if username == "" && password == "" {
		return nil
	}
	return &http.BasicAuth{Username: username, Password: password}
}

// sensitiveQueryKeys are URL query parameters whose values are credentials.
var sensitiveQueryKeys = []string{"access_token", "private_token", "x-access-token"}

// maskURL redacts credentials embedded anywhere in s: scheme://userinfo@host
// (masking through the LAST '@' in the authority, so a token that itself
// contains '@' can't leak its tail) plus known token query parameters. Applied
// to every log line and error string so tokens never reach the client.
func maskURL(s string) string {
	out := maskUserinfo(s)
	for _, key := range sensitiveQueryKeys {
		out = maskQueryValue(out, key)
	}
	return out
}

// maskUserinfo replaces the userinfo of every scheme://user:token@host in s.
func maskUserinfo(s string) string {
	const sep = "://"
	out := s
	for i := 0; ; {
		scheme := strings.Index(out[i:], sep)
		if scheme == -1 {
			return out
		}
		scheme += i
		rest := out[scheme+len(sep):]
		authEnd := len(rest)
		if j := strings.IndexAny(rest, "/?#"); j != -1 {
			authEnd = j
		}
		// Only an '@' inside the authority (before any path/query) is userinfo.
		at := strings.LastIndexByte(rest[:authEnd], '@')
		if at == -1 {
			i = scheme + len(sep)
			continue
		}
		out = out[:scheme+len(sep)] + "***@" + rest[at+1:]
		i = scheme + len(sep) + len("***@")
	}
}

// maskQueryValue replaces the value of a "key=" pair wherever it appears in s.
// EqualFold over a fixed window, not ToLower(s): lowercasing can change byte
// length (İ, K) and offsets from the lowered copy would slice s wrongly.
func maskQueryValue(s, key string) string {
	const masked = "***"
	needle := key + "="
	out := s
	for start := 0; start+len(needle) <= len(out); {
		if !strings.EqualFold(out[start:start+len(needle)], needle) {
			start++
			continue
		}
		valStart := start + len(needle)
		valEnd := valStart
		for valEnd < len(out) && !strings.ContainsRune("&#? \t\n\r\"'", rune(out[valEnd])) {
			valEnd++
		}
		if valEnd == valStart {
			start = valEnd
			continue
		}
		out = out[:valStart] + masked + out[valEnd:]
		start = valStart + len(masked)
	}
	return out
}

// scrubURLErr masks embedded credentials in an error message. Drops the chain:
// go-git errors carry no sentinel to match and only their text leaks the token.
func scrubURLErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", maskURL(err.Error()))
}

// compareSemVer compares tags via x/mod/semver (-1/0/1). Semver tags rank above
// non-semver ones, which compare lexically among themselves: mixing the two
// orderings pairwise would not be transitive and would leave sort order
// dependent on the input permutation.
func compareSemVer(a, b string) int {
	av, bv := ensureSemverPrefix(a), ensureSemverPrefix(b)
	aOK, bOK := semver.IsValid(av), semver.IsValid(bv)
	switch {
	case aOK && bOK:
		return semver.Compare(av, bv)
	case aOK:
		return 1
	case bOK:
		return -1
	default:
		return strings.Compare(a, b)
	}
}

// ensureSemverPrefix prepends "v" if missing (x/mod/semver requires it).
func ensureSemverPrefix(s string) string {
	if strings.HasPrefix(s, "v") {
		return s
	}
	return "v" + s
}

// Ensure Service implements the interface.
var _ gitfetchersvc.Service = (*Service)(nil)
