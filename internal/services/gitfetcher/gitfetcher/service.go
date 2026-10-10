// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package gitfetcher

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"golang.org/x/mod/semver"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	coreerrs "github.com/altessa-s/go-atlas/core/errors"
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

	validateTimeout = 30 * time.Second
)

var commitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

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

// ListRefs returns tags newest first, then branches, with their commits.
func (s *Service) ListRefs(ctx context.Context, repositoryURL string) ([]entities.ProtoRef, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, s.timeout)
	defer cancel()

	refs, err := s.listRemote(ctx, repositoryURL)
	if err != nil {
		return nil, err
	}
	return collectRefs(refs), nil
}

// ResolveRef finds a tag, branch or full commit SHA; errs.ErrProtoRefNotFound when none matches.
func (s *Service) ResolveRef(ctx context.Context, repositoryURL, ref string) (entities.ProtoRef, error) {
	ref = strings.TrimSpace(ref)
	refs, err := s.ListRefs(ctx, repositoryURL)
	if err != nil {
		return entities.ProtoRef{}, err
	}
	name := strings.TrimPrefix(strings.TrimPrefix(ref, "refs/tags/"), "refs/heads/")
	for _, r := range refs {
		if r.Name == name {
			return r, nil
		}
	}
	if commitSHA.MatchString(ref) {
		sha := strings.ToLower(ref)
		return entities.ProtoRef{Name: sha, Kind: entities.RefKindCommit, Revision: sha}, nil
	}
	return entities.ProtoRef{}, fmt.Errorf("%w: %q is not a tag, branch or full commit SHA", errs.ErrProtoRefNotFound, ref)
}

// Fetch checks out ref and collects its .proto files and buf configs.
func (s *Service) Fetch(ctx context.Context, repositoryURL string, ref entities.ProtoRef) (*gitfetchersvc.FetchResult, error) {
	if err := s.checkRepoURL(repositoryURL); err != nil {
		return nil, err
	}
	opts := &git.CloneOptions{URL: repositoryURL, Auth: extractAuth(repositoryURL), Tags: git.NoTags}
	switch ref.Kind {
	case entities.RefKindTag:
		opts.ReferenceName, opts.Depth, opts.SingleBranch = plumbing.NewTagReferenceName(ref.Name), 1, true
	case entities.RefKindBranch:
		opts.ReferenceName, opts.Depth, opts.SingleBranch = plumbing.NewBranchReferenceName(ref.Name), 1, true
	case entities.RefKindCommit:
		opts.NoCheckout = true
	default:
		return nil, fmt.Errorf("%w: unsupported ref kind %q", errs.ErrInvalidRequest, ref.Kind)
	}

	ctx, cancel := corecontext.ApplyTimeout(ctx, s.timeout)
	defer cancel()

	tempDir, err := os.MkdirTemp(s.tempDir, TempDirPrefix)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "create temp dir")
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainCloneContext(ctx, tempDir, false, opts)
	if err != nil {
		s.logger.Error("clone failed",
			slog.String("url", maskURL(repositoryURL)), slog.String("ref", ref.Name), slogx.Error(err))
		if errors.Is(err, plumbing.ErrReferenceNotFound) || errors.Is(err, git.NoMatchingRefSpecError{}) {
			return nil, fmt.Errorf("%w: %q", errs.ErrProtoRefNotFound, ref.Name)
		}
		return nil, fmt.Errorf("clone repository at %q: %s", ref.Name, classifyGitErr(err))
	}

	revision, err := checkout(repo, ref)
	if err != nil {
		return nil, err
	}

	walk, err := protoutils.WalkProtoTree(tempDir, protoutils.WalkOptions{CollectConfigs: true, MaxFileSize: MaxFileSize})
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "walk repository")
	}
	for _, sk := range walk.Skipped {
		s.logger.Warn("fetch: skipped repository entry",
			slog.String("path", sk.Path), slog.String("reason", sk.Reason))
	}

	s.logger.Info("fetched proto files",
		slog.String("url", maskURL(repositoryURL)),
		slog.String("ref", ref.Name),
		slog.String("revision", revision),
		slog.Int("files", len(walk.Files)),
		slog.Int("configs", len(walk.Configs)))

	return &gitfetchersvc.FetchResult{
		Revision: revision,
		Files:    walk.Files,
		Configs:  walk.Configs,
		Skipped:  walk.Skipped,
	}, nil
}

// ValidateRepository checks if a repository URL is valid and accessible.
func (s *Service) ValidateRepository(ctx context.Context, repositoryURL string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, validateTimeout)
	defer cancel()

	_, err := s.listRemote(ctx, repositoryURL)
	return err
}

func (s *Service) listRemote(ctx context.Context, repositoryURL string) ([]*plumbing.Reference, error) {
	if err := s.checkRepoURL(repositoryURL); err != nil {
		return nil, err
	}
	remote := git.NewRemote(nil, &gitconfig.RemoteConfig{Name: "origin", URLs: []string{repositoryURL}})
	refs, err := remote.ListContext(ctx, &git.ListOptions{Auth: extractAuth(repositoryURL), PeelingOption: git.AppendPeeled})
	if err != nil {
		s.logger.Error("ls-remote failed", slog.String("url", maskURL(repositoryURL)), slogx.Error(err))
		return nil, fmt.Errorf("list remote refs: %s", classifyGitErr(err))
	}
	return refs, nil
}

func collectRefs(refs []*plumbing.Reference) []entities.ProtoRef {
	peeled := make(map[string]string)
	for _, r := range refs {
		if name := r.Name().String(); strings.HasSuffix(name, "^{}") {
			peeled[strings.TrimSuffix(name, "^{}")] = r.Hash().String()
		}
	}

	var tags, branches []entities.ProtoRef
	for _, r := range refs {
		name := r.Name()
		if r.Type() != plumbing.HashReference || strings.HasSuffix(name.String(), "^{}") {
			continue
		}
		revision := cmp.Or(peeled[name.String()], r.Hash().String())
		switch {
		case name.IsTag():
			tags = append(tags, entities.ProtoRef{Name: name.Short(), Kind: entities.RefKindTag, Revision: revision})
		case name.IsBranch():
			branches = append(branches, entities.ProtoRef{Name: name.Short(), Kind: entities.RefKindBranch, Revision: revision})
		}
	}

	slices.SortFunc(tags, func(a, b entities.ProtoRef) int { return -compareSemVer(a.Name, b.Name) })
	slices.SortFunc(branches, func(a, b entities.ProtoRef) int {
		return cmp.Or(cmp.Compare(branchRank(a.Name), branchRank(b.Name)), strings.Compare(a.Name, b.Name))
	})
	return append(tags, branches...)
}

var defaultBranches = []string{"main", "master"}

func branchRank(name string) int {
	if i := slices.Index(defaultBranches, name); i >= 0 {
		return i
	}
	return len(defaultBranches)
}

func checkout(repo *git.Repository, ref entities.ProtoRef) (string, error) {
	if ref.Kind != entities.RefKindCommit {
		head, err := repo.Head()
		if err != nil {
			return "", coreerrs.WrapOperation(err, "read checked out commit")
		}
		return head.Hash().String(), nil
	}
	hash := plumbing.NewHash(ref.Revision)
	if _, err := repo.CommitObject(hash); err != nil {
		return "", fmt.Errorf("%w: commit %s", errs.ErrProtoRefNotFound, ref.Revision)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", coreerrs.WrapOperation(err, "open worktree")
	}
	if err := wt.Checkout(&git.CheckoutOptions{Hash: hash, Force: true}); err != nil {
		return "", coreerrs.WrapOperation(err, "checkout commit")
	}
	return hash.String(), nil
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
	if before, _, ok := strings.Cut(u, "://"); ok {
		scheme := strings.ToLower(before)
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

// classifyGitErr maps a git transport error to a fixed message via go-git's sentinels. It never returns
// err.Error(), which can embed the remote server's raw response body.
func classifyGitErr(err error) string {
	switch {
	case errors.Is(err, transport.ErrRepositoryNotFound):
		return "repository not found"
	case errors.Is(err, transport.ErrEmptyRemoteRepository):
		return "remote repository is empty"
	case errors.Is(err, transport.ErrAuthenticationRequired), errors.Is(err, transport.ErrAuthorizationFailed):
		return "authentication required or access denied"
	case errors.Is(err, plumbing.ErrReferenceNotFound), errors.Is(err, git.NoMatchingRefSpecError{}):
		return "tag or reference not found"
	default:
		return "repository not accessible or not a git server"
	}
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
