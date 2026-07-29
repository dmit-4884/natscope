// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package filewatcher

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/altessa-s/go-atlas/core/runtime/panics"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	fwsvc "github.com/dmit-4884/natscope/internal/services/filewatcher"
)

const debounceDuration = 500 * time.Millisecond

// ErrNotStarted is returned by Watch before Start has created the watcher.
var ErrNotStarted = errors.New("filewatcher: not started")

type watchEntry struct {
	sourceID string
	dirPath  string
	timer    *time.Timer
}

// Service implements filewatcher.Service using fsnotify.
type Service struct {
	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	entries  map[string]*watchEntry // sourceID -> entry
	dirMap   map[string]string      // watched dir path -> sourceID
	callback fwsvc.ChangeCallback
	logger   *slog.Logger
	stopCh   chan struct{}
	stopOnce sync.Once

	// ctx scopes fired callbacks; cancel fires on Stop.
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new filewatcher service.
func New() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		entries: make(map[string]*watchEntry),
		dirMap:  make(map[string]string),
		logger:  slog.Default().With(slogx.Module("service:filewatcher")),
		stopCh:  make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// SetCallback sets the change notification callback.
func (s *Service) SetCallback(cb fwsvc.ChangeCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callback = cb
}

// Start initializes fsnotify and begins the event loop.
func (s *Service) Start() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.watcher = w
	s.mu.Unlock()

	go s.loop(w)

	s.logger.Info("filewatcher started")
	return nil
}

// Stop shuts down the watcher; safe to call more than once.
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.cancel()
	})

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop all timers
	for _, entry := range s.entries {
		if entry.timer != nil {
			entry.timer.Stop()
		}
	}

	if s.watcher != nil {
		_ = s.watcher.Close()
	}

	s.logger.Info("filewatcher stopped")
}

// Watch starts watching a directory for .proto file changes.
func (s *Service) Watch(sourceID string, dirPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.watcher == nil {
		return ErrNotStarted
	}

	// Unwatch previous if exists
	if existing, ok := s.entries[sourceID]; ok {
		s.removeWatchLocked(existing)
	}

	// Add the root directory and all subdirectories to watcher
	dirs, err := collectDirs(dirPath)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if err := s.watcher.Add(dir); err != nil {
			s.logger.Warn("failed to watch directory",
				slog.String("dir", dir),
				slogx.Error(err))
		}
		s.dirMap[dir] = sourceID
	}

	s.entries[sourceID] = &watchEntry{
		sourceID: sourceID,
		dirPath:  dirPath,
	}

	s.logger.Info("watching directory",
		slog.String("source_id", sourceID),
		slog.String("dir", dirPath),
		slog.Int("subdirs", len(dirs)))

	return nil
}

// Unwatch stops watching a directory.
func (s *Service) Unwatch(sourceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.entries[sourceID]; ok {
		s.removeWatchLocked(entry)
		delete(s.entries, sourceID)

		s.logger.Info("unwatched directory",
			slog.String("source_id", sourceID))
	}
}

func (s *Service) removeWatchLocked(entry *watchEntry) {
	if entry.timer != nil {
		entry.timer.Stop()
	}

	// Remove all dir watches for this source
	for dir, sid := range s.dirMap {
		if sid == entry.sourceID {
			_ = s.watcher.Remove(dir) //nolint:errcheck // best-effort: watcher may have already auto-removed if path is gone
			delete(s.dirMap, dir)
		}
	}
}

func (s *Service) loop(w *fsnotify.Watcher) {
	defer panics.Handle(s.ctx)
	for {
		select {
		case <-s.stopCh:
			return

		case event, ok := <-w.Events:
			if !ok {
				return
			}
			s.handleEvent(event)

		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			s.logger.Warn("fsnotify error", slogx.Error(err))
		}
	}
}

func (s *Service) handleEvent(event fsnotify.Event) {
	switch {
	case strings.HasSuffix(event.Name, ".proto"):
		if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
			return
		}
		s.scheduleBySubpath(filepath.Dir(event.Name))

	case event.Op&fsnotify.Create != 0:
		// fsnotify is not recursive, so newly created/moved-in subtrees must be wired
		// up by hand.
		info, err := os.Stat(event.Name)
		if err != nil || !info.IsDir() {
			return
		}
		s.addSubtree(event.Name)
		s.scheduleBySubpath(event.Name)

	case event.Op&(fsnotify.Remove|fsnotify.Rename) != 0:
		// A watched directory may have disappeared together with its protos.
		s.mu.Lock()
		_, wasWatchedDir := s.dirMap[event.Name]
		s.mu.Unlock()
		if wasWatchedDir {
			s.removeSubtreeWatches(event.Name)
			s.scheduleBySubpath(filepath.Dir(event.Name))
		}
	}
}

// scheduleBySubpath finds the owning source for a path (longest watched-dir
// prefix) and (re)arms its debounce timer.
func (s *Service) scheduleBySubpath(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceID, ok := s.dirMap[dir]
	if !ok {
		// The path may be deeper than any known directory (a freshly created
		// tree): find the owner by the longest entry-dir prefix.
		best := ""
		for _, e := range s.entries {
			if dir == e.dirPath || strings.HasPrefix(dir, e.dirPath+string(filepath.Separator)) {
				if len(e.dirPath) > len(best) {
					best = e.dirPath
					sourceID = e.sourceID
					ok = true
				}
			}
		}
		if !ok {
			return
		}
	}

	entry, ok := s.entries[sourceID]
	if !ok {
		return
	}
	if entry.timer != nil {
		entry.timer.Stop()
	}
	entry.timer = time.AfterFunc(debounceDuration, func() {
		s.fireCallback(entry.sourceID)
	})
}

// addSubtree wires a newly created directory (and its children) into the
// watcher under the owning source.
func (s *Service) addSubtree(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var owner string
	for _, e := range s.entries {
		if root == e.dirPath || strings.HasPrefix(root, e.dirPath+string(filepath.Separator)) {
			owner = e.sourceID
			break
		}
	}
	if owner == "" {
		return
	}
	dirs, err := collectDirs(root)
	if err != nil {
		s.logger.Warn("failed to collect new subtree", slog.String("dir", root), slogx.Error(err))
		return
	}
	for _, d := range dirs {
		if err := s.watcher.Add(d); err != nil {
			s.logger.Warn("failed to watch new directory", slog.String("dir", d), slogx.Error(err))
		}
		s.dirMap[d] = owner
	}
}

// removeSubtreeWatches drops dirMap entries (and watches) under a removed dir.
func (s *Service) removeSubtreeWatches(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for d := range s.dirMap {
		if d == root || strings.HasPrefix(d, root+string(filepath.Separator)) {
			_ = s.watcher.Remove(d) //nolint:errcheck // fsnotify auto-removes watches on deleted paths
			delete(s.dirMap, d)
		}
	}
}

func (s *Service) fireCallback(sourceID string) {
	defer panics.Handle(s.ctx)

	s.mu.Lock()
	cb := s.callback
	s.mu.Unlock()
	if cb == nil {
		return
	}
	s.logger.Info("proto source changed, triggering recompile", slog.String("source_id", sourceID))
	cb(s.ctx, sourceID)
}

// collectDirs returns the root dir and all subdirectories (for recursive
// watching).
func collectDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs, err
}

var _ fwsvc.Service = (*Service)(nil)
