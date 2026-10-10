// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

//go:build unix

package natscontext_test

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/natscontext"
)

func TestRead_SkipsAFIFOWithoutWaitingForAWriter(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "nats", "context")
	fifo := filepath.Join(root, "creds.fifo")
	writeFile(t, filepath.Join(dir, "prod.json"), `{"url": "nats://a:4222", "creds": "`+fifo+`"}`)
	require.NoError(t, syscall.Mkfifo(fifo, 0o600))
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe.json"), 0o600))

	done := make(chan struct{})
	go func() {
		defer close(done)
		contexts, err := natscontext.Read(dir)
		assert.NoError(t, err)
		prod := byName(t, contexts, "prod")
		assert.NotEmpty(t, prod.Warnings)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reading the contexts waits on a FIFO")
	}
}
