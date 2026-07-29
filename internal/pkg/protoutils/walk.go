// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// DefaultMaxProtoSize caps a single .proto file read during tree walks.
const DefaultMaxProtoSize = 1024 * 1024 // 1 MiB

// DefaultMaxTotalFiles and DefaultMaxTotalBytes bound a whole walk. The per-file
// cap alone does not: a fetched repository is attacker-shaped input and every
// file is held in memory as a string before it reaches storage.
const (
	DefaultMaxTotalFiles = 5000
	DefaultMaxTotalBytes = 64 * 1024 * 1024 // 64 MiB
)

// ErrWalkLimitExceeded is returned when a walk exceeds its file count or total
// byte budget.
var ErrWalkLimitExceeded = errors.New("protoutils: walk exceeds size limits")

// defaultExcludeDirNames are directory basenames never descended into.
var defaultExcludeDirNames = []string{".git", "node_modules"}

// bufConfigNames are the config files captured when CollectConfigs is set.
var bufConfigNames = map[string]bool{
	"buf.yaml":      true,
	"buf.work.yaml": true,
	"buf.lock":      true,
}

// WalkOptions tunes WalkProtoTree.
type WalkOptions struct {
	// ExcludePrefixes are slash-relative path prefixes (from the walk root)
	// dropped entirely. A prefix matches itself and everything under it.
	ExcludePrefixes []string
	// MaxFileSize overrides DefaultMaxProtoSize when > 0.
	MaxFileSize int64
	// MaxTotalFiles overrides DefaultMaxTotalFiles when > 0.
	MaxTotalFiles int
	// MaxTotalBytes overrides DefaultMaxTotalBytes when > 0.
	MaxTotalBytes int64
	// CollectConfigs captures buf.yaml / buf.work.yaml / buf.lock contents.
	CollectConfigs bool
}

// SkippedFile reports a path the walk intentionally did not ingest.
type SkippedFile struct {
	Path   string // slash-relative
	Reason string
}

// WalkResult is everything a walk produced.
type WalkResult struct {
	Files   []entities.ProtoFileEntry
	Configs []entities.ProtoFileEntry
	Skipped []SkippedFile
}

// WalkProtoTree walks root and returns every .proto file with slash-relative
// paths. Per-entry I/O errors become Skipped entries; only a dead root errors out.
func WalkProtoTree(root string, opts WalkOptions) (*WalkResult, error) {
	if info, err := os.Stat(root); err != nil {
		return nil, coreerrs.WrapOperation(err, "walk root")
	} else if !info.IsDir() {
		return nil, fmt.Errorf("walk root %q is not a directory", root)
	}

	maxSize := opts.MaxFileSize
	if maxSize <= 0 {
		maxSize = DefaultMaxProtoSize
	}
	maxFiles := opts.MaxTotalFiles
	if maxFiles <= 0 {
		maxFiles = DefaultMaxTotalFiles
	}
	maxBytes := opts.MaxTotalBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxTotalBytes
	}
	excludePrefixes := NormalizeRoots(opts.ExcludePrefixes)

	res := &WalkResult{}
	var totalFiles int
	var totalBytes int64
	walkErr := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)

		if err != nil {
			res.Skipped = append(res.Skipped, SkippedFile{Path: rel, Reason: "io error: " + err.Error()})
			return nil
		}
		// Never read through symlinks: os.ReadFile would follow them, so a link
		// named *.proto could exfiltrate arbitrary files and dodge the size cap.
		if d.Type()&os.ModeSymlink != 0 {
			res.Skipped = append(res.Skipped, SkippedFile{Path: rel, Reason: "symlink skipped"})
			return nil
		}
		if d.IsDir() {
			for _, name := range defaultExcludeDirNames {
				if d.Name() == name {
					return filepath.SkipDir
				}
			}
			for _, pref := range excludePrefixes {
				if rel == pref {
					res.Skipped = append(res.Skipped, SkippedFile{Path: rel, Reason: "excluded by prefix"})
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Files: excluded prefix may also point at a single file.
		for _, pref := range excludePrefixes {
			if rel == pref || strings.HasPrefix(rel, pref+"/") {
				return nil // directory-level skip already reported
			}
		}

		isProto := strings.HasSuffix(d.Name(), ".proto")
		isConfig := opts.CollectConfigs && bufConfigNames[d.Name()]
		if !isProto && !isConfig {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			res.Skipped = append(res.Skipped, SkippedFile{Path: rel, Reason: "stat failed: " + infoErr.Error()})
			return nil
		}
		if isProto && info.Size() > maxSize {
			res.Skipped = append(res.Skipped, SkippedFile{
				Path:   rel,
				Reason: fmt.Sprintf("size %d exceeds limit %d", info.Size(), maxSize),
			})
			return nil
		}

		totalFiles++
		totalBytes += info.Size()
		if totalFiles > maxFiles || totalBytes > maxBytes {
			return fmt.Errorf("%w: %d files / %d bytes (limits %d / %d)",
				ErrWalkLimitExceeded, totalFiles, totalBytes, maxFiles, maxBytes)
		}

		content, readErr := os.ReadFile(p)
		if readErr != nil {
			res.Skipped = append(res.Skipped, SkippedFile{Path: rel, Reason: "read failed: " + readErr.Error()})
			return nil
		}

		entry := entities.ProtoFileEntry{Path: rel, Content: string(content), Size: info.Size()}
		if isProto {
			res.Files = append(res.Files, entry)
		} else {
			res.Configs = append(res.Configs, entry)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return res, nil
}
