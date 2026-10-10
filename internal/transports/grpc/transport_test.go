// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package grpc

import "testing"

func TestHostAllowed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		loopback     bool
		authUser     string
		allowedHosts []string
		host         string
		want         bool
	}{
		{name: "localhost on bound port", loopback: true, host: "localhost:4280", want: true},
		{name: "localhost via tunnel port", loopback: true, host: "localhost:9000", want: true},
		{name: "ipv4 literal", loopback: true, host: "127.0.0.1:4280", want: true},
		{name: "ipv6 literal", loopback: true, host: "[::1]:4280", want: true},
		{name: "rebinding name on loopback", loopback: true, host: "evil.example:4280", want: false},
		{name: "unlisted name on loopback", loopback: true, allowedHosts: []string{"other.example"}, host: "evil.example:4280", want: false},
		{name: "configured name on loopback", loopback: true, allowedHosts: []string{"natscope.local"}, host: "natscope.local:4280", want: true},
		{name: "wide insecure bind rejects names", host: "evil.example:4280", want: false},
		{name: "wide insecure bind accepts published localhost", host: "localhost:8080", want: true},
		{name: "wide bind accepts configured name", allowedHosts: []string{"Natscope.Example.com:443"}, host: "natscope.example.com", want: true},
		{name: "wide auth bind accepts any name", authUser: "admin", host: "natscope.example.com", want: true},
		{name: "wide auth bind honors explicit allowlist", authUser: "admin", allowedHosts: []string{"natscope.example.com"}, host: "evil.example", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tr := &Transport{loopback: tc.loopback, authUser: tc.authUser, allowedHosts: hostNameSet(tc.allowedHosts)}
			if got := tr.hostAllowed(tc.host); got != tc.want {
				t.Errorf("hostAllowed(%q) = %v, want %v", tc.host, got, tc.want)
			}
		})
	}
}
