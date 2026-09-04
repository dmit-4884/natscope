// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig_test

import (
	"testing"

	"github.com/altessa-s/go-atlas/config"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
)

func TestLoggerDefaultsInteractive(t *testing.T) {
	l := appconfig.LoggerDefaults(true)

	if l.Level != config.LoggerLevelWarning {
		t.Errorf("Level = %q, want %q", l.Level, config.LoggerLevelWarning)
	}
	if l.OutputFormat != logconsole.Format {
		t.Errorf("OutputFormat = %q, want %q", l.OutputFormat, logconsole.Format)
	}
	if l.Output != config.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", l.Output, config.LoggerConsoleOutputStderr)
	}
	if !l.Colorized {
		t.Error("Colorized = false, want true")
	}
}

func TestLoggerDefaultsNonInteractive(t *testing.T) {
	l := appconfig.LoggerDefaults(false)

	if l.Level != config.LoggerLevelInfo {
		t.Errorf("Level = %q, want %q", l.Level, config.LoggerLevelInfo)
	}
	if l.OutputFormat != config.LogFormatText {
		t.Errorf("OutputFormat = %q, want %q", l.OutputFormat, config.LogFormatText)
	}
	if l.Output != config.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", l.Output, config.LoggerConsoleOutputStderr)
	}
}

func TestLoadSeedsLoggerDefaults(t *testing.T) {
	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(true))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Logger.Level != config.LoggerLevelWarning {
		t.Errorf("Level = %q, want %q", cfg.Logger.Level, config.LoggerLevelWarning)
	}
	if cfg.Logger.OutputFormat != logconsole.Format {
		t.Errorf("OutputFormat = %q, want %q", cfg.Logger.OutputFormat, logconsole.Format)
	}
	if cfg.Logger.Output != config.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", cfg.Logger.Output, config.LoggerConsoleOutputStderr)
	}
}

func TestLoadEnvOverridesLoggerSeed(t *testing.T) {
	t.Setenv("LOGGER__LEVEL", "debug")
	t.Setenv("LOGGER__OUTPUT_FORMAT", "json")

	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(true))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Logger.Level != config.LoggerLevelDebug {
		t.Errorf("Level = %q, want %q", cfg.Logger.Level, config.LoggerLevelDebug)
	}
	if cfg.Logger.OutputFormat != config.LogFormatJSON {
		t.Errorf("OutputFormat = %q, want %q", cfg.Logger.OutputFormat, config.LogFormatJSON)
	}
	if cfg.Logger.Output != config.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", cfg.Logger.Output, config.LoggerConsoleOutputStderr)
	}
}
