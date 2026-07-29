// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/errs"
)

func TestSplitCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		urls      []string
		wantURLs  []string
		wantCreds *URLCredentials
	}{
		{
			name:     "NoUserinfo",
			urls:     []string{"nats://localhost:4222"},
			wantURLs: []string{"nats://localhost:4222"},
		},
		{
			name:      "UserPassword",
			urls:      []string{"nats://bob:s3cr3t@localhost:4222"},
			wantURLs:  []string{"nats://localhost:4222"},
			wantCreds: &URLCredentials{Username: "bob", Password: "s3cr3t"},
		},
		{
			name:      "BareTokenIsToken",
			urls:      []string{"nats://tok3n@localhost:4222"},
			wantURLs:  []string{"nats://localhost:4222"},
			wantCreds: &URLCredentials{Token: "tok3n"},
		},
		{
			name:      "EmptyPasswordStillUserPass",
			urls:      []string{"nats://bob:@localhost:4222"},
			wantURLs:  []string{"nats://localhost:4222"},
			wantCreds: &URLCredentials{Username: "bob"},
		},
		{
			name:     "EmptyUserinfoCarriesNothing",
			urls:     []string{"nats://@localhost:4222"},
			wantURLs: []string{"nats://localhost:4222"},
		},
		{
			name:      "PercentEncodedIsDecoded",
			urls:      []string{"nats://bob:p%40ss%3Aword@localhost:4222"},
			wantURLs:  []string{"nats://localhost:4222"},
			wantCreds: &URLCredentials{Username: "bob", Password: "p@ss:word"},
		},
		{
			name: "SameCredentialsAcrossURLs",
			urls: []string{
				"nats://bob:s3cr3t@h1:4222",
				"nats://bob:s3cr3t@h2:4222",
			},
			wantURLs:  []string{"nats://h1:4222", "nats://h2:4222"},
			wantCreds: &URLCredentials{Username: "bob", Password: "s3cr3t"},
		},
		{
			name: "CredentialsOnOnlyOneURL",
			urls: []string{
				"nats://h1:4222",
				"nats://bob:s3cr3t@h2:4222",
			},
			wantURLs:  []string{"nats://h1:4222", "nats://h2:4222"},
			wantCreds: &URLCredentials{Username: "bob", Password: "s3cr3t"},
		},
		{
			name:      "TLSScheme",
			urls:      []string{"tls://bob:s3cr3t@h1:4222"},
			wantURLs:  []string{"tls://h1:4222"},
			wantCreds: &URLCredentials{Username: "bob", Password: "s3cr3t"},
		},
		{
			name:      "WebsocketSchemeWithPath",
			urls:      []string{"wss://bob:s3cr3t@h1:443/nats"},
			wantURLs:  []string{"wss://h1:443/nats"},
			wantCreds: &URLCredentials{Username: "bob", Password: "s3cr3t"},
		},
		{
			name:     "Empty",
			urls:     nil,
			wantURLs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotURLs, gotCreds, err := SplitCredentials(tt.urls)

			require.NoError(t, err)
			assert.Equal(t, tt.wantURLs, gotURLs)
			assert.Equal(t, tt.wantCreds, gotCreds)
		})
	}
}

func TestSplitCredentialsMixed(t *testing.T) {
	t.Parallel()

	_, _, err := SplitCredentials([]string{
		"nats://bob:s3cr3t@h1:4222",
		"nats://eve:hunter2@h2:4222",
	})

	require.ErrorIs(t, err, errs.ErrConnectionURLCredentialsMixed)
}

func TestSplitCredentialsDoesNotAliasInput(t *testing.T) {
	t.Parallel()

	in := []string{"nats://bob:s3cr3t@h1:4222"}
	out, _, err := SplitCredentials(in)

	require.NoError(t, err)
	assert.Equal(t, []string{"nats://bob:s3cr3t@h1:4222"}, in, "input must not be mutated")
	assert.Equal(t, []string{"nats://h1:4222"}, out)
}

func TestStripCredentials(t *testing.T) {
	t.Parallel()

	got := StripCredentials([]string{
		"nats://bob:s3cr3t@h1:4222",
		"nats://h2:4222",
		"nats://tok3n@h3:4222",
	})

	assert.Equal(t, []string{"nats://h1:4222", "nats://h2:4222", "nats://h3:4222"}, got)
}

func TestStripCredentialsNeverFailsOnMixed(t *testing.T) {
	t.Parallel()

	got := StripCredentials([]string{
		"nats://bob:s3cr3t@h1:4222",
		"nats://eve:hunter2@h2:4222",
	})

	assert.Equal(t, []string{"nats://h1:4222", "nats://h2:4222"}, got)
}

func TestMaskURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "NoUserinfo", raw: "nats://localhost:4222", want: "nats://localhost:4222"},
		{name: "UserPassword", raw: "nats://bob:s3cr3t@h1:4222", want: "nats://***@h1:4222"},
		{name: "BareToken", raw: "nats://tok3n@h1:4222", want: "nats://***@h1:4222"},
		{name: "WithPath", raw: "wss://bob:s3cr3t@h1:443/nats", want: "wss://***@h1:443/nats"},
		{name: "SchemeLess", raw: "bob:s3cr3t@h1:4222", want: "***@h1:4222"},
		{name: "AtInPathOnlyIsLeftAlone", raw: "wss://h1:443/a@b", want: "wss://h1:443/a@b"},
		{name: "AtInTokenKeepsHostIntact", raw: "nats://to@ken@h1:4222", want: "nats://***@h1:4222"},
		{name: "Empty", raw: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MaskURL(tt.raw))
		})
	}
}

// TestMaskURLFailsClosed pins the property that actually matters: whatever the
// input shape, no secret survives masking.
func TestMaskURLFailsClosed(t *testing.T) {
	t.Parallel()

	const secret = "s3cr3t"
	raws := []string{
		"nats://bob:" + secret + "@h1:4222",
		"nats://" + secret + "@h1:4222",
		"nats://bob:" + secret + "@h1:4222/path?x=1",
		"nats://bob:" + secret + "@@h1:4222",
		"nats://bob:" + secret + "@h1:4222|garbage",
		"nats://bob:" + secret + "@h\x7fost:4222",
		"://bob:" + secret + "@h1:4222",
		"bob:" + secret + "@h1:4222",
	}

	for _, raw := range raws {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			assert.NotContains(t, MaskURL(raw), secret)
		})
	}
}

func TestMaskURLs(t *testing.T) {
	t.Parallel()

	got := MaskURLs([]string{"nats://bob:s3cr3t@h1:4222", "nats://h2:4222"})

	assert.Equal(t, "nats://***@h1:4222,nats://h2:4222", got)
	assert.False(t, strings.Contains(got, "s3cr3t"))
}
