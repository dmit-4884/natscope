// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package logconsole

import (
	"cmp"
	"io"
	"log/slog"

	"github.com/altessa-s/go-atlas/config"
	"github.com/altessa-s/go-atlas/observability/slog/handler/colorized"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	slogfactory "github.com/altessa-s/go-atlas/observability/slog/factory"
)

// Format is the logger outputFormat value that selects this handler.
const Format config.LogFormat = "console"

const (
	timeFormat      = "15:04:05"
	defaultAppGroup = "app"
)

// Register installs the console format into the go-atlas logger factory.
func Register() {
	slogfactory.RegisterHandler(Format, NewHandler)
}

// NewHandler is a slogfactory.HandlerFactory for the console format.
func NewHandler(w io.Writer, cfg *config.Logger, opts *slog.HandlerOptions) slog.Handler {
	colorOpts := []colorized.Option{
		colorized.WithTimeFormat(timeFormat),
		colorized.WithPrefixAttributeKey(slogx.ModuleKey),
	}
	if opts != nil {
		if opts.Level != nil {
			colorOpts = append(colorOpts, colorized.WithLevel(opts.Level))
		}
		colorOpts = append(colorOpts, colorized.WithReplaceAttr(opts.ReplaceAttr))
		if opts.AddSource {
			colorOpts = append(colorOpts, colorized.WithAddSource())
		}
	}
	if !IsTerminal(w) || (cfg != nil && !cfg.Colorized) {
		colorOpts = append(colorOpts, colorized.WithNoColor())
	}

	appGroup := defaultAppGroup
	if cfg != nil {
		appGroup = cmp.Or(cfg.AppGroupName, defaultAppGroup)
	}

	return &handler{Handler: colorized.NewHandler(w, colorOpts...), appGroup: appGroup}
}

type handler struct {
	slog.Handler
	appGroup string
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	kept := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Key == h.appGroup && attr.Value.Kind() == slog.KindGroup {
			continue
		}
		kept = append(kept, attr)
	}
	return &handler{Handler: h.Handler.WithAttrs(kept), appGroup: h.appGroup}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{Handler: h.Handler.WithGroup(name), appGroup: h.appGroup}
}
