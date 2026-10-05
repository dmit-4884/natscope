// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package fx

import (
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
)

// TestReadsHostCliContexts_OnlyForTheLocalMachineAlone checks that any sign of other users turns host contexts off.
func TestReadsHostCliContexts_OnlyForTheLocalMachineAlone(t *testing.T) {
	cases := map[string]struct {
		cfg  *appconfig.Config
		want bool
	}{
		"loopback only":   {cfg: &appconfig.Config{}, want: true},
		"remote bind":     {cfg: &appconfig.Config{AllowRemote: true}, want: false},
		"extra host name": {cfg: &appconfig.Config{AllowedHosts: "natscope.example.com"}, want: false},
		"web auth":        {cfg: &appconfig.Config{WebAuth: &appconfig.WebAuthConfig{Username: "ops", Password: "secret"}}, want: false},
		"blank hosts":     {cfg: &appconfig.Config{AllowedHosts: " , "}, want: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := readsHostCliContexts(tc.cfg); got != tc.want {
				t.Fatalf("readsHostCliContexts() = %v, want %v", got, tc.want)
			}
		})
	}
}
