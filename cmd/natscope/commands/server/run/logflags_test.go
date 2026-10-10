// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/config/observability"

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
		{"unset flags keep the resolved defaults", "", "", observabilityconfig.LoggerLevelWarning, logconsole.Format, false},
		{"level flag wins", "debug", "", observabilityconfig.LoggerLevelDebug, logconsole.Format, false},
		{"format flag wins", "", "json", observabilityconfig.LoggerLevelWarning, observabilityconfig.LogFormatJSON, false},
		{"both flags", "info", "text", observabilityconfig.LoggerLevelInfo, observabilityconfig.LogFormatText, false},
		{"explicit error level is honored", "error", "", observabilityconfig.LoggerLevelError, logconsole.Format, false},
		{"unknown level is rejected", "loud", "", "", "", true},
		{"unknown format is rejected", "", "xml", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &observabilityconfig.Logger{Level: observabilityconfig.LoggerLevelWarning, OutputFormat: logconsole.Format}

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
