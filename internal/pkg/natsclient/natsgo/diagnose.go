// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"bufio"
	"bytes"
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
	"unicode"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"

	corectx "github.com/altessa-s/go-atlas/core/context"
)

const (
	checkTimeout      = 5 * time.Second
	greetingTimeout   = 3 * time.Second
	alertWaitFloor    = 250 * time.Millisecond
	alertWaitFactor   = 2
	certWarningWindow = 14 * 24 * time.Hour
	maxGreeting       = 64 << 10
	maxServerText     = 128
	hoursPerDay       = 24

	schemeNATS = "nats"
	schemeTLS  = "tls"
	schemeWS   = "ws"
	schemeWSS  = "wss"

	notReached = "Not reached"
	outOfTime  = "Not finished: the test ran out of time"
	notChecked = "Not checked: the test ran out of time"
)

var allSteps = []entities.ConnectionCheckStep{
	entities.CheckStepDNS, entities.CheckStepTCP, entities.CheckStepProtocol,
	entities.CheckStepTLS, entities.CheckStepAuth, entities.CheckStepJetStream,
}

var (
	errServerMaxConnections = errors.New("nats: " + nats.MAX_CONNECTIONS_ERR)
	errAuthTimeout          = errors.New("nats: authentication timeout")
)

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
	Version      string   `json:"version"`
	ServerName   string   `json:"server_name"`
	TLSRequired  bool     `json:"tls_required"`
	TLSAvailable bool     `json:"tls_available"`
	AuthRequired bool     `json:"auth_required"`
	JetStream    bool     `json:"jetstream"`
	ConnectURLs  []string `json:"connect_urls"`
}

type target struct {
	scheme, host, port string
}

func parseTarget(raw string) (target, error) {
	if !strings.Contains(raw, "://") {
		raw = schemeNATS + "://" + raw
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

// diagnosis collects the checks of one server URL.
type diagnosis struct {
	checks        map[entities.ConnectionCheckStep]entities.ConnectionCheck
	info          *serverInfo
	reached       bool
	outOfTime     bool
	tlsUsed       bool
	certRequested bool
	certSent      bool
	stepTimeout   time.Duration
	greetTimeout  time.Duration
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

// timedOut records a step the test's time budget cut short.
func (d *diagnosis) timedOut(step entities.ConnectionCheckStep, start time.Time) {
	d.outOfTime = true
	d.add(step, entities.CheckStatusSkipped, outOfTime, "", start)
}

// result lists the checks in step order; a step that never ran is reported as not reached or not checked in time.
func (d *diagnosis) result() []entities.ConnectionCheck {
	missing := notReached
	if d.outOfTime {
		missing = notChecked
	}
	out := make([]entities.ConnectionCheck, 0, len(allSteps))
	for _, step := range allSteps {
		c, ok := d.checks[step]
		if !ok {
			c = entities.ConnectionCheck{Step: step, Status: entities.CheckStatusSkipped, Detail: missing}
		}
		out = append(out, c)
	}
	return out
}

// progress counts the steps that passed, so the server that got furthest explains a failed test.
func (d *diagnosis) progress() int {
	n := 0
	for _, c := range d.checks {
		if c.Status == entities.CheckStatusOK || c.Status == entities.CheckStatusWarning {
			n++
		}
	}
	return n
}

// failure is the detail of the first failed step, which the test reports as its error.
func (d *diagnosis) failure() string {
	for _, c := range d.result() {
		if c.Status == entities.CheckStatusFailed {
			return c.Detail
		}
	}
	return "The test ran out of time"
}

// discovered lists the servers the greeting announced beyond the configured ones.
func (d *diagnosis) discovered(urls []string) []string {
	if d.info == nil {
		return nil
	}
	known := map[string]bool{}
	for _, raw := range urls {
		if t, err := parseTarget(raw); err == nil {
			known[t.addr()] = true
		}
	}
	scheme := schemeNATS
	if d.tlsUsed {
		scheme = schemeTLS
	}
	var out []string
	for _, hostPort := range d.info.ConnectURLs {
		if !known[hostPort] {
			out = append(out, scheme+"://"+serverText(hostPort))
		}
	}
	return out
}

// until is the earlier of now plus timeout and ctx's deadline.
func until(ctx context.Context, timeout time.Duration) time.Time {
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		return ctxDeadline
	}
	return deadline
}

// spent reports whether ctx is done or its deadline has passed, which a socket deadline can notice first.
func spent(ctx context.Context) bool {
	if ctx.Err() != nil {
		return true
	}
	deadline, ok := ctx.Deadline()
	return ok && !time.Now().Before(deadline)
}

// serverText drops control and formatting characters, bidirectional overrides among them, from text a server
// sent, and shortens it.
func serverText(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		if n == maxServerText {
			b.WriteString("…")
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

// diagnoseNetwork runs the DNS, TCP, protocol and TLS steps against one server URL.
func diagnoseNetwork(ctx context.Context, raw string, tlsCfg *entities.TlsConfig, connectTimeout time.Duration) *diagnosis {
	d := newDiagnosis(connectTimeout)
	t, err := parseTarget(raw)
	if err != nil {
		d.add(entities.CheckStepDNS, entities.CheckStatusFailed, "The server URL cannot be read", "Check the server URL.", time.Now())
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
	stepCtx, cancel := corectx.WithMaxTimeout(ctx, d.stepTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(stepCtx, t.host)
	switch {
	case err == nil:
		d.add(entities.CheckStepDNS, entities.CheckStatusOK, fmt.Sprintf("%s resolves to %s", t.host, strings.Join(addrs, ", ")), "", start)
		return true
	case spent(ctx):
		d.timedOut(entities.CheckStepDNS, start)
	default:
		d.add(entities.CheckStepDNS, entities.CheckStatusFailed, fmt.Sprintf("%s does not resolve: %v", t.host, err), dnsHint(t.host), start)
	}
	return false
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
	switch {
	case err == nil:
		d.add(entities.CheckStepTCP, entities.CheckStatusOK, "Connected to "+t.addr(), "", start)
		return conn
	case spent(ctx):
		d.timedOut(entities.CheckStepTCP, start)
	default:
		d.add(entities.CheckStepTCP, entities.CheckStatusFailed, fmt.Sprintf("Cannot connect to %s: %v", t.addr(), err), tcpHint(t, err), start)
	}
	return nil
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
	tlsFirst := cfg != nil && cfg.TlsFirst
	wantTLS := configured || tlsFirst || t.scheme == schemeTLS || t.scheme == schemeWSS

	if t.websocket() {
		d.add(entities.CheckStepProtocol, entities.CheckStatusSkipped, "WebSocket: the connection itself checks the protocol", "", time.Now())
		if wantTLS {
			return d.handshake(ctx, conn, t, tlsConf, true) != nil
		}
		d.add(entities.CheckStepTLS, entities.CheckStatusSkipped, "Not used", "", time.Now())
		return true
	}

	if tlsFirst {
		tc := d.handshake(ctx, conn, t, tlsConf, false)
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
	return d.handshake(ctx, conn, t, tlsConf, true) != nil
}

// readInfo runs the protocol step: the server must greet with INFO.
func (d *diagnosis) readInfo(ctx context.Context, conn net.Conn, t target, tlsConf *tls.Config, overTLS bool) bool {
	start := time.Now()
	stepCtx, cancel := corectx.WithMaxTimeout(ctx, d.stepTimeout)
	defer cancel()

	line, err := readLine(conn, until(stepCtx, d.greetTimeout))
	if info, ok := strings.CutPrefix(line, "INFO "); ok {
		d.info = &serverInfo{}
		if jsonErr := json.Unmarshal([]byte(info), d.info); jsonErr != nil {
			d.info = nil
			d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, "The server greeting cannot be read", "", start)
			return false
		}
		d.add(entities.CheckStepProtocol, entities.CheckStatusOK, describeInfo(d.info), "", start)
		return true
	}
	if overTLS && isTLSError(err) {
		d.tlsFailed(err, start)
		return false
	}

	detail, hint := unexpectedGreeting(stepCtx, conn, t, tlsConf, line, overTLS, d.greetTimeout)
	if spent(ctx) {
		d.timedOut(entities.CheckStepProtocol, start)
		return false
	}
	d.add(entities.CheckStepProtocol, entities.CheckStatusFailed, detail, hint, start)
	return false
}

const httpHint = "NATS clients use the client port, 4222 by default; 8222 is the HTTP monitoring port. " +
	"For a WebSocket listener, use ws:// or wss://."

// unexpectedGreeting explains what the port speaks when it did not greet with INFO.
func unexpectedGreeting(
	ctx context.Context, conn net.Conn, t target, tlsConf *tls.Config, line string, overTLS bool, greetTimeout time.Duration,
) (detail, hint string) {
	const notNATS = "Check that this is the client port of a NATS server, 4222 by default."
	switch {
	case line == "" && !overTLS && answersTLSFirst(ctx, t, tlsConf):
		return "The server sends no NATS greeting: it waits for a TLS handshake first", "Turn on TLS handshake first in the TLS settings."
	case line == "" && !overTLS && speaksHTTP(conn, until(ctx, greetTimeout)):
		return "This port speaks HTTP, not the NATS protocol", httpHint
	case strings.HasPrefix(line, "HTTP/"):
		return "This port speaks HTTP, not the NATS protocol", httpHint
	case line == "":
		return "The server sent nothing", notNATS
	default:
		return "This is not a NATS server", notNATS
	}
}

// readLine reads the first line the peer sends before deadline, or returns why it could not.
func readLine(conn net.Conn, deadline time.Time) (string, error) {
	_ = conn.SetReadDeadline(deadline)      //nolint:errcheck // a failed deadline only makes the read block until the socket closes
	defer conn.SetReadDeadline(time.Time{}) //nolint:errcheck // best effort
	line, err := bufio.NewReader(io.LimitReader(conn, maxGreeting)).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// speaksHTTP sends a line an HTTP server answers with an error status.
func speaksHTTP(conn net.Conn, deadline time.Time) bool {
	_ = conn.SetWriteDeadline(deadline) //nolint:errcheck // best effort
	if _, err := conn.Write([]byte("PING\r\n\r\n")); err != nil {
		return false
	}
	line, _ := readLine(conn, deadline) //nolint:errcheck // only the answer matters
	return strings.HasPrefix(line, "HTTP/")
}

// answersTLSFirst reports whether a fresh connection completes a TLS handshake right away. The probe presents no
// client certificate, as it trusts any server.
func answersTLSFirst(ctx context.Context, t target, base *tls.Config) bool {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", t.addr())
	if err != nil {
		return false
	}
	defer conn.Close()
	cfg := base.Clone()
	cfg.ServerName = t.host
	cfg.InsecureSkipVerify = true //nolint:gosec // only probes whether the server talks TLS first; nothing is sent over it
	cfg.Certificates = nil
	cfg.GetClientCertificate = nil
	return tls.Client(conn, cfg).HandshakeContext(ctx) == nil
}

func describeInfo(info *serverInfo) string {
	parts := []string{"NATS " + serverText(info.Version)}
	if name := serverText(info.ServerName); name != "" {
		parts[0] += " on " + name
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

// handshake runs the TLS step on conn and returns the TLS connection, or nil when it failed. With checkAlert, a
// TLS 1.3 server that asked for a client certificate gets a moment to reject it, which it does after the handshake.
func (d *diagnosis) handshake(ctx context.Context, conn net.Conn, t target, base *tls.Config, checkAlert bool) *tls.Conn {
	start := time.Now()
	cfg := base.Clone()
	if cfg.ServerName == "" {
		cfg.ServerName = t.host
	}
	certs := cfg.Certificates
	cfg.Certificates = nil
	d.certRequested, d.certSent = false, false
	cfg.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
		d.certRequested = true
		if len(certs) > 0 {
			d.certSent = true
			return &certs[0], nil
		}
		return &tls.Certificate{}, nil
	}

	stepCtx, cancel := corectx.WithMaxTimeout(ctx, d.stepTimeout)
	defer cancel()
	tc := tls.Client(conn, cfg)
	if err := tc.HandshakeContext(stepCtx); err != nil {
		if spent(ctx) {
			d.timedOut(entities.CheckStepTLS, start)
		} else {
			d.tlsFailed(err, start)
		}
		return nil
	}
	d.tlsUsed = true

	state := tc.ConnectionState()
	if checkAlert && d.certRequested && state.Version == tls.VersionTLS13 {
		if err := waitForAlert(tc, until(stepCtx, max(alertWaitFloor, alertWaitFactor*time.Since(start)))); err != nil {
			d.tlsFailed(err, start)
			return nil
		}
	}

	status, detail, hint := describeServerCert(state, cfg.InsecureSkipVerify)
	if d.certRequested && !d.certSent {
		status = entities.CheckStatusWarning
		detail += "; the server asks for a client certificate and accepted the connection without one"
		hint = "Add a client certificate and key under TLS if the server should know who connects."
	}
	d.add(entities.CheckStepTLS, status, detail, hint, start)
	return tc
}

// waitForAlert reads until deadline and returns the TLS alert the server sent, if any.
func waitForAlert(tc *tls.Conn, deadline time.Time) error {
	_ = tc.SetReadDeadline(deadline)      //nolint:errcheck // a failed deadline only makes the read block until the socket closes
	defer tc.SetReadDeadline(time.Time{}) //nolint:errcheck // best effort
	var one [1]byte
	if _, err := tc.Read(one[:]); isTLSError(err) {
		return err
	}
	return nil
}

func describeServerCert(state tls.ConnectionState, skipVerify bool) (entities.ConnectionCheckStatus, string, string) {
	version := tls.VersionName(state.Version)
	if len(state.PeerCertificates) == 0 {
		return entities.CheckStatusOK, version, ""
	}
	leaf := state.PeerCertificates[0]
	names, date := certNames(leaf), leaf.NotAfter.Format(time.DateOnly)
	status := entities.CheckStatusOK
	detail := fmt.Sprintf("%s, certificate for %s issued by %s, valid until %s", version, names, serverText(leaf.Issuer.CommonName), date)
	hint := ""
	switch left := time.Until(leaf.NotAfter); {
	case left < 0:
		status, hint = entities.CheckStatusWarning, "Renew the server certificate."
		detail = fmt.Sprintf("%s, certificate for %s expired on %s", version, names, date)
	case left < certWarningWindow:
		status, hint = entities.CheckStatusWarning, "Renew the server certificate before it expires."
		detail = fmt.Sprintf("%s, certificate for %s expires in %d days, on %s", version, names, int(left.Hours()/hoursPerDay), date)
	}
	if skipVerify {
		status = entities.CheckStatusWarning
		detail += "; not verified, because Skip certificate verification is on"
		if hint == "" {
			hint = "Add the CA certificate and turn Skip certificate verification off outside development."
		}
	}
	return status, detail, hint
}

func certNames(c *x509.Certificate) string {
	names := slices.Clone(c.DNSNames)
	for _, ip := range c.IPAddresses {
		names = append(names, ip.String())
	}
	if len(names) == 0 {
		return serverText(c.Subject.CommonName)
	}
	return serverText(strings.Join(names, ", "))
}

func (d *diagnosis) tlsFailed(err error, start time.Time) {
	detail, hint := explainTLSError(err, d.certRequested, d.certSent)
	d.add(entities.CheckStepTLS, entities.CheckStatusFailed, detail, hint, start)
}

func explainTLSError(err error, certRequested, certSent bool) (detail, hint string) {
	if hostErr, ok := errors.AsType[x509.HostnameError](err); ok {
		return fmt.Sprintf("The certificate is for %s, not %s", certNames(hostErr.Certificate), hostErr.Host),
			"Connect with a name the certificate lists, or reissue the certificate for this name."
	}
	if verifyErr, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
		if certs := verifyErr.UnverifiedCertificates; len(certs) > 0 && time.Now().After(certs[0].NotAfter) {
			return "The server certificate expired on " + certs[0].NotAfter.Format(time.DateOnly), "Renew the server certificate."
		}
		return "The server certificate is not trusted: " + serverText(strings.TrimPrefix(verifyErr.Err.Error(), "x509: ")),
			"Add the CA certificate under TLS, or turn on Skip certificate verification for a development server."
	}
	if alert, ok := remoteAlert(err); ok {
		switch {
		case certRequested && !certSent:
			return "The server requires a client certificate (mutual TLS)", "Add a client certificate and key under TLS."
		case certRequested:
			return "The server rejected the client certificate: " + alert,
				"Use a client certificate issued by a CA the server trusts, and check that it has not expired."
		default:
			return "The server rejected the TLS handshake: " + alert, "Check that the server and Natscope share a TLS version and cipher suite."
		}
	}
	if header, ok := errors.AsType[tls.RecordHeaderError](err); ok {
		if bytes.HasPrefix(header.RecordHeader[:], []byte("INFO")) {
			return "The server greeted in plain text instead of answering the TLS handshake",
				"Turn off TLS handshake first: this server starts TLS after its greeting."
		}
		return "The server does not answer with TLS", "Check that this port serves TLS."
	}
	return "The TLS handshake failed: " + serverText(err.Error()), ""
}

// remoteAlert returns the TLS alert the server sent, as it reaches the client after a TCP handshake.
func remoteAlert(err error) (string, bool) {
	if opErr, ok := errors.AsType[*net.OpError](err); ok && opErr.Op == "remote error" && opErr.Err != nil {
		return strings.TrimPrefix(opErr.Err.Error(), "tls: "), true
	}
	if alert, ok := errors.AsType[tls.AlertError](err); ok {
		return strings.TrimPrefix(alert.Error(), "tls: "), true
	}
	return "", false
}

func isTLSError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := remoteAlert(err); ok {
		return true
	}
	var (
		header    tls.RecordHeaderError
		verify    *tls.CertificateVerificationError
		host      x509.HostnameError
		authority x509.UnknownAuthorityError
	)
	return errors.As(err, &header) || errors.As(err, &verify) || errors.As(err, &host) || errors.As(err, &authority)
}

// connect runs the authentication step: a connection to this one server with the test's options.
func (d *diagnosis) connect(ctx context.Context, url string, opts []nats.Option, auth *entities.AuthConfig) *nats.Conn {
	start := time.Now()
	conn, err := nats.Connect(url, opts...)
	d.authCheck(ctx, err, auth, start)
	if err != nil {
		return nil
	}
	return conn
}

// authCheck reports the authentication step from the outcome of the connection.
func (d *diagnosis) authCheck(ctx context.Context, connErr error, auth *entities.AuthConfig, start time.Time) {
	method := entities.AuthMethodNone
	if auth != nil {
		method = auth.Method
	}
	fail := func(detail, hint string) {
		d.add(entities.CheckStepAuth, entities.CheckStatusFailed, detail, hint, start)
	}
	switch {
	case connErr == nil && method == entities.AuthMethodNone:
		d.add(entities.CheckStepAuth, entities.CheckStatusOK, "No authentication required", "", start)
	case connErr == nil && d.info != nil && !d.info.AuthRequired:
		d.add(entities.CheckStepAuth, entities.CheckStatusWarning, "Connected, but the credentials were not checked: the server does not require authentication",
			"Anyone can connect to this server; it ignores "+authLabels[method]+".", start)
	case connErr == nil:
		d.add(entities.CheckStepAuth, entities.CheckStatusOK, "Authenticated with "+authLabels[method], "", start)
	case spent(ctx):
		d.timedOut(entities.CheckStepAuth, start)
	case isTLSError(connErr):
		d.tlsFailed(connErr, start)
		d.add(entities.CheckStepAuth, entities.CheckStatusSkipped, notReached, "", start)
	case errors.Is(connErr, nats.ErrAuthExpired), errors.Is(connErr, nats.ErrAccountAuthExpired):
		fail(sanitizeTestError(connErr), "The credentials have expired: issue new ones.")
	case errors.Is(connErr, nats.ErrAuthorization), errors.Is(connErr, nats.ErrAuthRevoked):
		fail(sanitizeTestError(connErr), authHints[method])
	case errors.Is(connErr, nats.ErrMaxConnectionsExceeded), errors.Is(connErr, errServerMaxConnections):
		fail("The server refuses new connections: it has reached its connection limit", "Close idle clients, or raise max_connections on the server.")
	case errors.Is(connErr, nats.ErrMaxAccountConnectionsExceeded):
		fail("The account has reached its connection limit", "Close idle clients of this account, or raise the account's connection limit.")
	case errors.Is(connErr, errAuthTimeout):
		fail("The server stopped waiting for the credentials", "Check auth_timeout on the server: a slow link or a slow auth callout service can exceed it.")
	default:
		fail("The connection failed: "+sanitizeTestError(connErr), "")
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
	stepCtx, cancel := corectx.WithMaxTimeout(ctx, checkTimeout)
	defer cancel()
	info, err := js.AccountInfo(stepCtx)
	const withoutJetStream = "Streams, consumers, KV and Object Store need JetStream; core publish, subscribe and request work without it."
	switch {
	case err == nil:
		d.add(entities.CheckStepJetStream, entities.CheckStatusOK,
			fmt.Sprintf("Enabled%s: %d streams, %d consumers", where, info.Streams, info.Consumers), "", start)
		return true
	case errors.Is(err, jetstream.ErrJetStreamNotEnabledForAccount):
		d.add(entities.CheckStepJetStream, entities.CheckStatusWarning, "JetStream is not enabled for this account"+where, withoutJetStream, start)
	case errors.Is(err, jetstream.ErrJetStreamNotEnabled) && where == "":
		d.add(entities.CheckStepJetStream, entities.CheckStatusWarning, "JetStream is not enabled for this account", withoutJetStream, start)
	case errors.Is(err, jetstream.ErrJetStreamNotEnabled):
		d.add(entities.CheckStepJetStream, entities.CheckStatusFailed, "No JetStream answered"+where, hint, start)
	case spent(ctx):
		d.timedOut(entities.CheckStepJetStream, start)
	default:
		d.add(entities.CheckStepJetStream, entities.CheckStatusFailed, "JetStream did not answer"+where+": "+serverText(err.Error()), hint, start)
	}
	return false
}
