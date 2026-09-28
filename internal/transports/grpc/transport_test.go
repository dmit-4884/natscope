// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package grpc

import "testing"

func TestBuildAllowedHosts(t *testing.T) {
	cases := []struct {
		name       string
		address    string
		loopback   bool
		extraHosts []string
		wantHosts  []string
	}{
		{
			name:      "loopback ignores extra hosts",
			address:   "127.0.0.1:4280",
			loopback:  true,
			wantHosts: []string{"127.0.0.1:4280", "[::1]:4280", "localhost:4280"},
		},
		{
			name:      "wide bind accepts the bound address itself",
			address:   "0.0.0.0:4280",
			loopback:  false,
			wantHosts: []string{"0.0.0.0:4280"},
		},
		{
			name:       "wide bind adds configured extra hosts with the bound port",
			address:    "0.0.0.0:4280",
			loopback:   false,
			extraHosts: []string{"example.com", "other.example:9999"},
			wantHosts:  []string{"0.0.0.0:4280", "example.com:4280", "other.example:9999"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildAllowedHosts(tc.address, tc.loopback, tc.extraHosts)
			if len(got) != len(tc.wantHosts) {
				t.Fatalf("buildAllowedHosts() = %v, want %v", got, tc.wantHosts)
			}
			for _, want := range tc.wantHosts {
				if _, ok := got[want]; !ok {
					t.Errorf("buildAllowedHosts() missing %q; got %v", want, got)
				}
			}
		})
	}
}

func TestBuildAllowedHosts_LoopbackWithoutPortRejectsEverything(t *testing.T) {
	// No port to pin the allowlist to; fail closed rather than accept any Host.
	got := buildAllowedHosts("127.0.0.1", true, nil)
	if len(got) != 0 {
		t.Errorf("buildAllowedHosts() = %v, want empty", got)
	}
}
