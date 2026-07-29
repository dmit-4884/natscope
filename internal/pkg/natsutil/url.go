// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"net/url"
	"strings"

	"github.com/dmit-4884/natscope/internal/errs"
)

const maskedUserinfo = "***"

// URLCredentials are the credentials carried in a NATS server URL's userinfo.
// Follows nats.go: "user:pass@" is user/password, a bare "token@" is a token.
type URLCredentials struct {
	Username string
	Password string
	Token    string
}

// SplitCredentials removes the userinfo from every URL and returns the cleaned
// URLs plus the single credential set they carried, or nil when none did. URLs
// embedding different credentials yield [errs.ErrConnectionURLCredentialsMixed].
func SplitCredentials(urls []string) ([]string, *URLCredentials, error) {
	if len(urls) == 0 {
		return urls, nil, nil
	}

	out := make([]string, len(urls))
	var found *URLCredentials
	for i, raw := range urls {
		cleaned, creds := splitOne(raw)
		out[i] = cleaned
		if creds == nil {
			continue
		}
		if found != nil && *found != *creds {
			return nil, nil, errs.ErrConnectionURLCredentialsMixed
		}
		found = creds
	}
	return out, found, nil
}

// StripCredentials returns the URLs with any embedded userinfo removed. Unlike
// [SplitCredentials] it never fails.
func StripCredentials(urls []string) []string {
	if len(urls) == 0 {
		return urls
	}
	out := make([]string, len(urls))
	for i, raw := range urls {
		out[i], _ = splitOne(raw)
	}
	return out
}

// MaskURL replaces a URL's userinfo with "***" so it is safe to log. Operates on
// the raw string so input net/url rejects is masked too: this must fail closed.
func MaskURL(raw string) string {
	if !strings.Contains(raw, "@") {
		return raw
	}
	return maskAuthority(raw)
}

// MaskURLs masks every URL and joins them comma-separated for use as one log field.
func MaskURLs(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	masked := make([]string, len(urls))
	for i, raw := range urls {
		masked[i] = MaskURL(raw)
	}
	return strings.Join(masked, ",")
}

// splitOne strips the userinfo from one URL. A URL without userinfo, or one that
// does not parse, is returned unchanged: validating URLs is the caller's job.
func splitOne(raw string) (string, *URLCredentials) {
	if !strings.Contains(raw, "@") {
		return raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw, nil
	}

	username := u.User.Username()
	password, hasPassword := u.User.Password()
	u.User = nil
	cleaned := u.String()

	switch {
	case hasPassword:
		return cleaned, &URLCredentials{Username: username, Password: password}
	case username != "":
		return cleaned, &URLCredentials{Token: username}
	default:
		return cleaned, nil
	}
}

// maskAuthority keys off the last "@" before any path/query, so a token
// containing "@" cannot leak its tail and a path-only "@" is left alone.
func maskAuthority(raw string) string {
	const sep = "://"

	scheme := strings.Index(raw, sep)
	if scheme == -1 {
		if at := strings.LastIndexByte(raw, '@'); at >= 0 {
			return maskedUserinfo + "@" + raw[at+1:]
		}
		return raw
	}

	rest := raw[scheme+len(sep):]
	authEnd := len(rest)
	if j := strings.IndexAny(rest, "/?#"); j != -1 {
		authEnd = j
	}
	at := strings.LastIndexByte(rest[:authEnd], '@')
	if at == -1 {
		return raw
	}
	return raw[:scheme+len(sep)] + maskedUserinfo + "@" + rest[at+1:]
}
