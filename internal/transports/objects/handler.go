// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package objectstransport moves Object Store content over plain HTTP in a stream, so an object is not bound by the
// size of a single Connect message.
package objectstransport

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	grpchelpers "github.com/dmit-4884/natscope/internal/transports/grpc/helpers"
)

// Path is where the handler is mounted.
const Path = "/api/objects"

// errMissingTarget is the answer to a request that does not name an object.
var errMissingTarget = errors.New("connection, bucket and name are required")

// httpStatuses maps the status codes an object transfer fails with to HTTP statuses.
var httpStatuses = map[codes.Code]int{
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.NotFound:           http.StatusNotFound,
	codes.AlreadyExists:      http.StatusConflict,
	codes.FailedPrecondition: http.StatusPreconditionFailed,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
}

// Handler downloads an object on GET and uploads one on PUT; the connection, bucket and name query parameters name it.
type Handler struct {
	objects natssvc.ObjectStore
	logger  *slog.Logger
}

// New creates a Handler over the Object Store service.
func New(objects natssvc.ObjectStore) *Handler {
	return &Handler{objects: objects, logger: slog.Default().With(slogx.Module("transport:objects"))}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.download(w, r)
	case http.MethodPut:
		h.upload(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// target names the object a request is about.
type target struct {
	connection, bucket, name string
}

func readTarget(r *http.Request) (target, bool) {
	q := r.URL.Query()
	t := target{connection: q.Get("connection"), bucket: q.Get("bucket"), name: q.Get("name")}
	return t, t.connection != "" && t.bucket != "" && t.name != ""
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	t, ok := readTarget(r)
	if !ok {
		writeMessage(w, http.StatusBadRequest, errMissingTarget.Error())
		return
	}
	body, info, err := h.objects.OpenObject(r.Context(), t.connection, t.bucket, t.name)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer body.Close() //nolint:errcheck // read-only stream

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": t.name}))
	w.Header().Set("Content-Length", strconv.FormatUint(info.Size, 10))
	if _, err := io.Copy(w, body); err != nil {
		h.logger.WarnContext(r.Context(), "object download cut short", slog.String("bucket", t.bucket), slogx.Error(err))
	}
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	t, ok := readTarget(r)
	if !ok {
		writeMessage(w, http.StatusBadRequest, errMissingTarget.Error())
		return
	}
	meta := entities.ObjectMeta{Name: t.name, Description: r.URL.Query().Get("description")}
	if _, err := h.objects.PutObjectStream(r.Context(), t.connection, t.bucket, meta, r.Body, r.ContentLength); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	st := status.Convert(grpchelpers.StatusErrorConvert(r.Context(), err))
	code, ok := httpStatuses[st.Code()]
	if !ok {
		code = http.StatusInternalServerError
	}
	writeMessage(w, code, st.Message())
}

func writeMessage(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(struct { //nolint:errcheck // the status line is already out
		Message string `json:"message"`
	}{Message: message})
}
