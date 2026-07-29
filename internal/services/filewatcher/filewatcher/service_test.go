// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package filewatcher

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	svc := New()
	require.NotNil(t, svc)
	assert.NotNil(t, svc.entries)
	assert.NotNil(t, svc.dirMap)
}

func TestStartStop(t *testing.T) {
	t.Parallel()

	t.Run("Start_Success", func(t *testing.T) {
		t.Parallel()
		svc := New()
		err := svc.Start()
		require.NoError(t, err)
		require.NotNil(t, svc.watcher)
		svc.Stop()
	})

	t.Run("Stop_SafeToCall", func(t *testing.T) {
		t.Parallel()
		svc := New()
		err := svc.Start()
		require.NoError(t, err)
		// Stop should be safe to call
		svc.Stop()
	})
}

func TestWatchUnwatch(t *testing.T) {
	t.Parallel()

	t.Run("Watch_ValidDir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		svc := New()
		require.NoError(t, svc.Start())
		defer svc.Stop()

		err := svc.Watch("source-1", dir)
		require.NoError(t, err)

		svc.mu.Lock()
		_, exists := svc.entries["source-1"]
		svc.mu.Unlock()
		assert.True(t, exists, "entry should exist after Watch")
	})

	t.Run("Unwatch_Success", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		svc := New()
		require.NoError(t, svc.Start())
		defer svc.Stop()

		require.NoError(t, svc.Watch("source-1", dir))
		svc.Unwatch("source-1")

		svc.mu.Lock()
		_, exists := svc.entries["source-1"]
		svc.mu.Unlock()
		assert.False(t, exists, "entry should be removed after Unwatch")
	})

	t.Run("Unwatch_NonExistent_NoPanic", func(t *testing.T) {
		t.Parallel()
		svc := New()
		require.NoError(t, svc.Start())
		defer svc.Stop()

		// Should not panic
		svc.Unwatch("nonexistent")
	})

	t.Run("Watch_NonExistentDir", func(t *testing.T) {
		t.Parallel()
		svc := New()
		require.NoError(t, svc.Start())
		defer svc.Stop()

		// collectDirs walks the path; a nonexistent dir returns empty dirs, no error.
		// The Watch call itself does not fail — it just has nothing to watch.
		err := svc.Watch("source-1", "/nonexistent/path/xyz")
		require.NoError(t, err)

		svc.mu.Lock()
		entry, exists := svc.entries["source-1"]
		svc.mu.Unlock()
		assert.True(t, exists, "entry should be created even for nonexistent dir")
		assert.Equal(t, "/nonexistent/path/xyz", entry.dirPath)
	})

	t.Run("Watch_ReplacesPrevious", func(t *testing.T) {
		t.Parallel()
		dir1 := t.TempDir()
		dir2 := t.TempDir()

		svc := New()
		require.NoError(t, svc.Start())
		defer svc.Stop()

		require.NoError(t, svc.Watch("source-1", dir1))
		require.NoError(t, svc.Watch("source-1", dir2))

		svc.mu.Lock()
		entry := svc.entries["source-1"]
		svc.mu.Unlock()
		assert.Equal(t, dir2, entry.dirPath, "should point to new directory")
	})
}

func TestSetCallback(t *testing.T) {
	t.Parallel()

	svc := New()
	assert.Nil(t, svc.callback)

	called := false
	svc.SetCallback(func(_ context.Context, _ string) {
		called = true
	})

	svc.mu.Lock()
	cb := svc.callback
	svc.mu.Unlock()
	require.NotNil(t, cb)

	cb(t.Context(), "test")
	assert.True(t, called)
}

func TestDebounce_ProtoFileChange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create an initial proto file so the watched directory has content.
	protoPath := filepath.Join(dir, "test.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3";`), 0o644))

	svc := New()
	require.NoError(t, svc.Start())
	defer svc.Stop()

	var callCount atomic.Int32
	done := make(chan struct{}, 1)

	svc.SetCallback(func(_ context.Context, _ string) {
		callCount.Add(1)
		select {
		case done <- struct{}{}:
		default:
		}
	})

	require.NoError(t, svc.Watch("source-1", dir))

	// Write to the proto file to trigger change
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3"; // updated`), 0o644))

	// Wait for callback to fire (debounce is 500ms, so give it up to 2s)
	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("callback did not fire within 2 seconds")
	}

	assert.GreaterOrEqual(t, callCount.Load(), int32(1))
}

func TestDebounce_MultipleWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	protoPath := filepath.Join(dir, "test.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3";`), 0o644))

	svc := New()
	require.NoError(t, svc.Start())
	defer svc.Stop()

	var callCount atomic.Int32
	done := make(chan struct{}, 10)

	svc.SetCallback(func(_ context.Context, _ string) {
		callCount.Add(1)
		select {
		case done <- struct{}{}:
		default:
		}
	})

	require.NoError(t, svc.Watch("source-1", dir))

	// Rapid writes — should be debounced to fewer callbacks
	for i := range 5 {
		require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3"; // v`+string(rune('0'+i))), 0o644))
		time.Sleep(50 * time.Millisecond) // 50ms between writes, well within 500ms debounce
	}

	// Wait for at least one callback
	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("callback did not fire within 2 seconds")
	}

	// Give extra time for any additional debounced fires
	time.Sleep(800 * time.Millisecond)

	count := callCount.Load()
	// Debounce should result in fewer callbacks than writes
	assert.LessOrEqual(t, count, int32(3), "expected debounce to reduce callback count, got %d", count)
}

func TestFileFiltering_NonProtoFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	svc := New()
	require.NoError(t, svc.Start())
	defer svc.Stop()

	var callCount atomic.Int32

	svc.SetCallback(func(_ context.Context, _ string) {
		callCount.Add(1)
	})

	require.NoError(t, svc.Watch("source-1", dir))

	// Write a non-proto file — should NOT trigger callback
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.go"), []byte("package main"), 0o644))

	// Wait enough time for debounce + some margin
	time.Sleep(1 * time.Second)

	assert.Equal(t, int32(0), callCount.Load(), "callback should NOT fire for non-.proto files")
}

func TestFileFiltering_ProtoFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	svc := New()
	require.NoError(t, svc.Start())
	defer svc.Stop()

	done := make(chan struct{}, 1)

	svc.SetCallback(func(_ context.Context, _ string) {
		select {
		case done <- struct{}{}:
		default:
		}
	})

	require.NoError(t, svc.Watch("source-1", dir))

	// Write a .proto file — SHOULD trigger callback
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.proto"), []byte(`syntax = "proto3";`), 0o644))

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("callback should fire for .proto file changes")
	}
}

func TestCollectDirs(t *testing.T) {
	t.Parallel()

	t.Run("IncludesSubdirs", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sub := filepath.Join(dir, "subpkg")
		require.NoError(t, os.MkdirAll(sub, 0o755))

		dirs, err := collectDirs(dir)
		require.NoError(t, err)
		assert.Len(t, dirs, 2)
		assert.Contains(t, dirs, dir)
		assert.Contains(t, dirs, sub)
	})

	t.Run("SkipsGitDir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0o755))

		dirs, err := collectDirs(dir)
		require.NoError(t, err)
		assert.NotContains(t, dirs, gitDir)
	})

	t.Run("SkipsNodeModules", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		nmDir := filepath.Join(dir, "node_modules")
		require.NoError(t, os.MkdirAll(nmDir, 0o755))

		dirs, err := collectDirs(dir)
		require.NoError(t, err)
		assert.NotContains(t, dirs, nmDir)
	})
}

func TestWatcherDetectsNewSubdirectory(t *testing.T) {
	dir := t.TempDir()
	svc := New()
	require.NoError(t, svc.Start())
	defer svc.Stop()

	fired := make(chan string, 4)
	svc.SetCallback(func(_ context.Context, sourceID string) { fired <- sourceID })
	require.NoError(t, svc.Watch("src-1", dir))

	sub := filepath.Join(dir, "newpkg")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	time.Sleep(100 * time.Millisecond) // let the watcher attach to the new dir
	require.NoError(t, os.WriteFile(filepath.Join(sub, "a.proto"), []byte("x"), 0o644))

	select {
	case id := <-fired:
		assert.Equal(t, "src-1", id)
	case <-time.After(3 * time.Second):
		t.Fatal("change in a new subdirectory was not noticed")
	}
}

func TestWatcherDetectsMovedInTree(t *testing.T) {
	dir := t.TempDir()
	staging := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(staging, "tree", "deep"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(staging, "tree", "deep", "x.proto"), []byte("x"), 0o644))

	svc := New()
	require.NoError(t, svc.Start())
	defer svc.Stop()
	fired := make(chan string, 4)
	svc.SetCallback(func(_ context.Context, sourceID string) { fired <- sourceID })
	require.NoError(t, svc.Watch("src-1", dir))

	require.NoError(t, os.Rename(filepath.Join(staging, "tree"), filepath.Join(dir, "tree")))

	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("moving a proto tree into the watched root was not noticed")
	}
}

func TestWatcherDirRemovalCleansMapAndFires(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "a.proto"), []byte("x"), 0o644))

	svc := New()
	require.NoError(t, svc.Start())
	defer svc.Stop()
	fired := make(chan string, 4)
	svc.SetCallback(func(_ context.Context, sourceID string) { fired <- sourceID })
	require.NoError(t, svc.Watch("src-1", dir))

	require.NoError(t, os.RemoveAll(sub))

	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("removing a directory with protos should trigger a recompile")
	}

	svc.mu.Lock()
	_, stale := svc.dirMap[sub]
	svc.mu.Unlock()
	assert.False(t, stale, "dirMap must not retain a removed directory")
}

// TestStartIsRaceFreeWithConcurrentAccess exercises Start alongside the
// mutex-guarded readers of s.watcher. Under -race an unsynchronized write to
// that field fails here.
func TestStartIsRaceFreeWithConcurrentAccess(t *testing.T) {
	t.Parallel()

	svc := New()
	dir := t.TempDir()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 50 {
			svc.Unwatch("absent")
			svc.SetCallback(nil)
		}
	}()

	require.NoError(t, svc.Start())
	_ = svc.Watch("src", dir)

	<-done
	svc.Stop()
}

// TestStopBeforeStartIsSafe covers the shutdown path when the watcher was never
// created (fx aborting startup before the filewatcher hook ran).
func TestStopBeforeStartIsSafe(t *testing.T) {
	t.Parallel()

	svc := New()
	svc.Stop()
	svc.Stop() // idempotent
}

// TestLoopSurvivesStopWithoutPanic pins that Stop closes the watcher and the
// loop exits rather than reading from a nil field.
func TestLoopSurvivesStopWithoutPanic(t *testing.T) {
	t.Parallel()

	svc := New()
	require.NoError(t, svc.Start())
	svc.Stop()

	// A second Stop must not panic on the already-closed watcher.
	svc.Stop()
}

func TestWatchBeforeStartIsRejected(t *testing.T) {
	t.Parallel()

	svc := New()

	err := svc.Watch("src", t.TempDir())

	require.ErrorIs(t, err, ErrNotStarted, "Watch must not dereference a nil watcher")
}
