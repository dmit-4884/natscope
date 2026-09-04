// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/config"

	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
)

func TestApplyLogFlags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		level      string
		format     string
		wantLevel  string
		wantFormat string
		wantErr    bool
	}{
		{"unset flags keep the resolved defaults", "", "", config.LoggerLevelWarning, logconsole.Format, false},
		{"level flag wins", "debug", "", config.LoggerLevelDebug, logconsole.Format, false},
		{"format flag wins", "", "json", config.LoggerLevelWarning, config.LogFormatJSON, false},
		{"both flags", "info", "text", config.LoggerLevelInfo, config.LogFormatText, false},
		{"explicit error level is honored", "error", "", config.LoggerLevelError, logconsole.Format, false},
		{"unknown level is rejected", "loud", "", "", "", true},
		{"unknown format is rejected", "", "xml", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Logger{Level: config.LoggerLevelWarning, OutputFormat: logconsole.Format}

			err := applyLogFlags(cfg, tc.level, tc.format)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantLevel, cfg.Level)
			require.Equal(t, tc.wantFormat, cfg.OutputFormat)
		})
	}
}
