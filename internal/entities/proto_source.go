// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"net/url"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"
)

// SourceType determines how proto files are obtained.
type SourceType string

const (
	SourceTypeGit   SourceType = "git"
	SourceTypeLocal SourceType = "local"
	// SourceTypeFiles is a manual source: explicit .proto files + include dirs;
	// recompile is always user-triggered.
	SourceTypeFiles SourceType = "files"
)

// ProtoCompileResult snapshots the most recent compile attempt; server-side
// telemetry, never written by clients.
type ProtoCompileResult struct {
	At           int64
	Ok           bool
	Error        *string
	MessageCount int32
	FileCount    int32
	// Diagnostics is the bounded snapshot (first maxStoredDiagnostics entries) for
	// the UI.
	Diagnostics []CompileDiagnostic
	// Roots are the resolved import roots ("" implicit, never listed).
	Roots []string
	// RootsOrigin is the resolution tier that produced Roots: "manual" | "buf" |
	// "inferred".
	RootsOrigin string
}

// ProtoSource is a registered source of protobuf definitions.
type ProtoSource struct {
	BaseEntity

	// Name is the human-readable name (e.g., "Core Proto").
	Name string

	// SourceType determines how proto files are obtained.
	SourceType SourceType

	// Enabled controls whether this source participates in descriptor compilation.
	Enabled bool

	// Repository is the Git repository URL (Git type only).
	Repository string

	// Token is the access token for Git authentication (Git type only).
	Token *string

	// LocalPath is the directory path on server disk (Local type only).
	LocalPath *string

	// WatcherEnabled controls whether fsnotify watches the directory (Local type
	// only).
	WatcherEnabled bool

	// Files are the explicit .proto compile targets (Files type only); their dirs
	// are NOT auto-added as import paths.
	Files []string

	// IncludeDirs are import-resolution dirs (Files type only); not compiled —
	// like `protoc -I`.
	IncludeDirs []string

	// ImportRoots, when non-empty, DISABLES auto root detection: these become the
	// only roots (else buf/suffix inference).
	ImportRoots []string

	// ExcludePrefixes are slash-relative path prefixes excluded from compilation
	// (e.g. "gen", "pb"); no implicit defaults.
	ExcludePrefixes []string

	// LastCompile is the most recent compile pass result; server-side telemetry,
	// never accepted from clients.
	LastCompile *ProtoCompileResult
}

// ProtoSourceNew creates a new ProtoSource with generated Id and timestamps.
func ProtoSourceNew(init ...func(*ProtoSource)) *ProtoSource {
	s := &ProtoSource{
		BaseEntity: *New(),
		SourceType: SourceTypeGit,
		Enabled:    true,
	}

	if len(init) > 0 && init[0] != nil {
		init[0](s)
	}

	return s
}

// ApplyUpdate applies the update request to the source.
func (s *ProtoSource) ApplyUpdate(req *ProtoSourceUpdate) {
	if s == nil || req == nil {
		return
	}

	converter.Convert(req, s,
		converter.WithIgnoreNilValues(),
		converter.WithIgnoreFields("etag"))
	s.BeforeUpdate()
}

// AuthenticatedURL returns the repository URL with token authentication (Git
// type only).
func (s *ProtoSource) AuthenticatedURL() string {
	if s.SourceType != SourceTypeGit && s.SourceType != "" {
		return s.Repository
	}
	if s.Token == nil || *s.Token == "" {
		return s.Repository
	}

	u, err := url.Parse(s.Repository)
	if err != nil {
		return s.Repository
	}

	u.User = url.UserPassword("oauth2", *s.Token)
	return u.String()
}

// ProtoSources is a slice of ProtoSource pointers.
type ProtoSources []*ProtoSource

// IDs returns a slice of source IDs.
func (s ProtoSources) IDs() []string {
	return slices.To(s, func(source *ProtoSource) string { return source.Id })
}

// ProtoSourcesList is the listing criteria for sources.
type ProtoSourcesList struct {
	ListBase
}

// ProtoSourceCreate is the create DTO for a proto source.
type ProtoSourceCreate struct {
	Name       string     `normalize:"trim"`
	SourceType SourceType `normalize:"trim,lowercase"`

	// Git type fields
	Repository string  `normalize:"trim"`
	Token      *string `normalize:"nil_on_empty"`

	// Local type fields
	LocalPath      *string `normalize:"trim,nil_on_empty"`
	WatcherEnabled *bool

	// Files type fields
	Files       []string
	IncludeDirs []string

	// Compilation tweaks (optional; sensible defaults applied when nil/empty)
	ImportRoots     []string
	ExcludePrefixes []string
}

// ProtoSourceUpdate is the update DTO for a proto source.
type ProtoSourceUpdate struct {
	Id string `normalize:"trim"`

	Name            *string `normalize:"trim"`
	Repository      *string `normalize:"trim"`
	Token           *string
	LocalPath       *string `normalize:"trim,nil_on_empty"`
	Files           []string
	IncludeDirs     []string
	Enabled         *bool
	ImportRoots     []string
	ExcludePrefixes []string
}
