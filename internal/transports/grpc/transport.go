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
	server       *http.Server
	listener     net.Listener
	mux          *http.ServeMux
	logger       *slog.Logger
	address      string
	frontendFS   fs.FS
	csrf         *http.CrossOriginProtection
	connectOpt   []connect.HandlerOption
	authUser     string
	authPass     string
	loopback     bool
	allowedHosts map[string]struct{}
	baseCtx      context.Context //nolint:containedctx // canceled by GracefulStop
	cancelBase   context.CancelFunc
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
// embedded UI. Host must be localhost, an IP literal or in allowedHosts, unless basic auth
// guards a non-loopback bind with no allowedHosts.
func New(
	address string,
	frontendFS fs.FS,
	logger *slog.Logger,
	loopback bool,
	allowedHosts []string,
	connectOpts ...connect.HandlerOption,
) (*Transport, error) {
	csrf := http.NewCrossOriginProtection()
	// The Vite dev server (:5173) is only relevant to a loopback dev setup; a
	// remote deployment must not trust it for cross-origin state changes.
	if loopback {
		for _, o := range devTrustedOrigins {
			if err := csrf.AddTrustedOrigin(o); err != nil {
				logger.Warn("invalid trusted origin for CSRF protection",
					slog.String("origin", o), slogx.Error(err))
			}
		}
	}

	t := &Transport{
		mux:          http.NewServeMux(),
		logger:       logger,
		address:      address,
		frontendFS:   frontendFS,
		csrf:         csrf,
		connectOpt:   connectOpts,
		loopback:     loopback,
		allowedHosts: hostNameSet(allowedHosts),
	}
	return t, nil
}

// hostNameSet normalizes configured hosts to lowercase names without ports.
func hostNameSet(hosts []string) map[string]struct{} {
	set := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		if name := hostName(h); name != "" {
			set[name] = struct{}{}
		}
	}
	return set
}

// hostName strips the port and IPv6 brackets from a Host value and lowercases it.
func hostName(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(host, "["), "]"))
}

// hostAllowed reports whether host passes the DNS-rebinding check.
func (t *Transport) hostAllowed(host string) bool {
	name := hostName(host)
	if name == "localhost" || net.ParseIP(name) != nil {
		return true
	}
	if _, ok := t.allowedHosts[name]; ok {
		return true
	}
	return !t.loopback && t.authUser != "" && len(t.allowedHosts) == 0
}

// checkHost rejects requests whose Host fails hostAllowed before CSRF or auth run.
func (t *Transport) checkHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !t.hostAllowed(r.Host) {
			t.logger.Warn("rejected request with untrusted Host header",
				slog.String("host", r.Host),
				slog.String("remote", r.RemoteAddr),
				slog.String("path", r.URL.Path))
			http.Error(w, "invalid host", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
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

// Mount serves h at pattern behind the same Host check, auth and CSRF guards. Must be called before Start.
func (t *Transport) Mount(pattern string, h http.Handler) {
	t.mux.Handle(pattern, h)
}

// assetsPrefix holds Vite's hashed build output; a miss under it is a 404, not index.html.
const assetsPrefix = "/assets/"

// serveSPA serves embedded files for GET/HEAD; unknown paths fall through to index.html (404 if no dist).
func (t *Transport) serveSPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	isAsset := strings.HasPrefix(r.URL.Path, assetsPrefix)

	if f, err := t.frontendFS.Open(path); err == nil {
		_ = f.Close()
		// Vite emits content-hashed filenames under /assets, so they can be
		// cached indefinitely; everything else is revalidated.
		if isAsset {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		http.FileServer(http.FS(t.frontendFS)).ServeHTTP(w, r)
		return
	}

	if isAsset {
		http.NotFound(w, r)
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
//
//nolint:contextcheck // baseCtx outlives ctx
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
	handler = t.checkHost(handler)

	// baseCtx parents every request context and is canceled by GracefulStop, not by Listen's caller.
	t.baseCtx, t.cancelBase = context.WithCancel(context.Background())
	baseCtx := t.baseCtx
	t.server = &http.Server{
		Handler:   handler,
		Protocols: &protocols,
		BaseContext: func(net.Listener) context.Context {
			return baseCtx
		},
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
// complete or ctx fires. It cancels the request base context first so streaming handlers end.
func (t *Transport) GracefulStop(ctx context.Context) error {
	if t.server == nil {
		return nil
	}
	if t.cancelBase != nil {
		t.cancelBase()
	}
	return t.server.Shutdown(ctx)
}
