// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package logconsole_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/config"

	"github.com/dmit-4884/natscope/internal/pkg/logconsole"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	slogfactory "github.com/altessa-s/go-atlas/observability/slog/factory"
)

var recordTime = time.Date(2026, time.September, 4, 18, 0, 15, 0, time.UTC)

func newHandler(t *testing.T, buf *bytes.Buffer) slog.Handler {
	t.Helper()
	return logconsole.NewHandler(buf, &config.Logger{}, &slog.HandlerOptions{Level: slog.LevelDebug})
}

func record(level slog.Level, msg string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(recordTime, level, msg, 0)
	r.AddAttrs(attrs...)
	return r
}

func TestHandlerWritesCompactLine(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(t, &buf)

	err := h.Handle(t.Context(), record(slog.LevelWarn, "OS keychain unavailable", slog.String("dir", "/tmp/x")))

	require.NoError(t, err)
	require.Equal(t, "18:00:15 WARN OS keychain unavailable dir=/tmp/x\n", buf.String())
}

func TestHandlerRendersSubsystemAsPrefix(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(t, &buf)

	err := h.Handle(t.Context(), record(slog.LevelWarn, "vault fallback",
		slogx.Module("infra:secrets"), slog.String("dir", "/tmp/x")))

	require.NoError(t, err)
	require.Equal(t, "18:00:15 WARN infra:secrets vault fallback dir=/tmp/x\n", buf.String())
}

func TestHandlerDropsAppGroup(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(t, &buf).WithAttrs([]slog.Attr{
		slog.Group("app", slog.String("name", "natscope"), slog.String("version", "0.2.0")),
		slog.String("conn", "local"),
	})

	err := h.Handle(t.Context(), record(slog.LevelError, "boom"))

	require.NoError(t, err)
	require.Equal(t, "18:00:15 ERR boom conn=local\n", buf.String())
}

func TestHandlerHonorsAppGroupName(t *testing.T) {
	var buf bytes.Buffer
	h := logconsole.NewHandler(&buf, &config.Logger{AppGroupName: "svc"}, &slog.HandlerOptions{Level: slog.LevelDebug}).
		WithAttrs([]slog.Attr{slog.Group("svc", slog.String("name", "natscope"))})

	err := h.Handle(t.Context(), record(slog.LevelInfo, "hello"))

	require.NoError(t, err)
	require.NotContains(t, buf.String(), "svc.name")
}

func TestHandlerKeepsAppFilterAfterWithGroup(t *testing.T) {
	var buf bytes.Buffer
	h := newHandler(t, &buf).WithGroup("req").WithAttrs([]slog.Attr{
		slog.Group("app", slog.String("name", "natscope")),
	})

	err := h.Handle(t.Context(), record(slog.LevelInfo, "hello", slog.Int("id", 7)))

	require.NoError(t, err)
	require.Contains(t, buf.String(), "hello")
	require.Contains(t, buf.String(), "id=7")
	require.NotContains(t, buf.String(), "app.name")
}

func TestHandlerFiltersByLevel(t *testing.T) {
	h := logconsole.NewHandler(&bytes.Buffer{}, &config.Logger{}, &slog.HandlerOptions{Level: slog.LevelWarn})

	require.False(t, h.Enabled(t.Context(), slog.LevelInfo))
	require.True(t, h.Enabled(t.Context(), slog.LevelWarn))
}

func TestRegisterMakesFormatBuildable(t *testing.T) {
	logconsole.Register()

	logger, err := slogfactory.New(&config.Logger{
		Level:        config.LoggerLevelWarning,
		Output:       config.LoggerConsoleOutputStderr,
		OutputFormat: logconsole.Format,
	}).Build()

	require.NoError(t, err)
	require.NotNil(t, logger)
}

func TestIsTerminal(t *testing.T) {
	require.False(t, logconsole.IsTerminal(&bytes.Buffer{}))

	f, err := os.Create(filepath.Join(t.TempDir(), "plain"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	require.False(t, logconsole.IsTerminal(f))
}
