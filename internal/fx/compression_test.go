// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package fx

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func responseEncoding(t *testing.T, loopback bool, size int) string {
	t.Helper()
	handler := connect.NewUnaryHandler("/test.v1.Echo/Say",
		func(context.Context, *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
			return connect.NewResponse(wrapperspb.String(strings.Repeat("a", size))), nil
		},
		compressionOption(loopback),
	)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	body, err := proto.Marshal(wrapperspb.String("hi"))
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/test.v1.Echo/Say", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/proto")
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := srv.Client().Transport.RoundTrip(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return resp.Header.Get("Content-Encoding")
}

func TestCompressionOptions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		loopback bool
		size     int
		want     string
	}{
		{"loopback leaves a large response uncompressed", true, 64 << 10, ""},
		{"a remote listener compresses a large response", false, 64 << 10, "gzip"},
		{"a remote listener leaves a small response uncompressed", false, 100, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, responseEncoding(t, tc.loopback, tc.size))
		})
	}
}
