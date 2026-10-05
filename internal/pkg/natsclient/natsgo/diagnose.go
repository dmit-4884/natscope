// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"

	corectx "github.com/altessa-s/go-atlas/core/context"
)

const (
	checkTimeout      = 5 * time.Second
	greetingTimeout   = 3 * time.Second
	certWarningWindow = 14 * 24 * time.Hour
	maxGreeting       = 64 << 10
	hoursPerDay       = 24

	schemeTLS = "tls"
	schemeWS  = "ws"
	schemeWSS = "wss"
)

var allSteps = []entities.ConnectionCheckStep{
	entities.CheckStepDNS, entities.CheckStepTCP, entities.CheckStepProtocol,
	entities.CheckStepTLS, entities.CheckStepAuth, entities.CheckStepJetStream,
}

// inContainer reports whether Natscope runs inside a Docker or Podman container.
var inContainer = func() bool {
	for _, marker := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(marker); err == nil {
			return true
		}
	}
	return false
}

// serverInfo is the part of the NATS INFO greeting the diagnosis reports.
type serverInfo struct {
	Version      string `json:"version"`
	ServerName   string `json:"server_name"`
	TLSRequired  bool   `json:"tls_required"`
	TLSAvailable bool   `json:"tls_available"`
	AuthRequired bool   `json:"auth_required"`
	JetStream    bool   `json:"jetstream"`
}

type target struct {
	scheme, host, port string
}

func parseTarget(raw string) (target, error) {
	if !strings.Contains(raw, "://") {
		raw = "nats://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return target{}, err
	}
	t := target{scheme: strings.ToLower(u.Scheme), host: u.Hostname(), port: u.Port()}
	if t.port == "" {
		switch t.scheme {
		case schemeWS:
			t.port = "80"
		case schemeWSS:
			t.port = "443"
		default:
			t.port = "4222"
		}
	}
	return t, nil
}

func (t target) addr() string { return net.JoinHostPort(t.host, t.port) }

func (t target) websocket() bool { return t.scheme == schemeWS || t.scheme == schemeWSS }

// diagnosis collects the checks of one connection test.
type diagnosis struct {
	checks       map[entities.ConnectionCheckStep]entities.ConnectionCheck
	info         *serverInfo
	reached      bool
	stepTimeout  time.Duration
	greetTimeout time.Duration
}

// newDiagnosis bounds each network step by the connection's connect timeout.
func newDiagnosis(connectTimeout time.Duration) *diagnosis {
	return &diagnosis{
		checks:       map[entities.ConnectionCheckStep]entities.ConnectionCheck{},
		stepTimeout:  min(checkTimeout, connectTimeout),
		greetTimeout: min(greetingTimeout, connectTimeout),
	}
}

func (d *diagnosis) add(step entities.ConnectionCheckStep, status entities.ConnectionCheckStatus, detail, hint string, start time.Time) {
	d.checks[step] = entities.ConnectionCheck{
		Step: step, Status: status, Detail: detail, Hint: hint, DurationMs: time.Since(start).Milliseconds(),
	}
}

// result lists the checks in step order; a step that never ran is reported as not reached.
func (d *diagnosis) result() []entities.ConnectionCheck {
	out := make([]entities.ConnectionCheck, 0, len(allSteps))
	for _, step := range allSteps {
		c, ok := d.checks[step]
		if !ok {
			c = entities.ConnectionCheck{Step: step, Status: entities.CheckStatusSkipped, Detail: "Not reached"}
		}
		out = append(out, c)
	}
	return out
}

// diagnoseNetwork runs the DNS, TCP, protocol and TLS steps against one server URL.
func diagnoseNetwork(ctx context.Context, raw string, tlsCfg *entities.TlsConfig, connectTimeout time.Duration) *diagnosis {
	d := newDiagnosis(connectTimeout)
	t, err := parseTarget(raw)
	if err != nil {
		d.add(entities.CheckStepDNS, entities.CheckStatusFailed, "The server URL cannot be read: "+err.Error(), "", time.Now())
		return d
	}
	if !d.resolve(ctx, t) {
		return d
	}
	conn := d.dial(ctx, t)
	if conn == nil {
		return d
	}
	defer conn.Close()
	d.reached = d.greet(ctx, conn, t, tlsCfg)
	return d
}

func (d *diagnosis) resolve(ctx context.Context, t target) bool {
	start := time.Now()
	if net.ParseIP(t.host) != nil {
		d.add(entities.CheckStepDNS, entities.CheckStatusOK, t.host+" is an IP address", "", start)
		return true
	}
	ctx, cancel := corectx.WithMaxTimeout(ctx, d.stepTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, t.host)
	if err != nil {
		d.add(entities.CheckStepDNS, entities.CheckStatusFailed, fmt.Sprintf("%s does not resolve: %v", t.host, err), dnsHint(t.host), start)
		return false
	}
	d.add(entities.CheckStepDNS, entities.CheckStatusOK, fmt.Sprintf("%s resolves to %s", t.host, strings.Join(addrs, ", ")), "", start)
	return true
}

func dnsHint(host string) string {
	switch {
	case inContainer() && host == "host.docker.internal":
		return "On Linux, start the Natscope container with --add-host=host.docker.internal:host-gateway."
	case inContainer():
		return "Natscope runs in a container: use the name of a container on the same Docker network, or host.docker.internal for a server on the host."
	default:
		return "Check the host name, and that the machine running Natscope can resolve it."
	}
}

func (d *diagnosis) dial(ctx context.Context, t target) net.Conn {
	start := time.Now()
	dialer := net.Dialer{Timeout: d.stepTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", t.addr())
	if err != nil {
		d.add(entities.CheckStepTCP, entities.CheckStatusFailed, fmt.Sprintf("Cannot connect to %s: %v", t.addr(), err), tcpHint(t, err), start)
		return nil
	}
	d.add(entities.CheckStepTCP, entities.CheckStatusOK, "Connected to "+t.addr(), "", start)
	return conn
}

func tcpHint(t target, err error) string {
	if inContainer() && isLoopback(t.host) {
		return "Natscope runs in a container, where " + t.host + " is the container itself: use host.docker.internal to reach a server on the host."
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return fmt.Sprintf("Nothing listens on port %s. Is the NATS server running, and is %s its client port (4222 by default)?", t.port, t.port)
	}
	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		return "No answer: a firewall may drop the traffic, or the address is wrong."
	}
	return ""
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// greet reads the NATS greeting and runs the TLS handshake the server or the connection asks for.
func (d *diagnosis) greet(ctx context.Context, conn net.Conn, t target, cfg *entities.TlsConfig) bool {
	tlsConf, configured, err := buildTLSConfig(cfg)
	if err != nil {
		d.add(entities.CheckStepTLS, entities.CheckStatusFailed, err.Error(), "Fix the TLS material of the connection.", time.Now())
		return false
	}
	wantTLS := configured || t.scheme == schemeTLS || t.scheme == schemeWSS

	if t.websocket() {
		d.add(entities.CheckStepProtocol, entities.CheckStatusSkipped, "WebSocket: the connection itself checks the protocol", "", time.Now())
		if t.scheme == schemeWSS {
			return d.handshake(ctx, conn, t, tlsConf) != nil
		}
		d.add(entities.CheckStepTLS, entities.CheckStatusSkipped, "Not used", "", time.Now())
		return true
	}

	if cfg != nil && cfg.TlsFirst {
		tc := d.handshake(ctx, conn, t, tlsConf)
		return tc != nil && d.readInfo(ctx, tc, t, tlsConf, true)
	}
	if !d.readInfo(ctx, conn, t, tlsConf, false) {
		return false
	}
	if !d.info.TLSRequired && !wantTLS {
		d.add(entities.CheckStepTLS, entities.CheckStatusSkipped, "Not used: the server does not require TLS", "", time.Now())
		return true
	}
	if !d.info.TLSRequired && !d.info.TLSAvailable {
		d.add(entities.CheckStepTLS, entities.CheckStatusFailed, "The server does not offer TLS",
			"Remove the TLS settings and use nats://, or enable TLS on the server.", time.Now())
		return false
	}
	return d.handshake(ctx, conn, t, tlsConf) != nil
}

// readInfo runs the protocol step: the server must greet with INFO.
func (d *diagnosis) readInfo(ctx context.Context, conn net.Conn, t target, tlsConf *tls.Config, overTLS bool) bool {
	start := time.Now()
	line := readLine(conn, d.greetTimeout)
	if info, ok := strings.CutPrefix(line, "INFO "); ok {
		d.info = &serverInfo{}
		if err := json.Unmarshal([]byte(info), d.info); err != nil {
			d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, "The server greeting cannot be read", "", start)
			return false
		}
		d.add(entities.CheckStepProtocol, entities.CheckStatusOK, describeInfo(d.info), "", start)
		return true
	}

	switch {
	case line == "" && !overTLS && answersTLSFirst(ctx, t, tlsConf, d.stepTimeout):
		d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, "The server sends no NATS greeting: it waits for a TLS handshake first",
			"Turn on TLS handshake first in the TLS settings.", start)
	case line == "" && !overTLS && speaksHTTP(conn, d.greetTimeout):
		d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, "This port speaks HTTP, not the NATS protocol", httpHint, start)
	case strings.HasPrefix(line, "HTTP/"):
		d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, "This port speaks HTTP, not the NATS protocol", httpHint, start)
	case line == "":
		d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, "The server sent nothing",
			"Check that this is the client port of a NATS server, 4222 by default.", start)
	default:
		d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, "This is not a NATS server",
			"Check that this is the client port of a NATS server, 4222 by default.", start)
	}
	return false
}

const httpHint = "NATS clients use the client port, 4222 by default; 8222 is the HTTP monitoring port. " +
	"For a WebSocket listener, use ws:// or wss://."

// readLine reads the first line the peer sends, or "" when it sends nothing in time.
func readLine(conn net.Conn, timeout time.Duration) string {
	_ = conn.SetReadDeadline(time.Now().Add(timeout)) //nolint:errcheck // a failed deadline only makes the read block until the socket closes
	defer conn.SetReadDeadline(time.Time{})           //nolint:errcheck // best effort
	line, err := bufio.NewReader(io.LimitReader(conn, maxGreeting)).ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimRight(line, "\r\n")
}

// speaksHTTP sends a line an HTTP server answers with an error status.
func speaksHTTP(conn net.Conn, timeout time.Duration) bool {
	if _, err := conn.Write([]byte("PING\r\n\r\n")); err != nil {
		return false
	}
	return strings.HasPrefix(readLine(conn, timeout), "HTTP/")
}

// answersTLSFirst reports whether a fresh connection completes a TLS handshake right away.
func answersTLSFirst(ctx context.Context, t target, base *tls.Config, timeout time.Duration) bool {
	ctx, cancel := corectx.WithMaxTimeout(ctx, timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", t.addr())
	if err != nil {
		return false
	}
	defer conn.Close()
	cfg := base.Clone()
	cfg.ServerName = t.host
	cfg.InsecureSkipVerify = true //nolint:gosec // only probes whether the server talks TLS first; nothing is sent over it
	return tls.Client(conn, cfg).HandshakeContext(ctx) == nil
}

func describeInfo(info *serverInfo) string {
	parts := []string{"NATS " + info.Version}
	if info.ServerName != "" {
		parts[0] += " on " + info.ServerName
	}
	if info.TLSRequired {
		parts = append(parts, "TLS required")
	}
	if info.AuthRequired {
		parts = append(parts, "authentication required")
	}
	if info.JetStream {
		parts = append(parts, "JetStream on")
	}
	return strings.Join(parts, "; ")
}

// handshake runs the TLS step on conn and returns the TLS connection, or nil when it failed.
func (d *diagnosis) handshake(ctx context.Context, conn net.Conn, t target, base *tls.Config) *tls.Conn {
	start := time.Now()
	cfg := base.Clone()
	if cfg.ServerName == "" {
		cfg.ServerName = t.host
	}
	certs := cfg.Certificates
	cfg.Certificates = nil
	requested := false
	cfg.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
		requested = true
		if len(certs) > 0 {
			return &certs[0], nil
		}
		return &tls.Certificate{}, nil
	}

	ctx, cancel := corectx.WithMaxTimeout(ctx, d.stepTimeout)
	defer cancel()
	tc := tls.Client(conn, cfg)
	if err := tc.HandshakeContext(ctx); err != nil {
		detail, hint := explainTLSError(err)
		d.add(entities.CheckStepTLS, entities.CheckStatusFailed, detail, hint, start)
		return nil
	}

	state := tc.ConnectionState()
	leaf := state.PeerCertificates[0]
	status := entities.CheckStatusOK
	detail := fmt.Sprintf("%s, certificate for %s issued by %s, valid until %s",
		tls.VersionName(state.Version), certNames(leaf), leaf.Issuer.CommonName, leaf.NotAfter.Format(time.DateOnly))
	hint := ""
	if left := time.Until(leaf.NotAfter); left < certWarningWindow {
		status = entities.CheckStatusWarning
		detail = fmt.Sprintf("%s, certificate for %s expires in %d days, on %s",
			tls.VersionName(state.Version), certNames(leaf), max(int(left.Hours()/hoursPerDay), 0), leaf.NotAfter.Format(time.DateOnly))
		hint = "Renew the server certificate before it expires."
	}
	if cfg.InsecureSkipVerify {
		status = entities.CheckStatusWarning
		detail += "; not verified, because Skip certificate verification is on"
		hint = "Add the CA certificate and turn Skip certificate verification off outside development."
	}
	if requested && len(certs) == 0 {
		status = entities.CheckStatusFailed
		detail = "The server asks for a client certificate (mutual TLS)"
		hint = "Add a client certificate and key under TLS."
	}
	d.add(entities.CheckStepTLS, status, detail, hint, start)
	return tc
}

func certNames(c *x509.Certificate) string {
	names := slices.Clone(c.DNSNames)
	for _, ip := range c.IPAddresses {
		names = append(names, ip.String())
	}
	if len(names) == 0 {
		return c.Subject.CommonName
	}
	return strings.Join(names, ", ")
}

func explainTLSError(err error) (detail, hint string) {
	if hostErr, ok := errors.AsType[x509.HostnameError](err); ok {
		return fmt.Sprintf("The certificate is for %s, not %s", certNames(hostErr.Certificate), hostErr.Host),
			"Connect with a name the certificate lists, or reissue the certificate for this name."
	}
	if verifyErr, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
		if certs := verifyErr.UnverifiedCertificates; len(certs) > 0 && time.Now().After(certs[0].NotAfter) {
			return "The server certificate expired on " + certs[0].NotAfter.Format(time.DateOnly), "Renew the server certificate."
		}
		return "The server certificate is not trusted: " + strings.TrimPrefix(verifyErr.Err.Error(), "x509: "),
			"Add the CA certificate under TLS, or turn on Skip certificate verification for a development server."
	}
	if alert, ok := errors.AsType[tls.AlertError](err); ok {
		return "The server rejected the TLS handshake: " + alert.Error(),
			"If the server requires mutual TLS, add a client certificate and key under TLS."
	}
	return "The TLS handshake failed: " + err.Error(), ""
}

// authCheck reports the authentication step from the outcome of the full connection.
func authCheck(d *diagnosis, connErr error, auth *entities.AuthConfig, start time.Time) {
	method := entities.AuthMethodNone
	if auth != nil {
		method = auth.Method
	}
	switch {
	case connErr == nil && method == entities.AuthMethodNone:
		d.add(entities.CheckStepAuth, entities.CheckStatusOK, "No authentication required", "", start)
	case connErr == nil:
		d.add(entities.CheckStepAuth, entities.CheckStatusOK, "Authenticated with "+authLabels[method], "", start)
	case !d.reached:
		d.add(entities.CheckStepAuth, entities.CheckStatusSkipped, "Not reached", "", start)
	case errors.Is(connErr, nats.ErrAuthExpired), errors.Is(connErr, nats.ErrAccountAuthExpired):
		d.add(entities.CheckStepAuth, entities.CheckStatusFailed, sanitizeTestError(connErr), "The credentials have expired: issue new ones.", start)
	case errors.Is(connErr, nats.ErrAuthorization), errors.Is(connErr, nats.ErrAuthRevoked):
		d.add(entities.CheckStepAuth, entities.CheckStatusFailed, sanitizeTestError(connErr), authHints[method], start)
	default:
		d.add(entities.CheckStepAuth, entities.CheckStatusFailed, "The connection failed: "+sanitizeTestError(connErr), "", start)
	}
}

var authLabels = map[entities.AuthMethod]string{
	entities.AuthMethodNone:        "no credentials",
	entities.AuthMethodUserPass:    "user and password",
	entities.AuthMethodToken:       "a token",
	entities.AuthMethodNKey:        "an NKey",
	entities.AuthMethodCredentials: "a credentials file",
}

var authHints = map[entities.AuthMethod]string{
	entities.AuthMethodNone:        "The server requires authentication: choose an auth method.",
	entities.AuthMethodUserPass:    "Check the user name and password.",
	entities.AuthMethodToken:       "Check the token.",
	entities.AuthMethodNKey:        "The server does not know this NKey user.",
	entities.AuthMethodCredentials: "The credentials may belong to another account or operator, or have been revoked.",
}

// jetStreamCheck reports the account's JetStream in the configured domain or API prefix and whether it is on.
func jetStreamCheck(ctx context.Context, d *diagnosis, conn *nats.Conn, cfg *entities.ConnectionConfig) bool {
	start := time.Now()
	domain, prefix := jetStreamTarget(cfg)
	where := ""
	hint := ""
	switch {
	case domain != "":
		where = " in domain " + domain
		hint = "Check the JetStream domain: no JetStream answered in domain " + domain + "."
	case prefix != "":
		where = " through " + prefix
		hint = "Check the API prefix, and that the other account exports its JetStream API to yours."
	}

	js, err := newJetStream(conn, domain, prefix)
	if err != nil {
		d.add(entities.CheckStepJetStream, entities.CheckStatusFailed, err.Error(), hint, start)
		return false
	}
	ctx, cancel := corectx.WithMaxTimeout(ctx, checkTimeout)
	defer cancel()
	info, err := js.AccountInfo(ctx)
	switch {
	case err == nil:
		d.add(entities.CheckStepJetStream, entities.CheckStatusOK,
			fmt.Sprintf("Enabled%s: %d streams, %d consumers", where, info.Streams, info.Consumers), "", start)
		return true
	case where == "" && (errors.Is(err, jetstream.ErrJetStreamNotEnabled) || errors.Is(err, jetstream.ErrJetStreamNotEnabledForAccount)):
		d.add(entities.CheckStepJetStream, entities.CheckStatusWarning, "JetStream is not enabled for this account",
			"Streams, consumers, KV and Object Store need JetStream; core publish, subscribe and request work without it.", start)
	default:
		d.add(entities.CheckStepJetStream, entities.CheckStatusFailed, "JetStream did not answer"+where+": "+err.Error(), hint, start)
	}
	return false
}
