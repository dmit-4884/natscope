// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig_test

import (
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
)

// BindsLoopback must classify every address form correctly; a false positive
// here would let the fail-closed remote-bind gate be bypassed.
func TestBindsLoopback(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:4280", true},
		{"localhost:4280", true},
		{"[::1]:4280", true},
		{"[::ffff:127.0.0.1]:4280", true},
		{"0.0.0.0:4280", false},
		{":4280", false},
		{"[::]:4280", false},
		{"192.168.1.5:4280", false},
		{"10.0.0.9:4280", false},
		{"not-an-address", false},
	}
	for _, tc := range cases {
		c := &appconfig.Config{GRPCWebAddress: tc.addr}
		if got := c.BindsLoopback(); got != tc.want {
			t.Errorf("BindsLoopback(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

func TestWebAuthEnabled(t *testing.T) {
	cases := []struct {
		name string
		cfg  *appconfig.WebAuthConfig
		want bool
	}{
		{"nil", nil, false},
		{"both set", &appconfig.WebAuthConfig{Username: "u", Password: "p"}, true},
		{"empty password", &appconfig.WebAuthConfig{Username: "u", Password: ""}, false},
		{"empty username", &appconfig.WebAuthConfig{Username: "", Password: "p"}, false},
		{"whitespace password", &appconfig.WebAuthConfig{Username: "u", Password: "   "}, false},
		{"whitespace username", &appconfig.WebAuthConfig{Username: "\t", Password: "p"}, false},
	}
	for _, tc := range cases {
		if got := tc.cfg.Enabled(); got != tc.want {
			t.Errorf("%s: Enabled() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
