// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"path"
	"sort"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"
)

// BufLayout is the module layout from buf config files.
type BufLayout struct {
	// Slash-relative module roots; empty means a single module at source root.
	Roots []string
	// Raw buf.lock contents (possibly several in a workspace).
	Locks [][]byte
}

// DetectBufLayout derives module roots from collected configs; found=false
// means no buf config. Shallowest buf.work.yaml wins, else union of buf.yaml roots.
func DetectBufLayout(configs []entities.ProtoFileEntry) (*BufLayout, bool) {
	var works, bufs []entities.ProtoFileEntry
	layout := &BufLayout{}
	for _, c := range configs {
		switch path.Base(c.Path) {
		case "buf.work.yaml":
			works = append(works, c)
		case "buf.yaml":
			bufs = append(bufs, c)
		case "buf.lock":
			layout.Locks = append(layout.Locks, []byte(c.Content))
		}
	}
	if len(works) == 0 && len(bufs) == 0 {
		return nil, false
	}

	if len(works) > 0 {
		sort.Slice(works, func(i, j int) bool {
			return strings.Count(works[i].Path, "/") < strings.Count(works[j].Path, "/")
		})
		w := works[0]
		for _, d := range yamlListItems(w.Content, "directories") {
			layout.Roots = append(layout.Roots, joinRelDir(path.Dir(w.Path), d))
		}
		layout.Roots = NormalizeRoots(layout.Roots)
		return layout, true
	}

	for _, b := range bufs {
		dir := path.Dir(b.Path)
		mods := bufModulePaths(b.Content)
		if len(mods) == 0 {
			// v1 module, or v2 without modules: rooted at config's dir.
			layout.Roots = append(layout.Roots, joinRelDir(dir, "."))
			continue
		}
		for _, m := range mods {
			layout.Roots = append(layout.Roots, joinRelDir(dir, m))
		}
	}
	layout.Roots = NormalizeRoots(layout.Roots)
	return layout, true
}

// joinRelDir joins a slash-relative dir ("." for root) with a child, returning
// a slash-relative result without leading "./".
func joinRelDir(dir, child string) string {
	if dir == "." {
		dir = ""
	}
	child = strings.Trim(strings.TrimSpace(child), "/")
	if child == "." {
		child = ""
	}
	switch {
	case dir == "":
		return child
	case child == "":
		return dir
	default:
		return dir + "/" + child
	}
}

// yamlListItems extracts scalar "- value" items under a top-level "key:" list
// in a minimal YAML doc, until the first non-list line. Quotes/comments stripped.
func yamlListItems(content, key string) []string {
	var out []string
	inKey := false
	for _, raw := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, key+":") && !strings.HasPrefix(raw, " ") {
			inKey = true
			continue
		}
		if !inKey {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			v := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			v = stripScalar(v)
			if v != "" {
				out = append(out, v)
			}
			continue
		}
		// Any non-item line ends the list.
		inKey = false
	}
	return out
}

// bufModulePaths extracts modules[].path from a buf.yaml v2 document.
func bufModulePaths(content string) []string {
	var out []string
	inModules := false
	for _, raw := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "modules:") && !strings.HasPrefix(raw, " ") {
			inModules = true
			continue
		}
		if !inModules {
			continue
		}
		// Top-level key ends the modules block.
		if !strings.HasPrefix(raw, " ") && !strings.HasPrefix(trimmed, "- ") {
			inModules = false
			continue
		}
		item := strings.TrimPrefix(trimmed, "- ")
		if k, v, ok := splitKV(item); ok && k == "path" {
			v = stripScalar(v)
			if v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// stripScalar removes trailing " #" comments and surrounding quotes.
func stripScalar(v string) string {
	return strings.TrimSpace(unquoteScalar(v))
}
