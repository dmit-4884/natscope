// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package objectstransport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
)

type fakeObjects struct {
	natssvc.ObjectStore
	content  string
	openErr  error
	putErr   error
	put      string
	putMeta  entities.ObjectMeta
	putSize  int64
	putWhere [2]string
}

func (f *fakeObjects) OpenObject(_ context.Context, _, _, name string) (io.ReadCloser, *entities.ObjectInfo, error) {
	if f.openErr != nil {
		return nil, nil, f.openErr
	}
	return io.NopCloser(strings.NewReader(f.content)), &entities.ObjectInfo{Name: name, Size: uint64(len(f.content))}, nil
}

func (f *fakeObjects) PutObjectStream(
	_ context.Context, connectionID, bucket string, meta entities.ObjectMeta, r io.Reader, size int64,
) (*entities.ObjectInfo, error) {
	if f.putErr != nil {
		return nil, f.putErr
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	f.put, f.putMeta, f.putSize, f.putWhere = string(data), meta, size, [2]string{connectionID, bucket}
	return &entities.ObjectInfo{Name: meta.Name, Size: uint64(len(data))}, nil
}

func serve(t *testing.T, objects *fakeObjects, method, query, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, Path+"?"+query, strings.NewReader(body))
	rec := httptest.NewRecorder()
	New(objects).ServeHTTP(rec, req)
	return rec
}

func objectQuery(name string) string {
	return url.Values{"connection": {"conn-1"}, "bucket": {"BIG"}, "name": {name}}.Encode()
}

func TestHandler_DownloadsAnObjectAsAnAttachment(t *testing.T) {
	t.Parallel()

	rec := serve(t, &fakeObjects{content: "payload"}, http.MethodGet, objectQuery("reports/q3 é.bin"), "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "payload", rec.Body.String())
	assert.Equal(t, "7", rec.Header().Get("Content-Length"))
	assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename*=utf-8''reports%2Fq3%20%C3%A9.bin`, rec.Header().Get("Content-Disposition"))
}

func TestHandler_UploadsTheRequestBody(t *testing.T) {
	t.Parallel()
	objects := &fakeObjects{}
	query := objectQuery("dump.bin") + "&description=" + url.QueryEscape("nightly dump")

	rec := serve(t, objects, http.MethodPut, query, "big body")

	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "big body", objects.put)
	assert.Equal(t, int64(len("big body")), objects.putSize)
	assert.Equal(t, entities.ObjectMeta{Name: "dump.bin", Description: "nightly dump"}, objects.putMeta)
	assert.Equal(t, [2]string{"conn-1", "BIG"}, objects.putWhere)
}

func TestHandler_ReportsFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		objects *fakeObjects
		method  string
		query   string
		code    int
	}{
		{"a missing name", &fakeObjects{}, http.MethodGet, "connection=conn-1&bucket=BIG", http.StatusBadRequest},
		{"an unknown object", &fakeObjects{openErr: errs.ErrObjectNotFound}, http.MethodGet, objectQuery("x"), http.StatusNotFound},
		{"a read-only connection", &fakeObjects{putErr: errs.ErrConnectionReadOnly}, http.MethodPut, objectQuery("x"), http.StatusPreconditionFailed},
		{"another method", &fakeObjects{}, http.MethodDelete, objectQuery("x"), http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := serve(t, tc.objects, tc.method, tc.query, "")

			assert.Equal(t, tc.code, rec.Code)
			if tc.code != http.StatusMethodNotAllowed {
				var body struct{ Message string }
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotEmpty(t, body.Message)
			}
		})
	}
}
