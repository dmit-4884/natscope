// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	// "/"-separated parts in "<registry>/<owner>/<name>".
	expectedModuleNameParts = 3

	// Smallest scalar that could carry surrounding quotes; shorter is verbatim.
	minQuotedValueLen = 2
)

// bufDep is one BSR module dep from buf.lock. Name is
// "<registry>/<owner>/<name>"; DigestType ("b4"/"b5") names the cache subdir.
type bufDep struct {
	Name       string
	Commit     string
	DigestType string
}

// ResolveBufDeps parses buf.lock in sourceRoot and returns extra import paths
// for BSR deps. Never errors; a missing dep is skipped and surfaces downstream.
func ResolveBufDeps(sourceRoot string) []string {
	return resolveBufDepsWith(sourceRoot, defaultBufCacheDir())
}

// resolveBufDepsWith is the testable entry point with an explicit cache root.
func resolveBufDepsWith(sourceRoot, cacheRoot string) []string {
	if sourceRoot == "" || cacheRoot == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(sourceRoot, "buf.lock"))
	if err != nil {
		return nil
	}
	return resolveBufDepsFromLockWith(data, cacheRoot)
}

// ResolveBufDepsFromLock resolves BSR deps from in-memory buf.lock bytes
// against the default buf cache. Skip-on-missing like ResolveBufDeps.
func ResolveBufDepsFromLock(lockData []byte) []string {
	return resolveBufDepsFromLockWith(lockData, defaultBufCacheDir())
}

// resolveBufDepsFromLockWith resolves BSR dep include-dirs from buf.lock bytes
// against cacheRoot. Returns nil for any "nothing to add" reason; never errors.
func resolveBufDepsFromLockWith(lockData []byte, cacheRoot string) []string {
	if len(lockData) == 0 || cacheRoot == "" {
		return nil
	}
	deps := parseBufLock(lockData)
	if len(deps) == 0 {
		return nil
	}
	paths := make([]string, 0, len(deps))
	seen := make(map[string]bool, len(deps))
	for _, d := range deps {
		p, ok := resolveModuleFilesDir(cacheRoot, d)
		if !ok || seen[p] {
			continue
		}
		seen[p] = true
		paths = append(paths, p)
	}
	return paths
}

// defaultBufCacheDir follows buf CLI precedence: $BUF_CACHE_DIR verbatim (no
// /buf suffix), else Windows %LocalAppData%\buf, else $XDG_CACHE_HOME/buf or $HOME/.cache/buf.
func defaultBufCacheDir() string {
	if v := strings.TrimSpace(os.Getenv("BUF_CACHE_DIR")); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		if v := os.Getenv("LocalAppData"); v != "" {
			return filepath.Join(v, "buf")
		}
		return ""
	}
	if v := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); v != "" {
		return filepath.Join(v, "buf")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".cache", "buf")
}

// parseBufLock extracts deps from buf.lock (v1 and v2), detecting entry format
// per-item by fields present (v2: name; v1: remote/owner/repo), not top-level version.
func parseBufLock(data []byte) []bufDep {
	var (
		out []bufDep
		cur map[string]string
	)
	flush := func() {
		if cur == nil {
			return
		}
		if d, ok := entryToDep(cur); ok {
			out = append(out, d)
		}
		cur = nil
	}
	for _, raw := range bytes.Split(data, []byte{'\n'}) {
		line := string(raw)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			flush()
			cur = map[string]string{}
			trimmed = strings.TrimPrefix(trimmed, "- ")
		}
		if cur == nil {
			continue
		}
		k, v, ok := splitKV(trimmed)
		if !ok {
			continue
		}
		cur[k] = v
	}
	flush()
	return out
}

var commitRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// entryToDep validates one deps-list entry into a bufDep. ok=false if
// malformed/incomplete.
func entryToDep(e map[string]string) (bufDep, bool) {
	commit := strings.ToLower(strings.TrimSpace(e["commit"]))
	if !commitRe.MatchString(commit) {
		return bufDep{}, false
	}
	dt := digestTypeFromPrefix(e["digest"])
	if dt == "" {
		return bufDep{}, false
	}

	if name := strings.TrimSpace(e["name"]); name != "" {
		if !validModuleName(name) {
			return bufDep{}, false
		}
		return bufDep{Name: name, Commit: commit, DigestType: dt}, true
	}

	remote := strings.TrimSpace(e["remote"])
	owner := strings.TrimSpace(e["owner"])
	repo := strings.TrimSpace(e["repository"])
	if remote == "" || owner == "" || repo == "" {
		return bufDep{}, false
	}
	name := remote + "/" + owner + "/" + repo
	if !validModuleName(name) {
		return bufDep{}, false
	}
	return bufDep{Name: name, Commit: commit, DigestType: dt}, true
}

// digestTypeFromPrefix maps a buf.lock digest prefix to its cache subdir
// (b4/b5) — the prefix is the only authoritative signal.
func digestTypeFromPrefix(digest string) string {
	switch {
	case strings.HasPrefix(digest, "b5:"):
		return "b5"
	case strings.HasPrefix(digest, "shake256:"):
		return "b4"
	default:
		return ""
	}
}

// validModuleName enforces buf's "<registry>/<owner>/<name>" shape. Registry
// allows ':' for ports (self-hosted BSR); owner/name accept alnum plus -._.
func validModuleName(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) != expectedModuleNameParts {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			switch {
			case r >= 'a' && r <= 'z':
			case r >= 'A' && r <= 'Z':
			case r >= '0' && r <= '9':
			case r == '-' || r == '_' || r == '.' || r == ':':
			default:
				return false
			}
		}
	}
	return true
}

// splitKV splits "key: value" into (key, value, true). Inline comments after
// " #" are stripped; matching surrounding quotes on the value are removed.
func splitKV(line string) (string, string, bool) {
	idx := strings.IndexByte(line, ':')
	if idx <= 0 {
		return "", "", false
	}
	k := strings.TrimSpace(line[:idx])
	v := unquoteScalar(strings.TrimSpace(line[idx+1:]))
	return k, v, true
}

// unquoteScalar strips a trailing " #" inline comment and matching surrounding
// single/double quotes from a scalar value.
func unquoteScalar(v string) string {
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	if len(v) >= minQuotedValueLen {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	return v
}

// resolveModuleFilesDir locates a cached module's files directory. module.yaml
// missing/invalid means an incomplete cache entry — buf writes it last.
func resolveModuleFilesDir(cacheRoot string, dep bufDep) (string, bool) {
	modDir := filepath.Join(
		cacheRoot, "v3", "modules", dep.DigestType,
		filepath.FromSlash(dep.Name), dep.Commit,
	)
	if !dirExists(modDir) {
		return "", false
	}
	filesSub, ok := readModuleFilesDir(filepath.Join(modDir, "module.yaml"))
	if !ok {
		return "", false
	}
	full := filepath.Join(modDir, filesSub)
	if !dirExists(full) {
		return "", false
	}
	return full, true
}

// readModuleFilesDir reads a cached module.yaml, returning files_dir when
// version=="v1"; stops at the first list entry to avoid misreading `deps:` keys.
func readModuleFilesDir(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var version, filesDir string
	for _, raw := range bytes.Split(data, []byte{'\n'}) {
		line := strings.TrimSpace(string(raw))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			break
		}
		k, v, ok := splitKV(line)
		if !ok {
			continue
		}
		switch k {
		case "version":
			version = v
		case "files_dir":
			filesDir = v
		}
	}
	if version != "v1" || filesDir == "" {
		return "", false
	}
	return filesDir, true
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
