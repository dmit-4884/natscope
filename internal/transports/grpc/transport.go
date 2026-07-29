// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package grpc owns the unified Connect-RPC transport: one HTTP/2+h2c listener
// serving Connect, gRPC, and gRPC-Web natively off the same mux as the embedded
// SPA.
package grpc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// Handler registers a Connect handler onto the shared mux, returning its route
// prefix and http.Handler.
type Handler interface {
	HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

// Transport owns the long-lived HTTP listener and mux; lifecycle managed by fx
// Start/Stop hooks.
type Transport struct {
	server     *http.Server
	listener   net.Listener
	mux        *http.ServeMux
	logger     *slog.Logger
	address    string
	frontendFS fs.FS
	csrf       *http.CrossOriginProtection
	connectOpt []connect.HandlerOption
	authUser   string
	authPass   string
}

// Server hardening limits. Read/Write timeouts are deliberately absent — live
// subscriptions hold the response open indefinitely — so only the header phase
// and idle keep-alives are bounded.
const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 120 * time.Second
	maxHeaderBytes    = 1 << 20 // 1 MiB
)

// devTrustedOrigins allow cross-origin state-changing requests; cover the Vite
// dev server (production is same-origin).
var devTrustedOrigins = []string{
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

// New creates a Connect transport; frontendFS may be nil when built without an
// embedded UI.
func New(
	address string,
	frontendFS fs.FS,
	logger *slog.Logger,
	trustDevOrigins bool,
	connectOpts ...connect.HandlerOption,
) (*Transport, error) {
	csrf := http.NewCrossOriginProtection()
	// The Vite dev server (:5173) is only relevant to a loopback dev setup; a
	// remote deployment must not trust it for cross-origin state changes.
	if trustDevOrigins {
		for _, o := range devTrustedOrigins {
			if err := csrf.AddTrustedOrigin(o); err != nil {
				logger.Warn("invalid trusted origin for CSRF protection",
					slog.String("origin", o), slogx.Error(err))
			}
		}
	}

	t := &Transport{
		mux:        http.NewServeMux(),
		logger:     logger,
		address:    address,
		frontendFS: frontendFS,
		csrf:       csrf,
		connectOpt: connectOpts,
	}
	return t, nil
}

// RegisterHandlers mounts every Connect handler then installs the SPA fallback
// last. Must be called before Start.
func (t *Transport) RegisterHandlers(handlers []Handler) {
	for _, h := range handlers {
		path, handler := h.HTTPHandler(t.connectOpt...)
		t.mux.Handle(path, handler)
	}
	t.mux.HandleFunc("/", t.serveSPA)
}

// serveSPA serves embedded files as-is; unknown paths fall through to
// index.html (404 if no dist embedded).
func (t *Transport) serveSPA(w http.ResponseWriter, r *http.Request) {
	if t.frontendFS == nil {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path
	if path == "/" {
		path = "index.html"
	} else if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if f, err := t.frontendFS.Open(path); err == nil {
		_ = f.Close()
		// Vite emits content-hashed filenames under /assets, so they can be
		// cached indefinitely; everything else is revalidated.
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		http.FileServer(http.FS(t.frontendFS)).ServeHTTP(w, r)
		return
	}

	indexFile, err := fs.ReadFile(t.frontendFS, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// The SPA shell must never be cached stale across an upgrade.
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexFile) //nolint:errcheck // SPA fallback; client disconnect mid-write is benign
}

// Listen binds the TCP socket and builds the http.Server; split from Serve so a
// port-in-use error fails fx startup synchronously instead of on a goroutine.
func (t *Transport) Listen(ctx context.Context) error {
	if t.listener != nil {
		return nil // already listening (idempotent)
	}
	lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", t.address)
	if err != nil {
		return coreerrs.WrapOperationWithContext(err, "listen", t.address)
	}
	t.listener = lis

	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	handler := t.csrf.Handler(t.mux)
	if t.authUser != "" {
		handler = t.basicAuth(handler)
	}
	t.server = &http.Server{
		Handler:   handler,
		Protocols: &protocols,
		// Read/Write timeouts stay zero on purpose: live subscriptions are
		// long-lived streams. Header limits still bound slow-header clients.
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
	return nil
}

// SetBasicAuth guards every route (SPA, RPC, streaming) behind HTTP basic
// auth. Must be called before Listen.
func (t *Transport) SetBasicAuth(username, password string) {
	t.authUser, t.authPass = username, password
}

// authFailureDelay slows online password guessing without per-client state.
const authFailureDelay = 300 * time.Millisecond

// basicAuth enforces credentials with constant-time comparison; hashing first
// hides length differences from the timing side channel. Failed attempts are
// logged and briefly delayed to blunt brute-force attempts.
func (t *Transport) basicAuth(next http.Handler) http.Handler {
	wantUser := sha256.Sum256([]byte(t.authUser))
	wantPass := sha256.Sum256([]byte(t.authPass))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if ok {
			gotUser := sha256.Sum256([]byte(user))
			gotPass := sha256.Sum256([]byte(pass))
			userOK := subtle.ConstantTimeCompare(gotUser[:], wantUser[:]) == 1
			passOK := subtle.ConstantTimeCompare(gotPass[:], wantPass[:]) == 1
			if userOK && passOK {
				next.ServeHTTP(w, r)
				return
			}
		}
		t.logger.Warn("rejected unauthenticated request",
			slog.String("remote", r.RemoteAddr),
			slog.String("path", r.URL.Path))
		time.Sleep(authFailureDelay)
		w.Header().Set("WWW-Authenticate", `Basic realm="natscope", charset="UTF-8"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// Serve blocks running http.Server.Serve until graceful stop; calls Listen
// first (which loses the synchronous-fail property — fine for tests).
func (t *Transport) Serve() error {
	if err := t.Listen(context.Background()); err != nil {
		return err
	}
	t.logger.Info("connect transport started", slog.String("address", t.address))
	if err := t.server.Serve(t.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// GracefulStop shuts the HTTP server down, returning when in-flight requests
// complete or ctx fires.
func (t *Transport) GracefulStop(ctx context.Context) error {
	if t.server == nil {
		return nil
	}
	return t.server.Shutdown(ctx)
}
