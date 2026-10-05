// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
)

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  string
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "natscope test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return &testCA{cert: cert, key: key, pem: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))}
}

// serverCert issues a certificate for names, valid for the given time.
func (ca *testCA) serverCert(t *testing.T, validFor time.Duration, names ...string) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: names[0]},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(validFor),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, n := range names {
		if ip := net.ParseIP(n); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, n)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func startTestServer(t *testing.T, configure func(*server.Options)) string {
	t.Helper()
	opts := &server.Options{Host: "127.0.0.1", Port: -1, NoLog: true, NoSigs: true}
	if configure != nil {
		configure(opts)
	}
	srv, err := server.NewServer(opts)
	require.NoError(t, err)
	go srv.Start()
	require.True(t, srv.ReadyForConnections(10*time.Second))
	t.Cleanup(srv.Shutdown)
	return srv.ClientURL()
}

func diagnoseURL(t *testing.T, in *entities.TestConnectionRequest) *entities.TestConnectionResult {
	t.Helper()
	res, err := NewDialer().TestConnection(t.Context(), in)
	require.NoError(t, err)
	return res
}

func check(t *testing.T, res *entities.TestConnectionResult, step entities.ConnectionCheckStep) entities.ConnectionCheck {
	t.Helper()
	for _, c := range res.Checks {
		if c.Step == step {
			return c
		}
	}
	t.Fatalf("no %v check in %+v", step, res.Checks)
	return entities.ConnectionCheck{}
}

func statuses(res *entities.TestConnectionResult) []entities.ConnectionCheckStatus {
	out := make([]entities.ConnectionCheckStatus, 0, len(res.Checks))
	for _, c := range res.Checks {
		out = append(out, c.Status)
	}
	return out
}

const (
	ok      = entities.CheckStatusOK
	warning = entities.CheckStatusWarning
	failed  = entities.CheckStatusFailed
	skipped = entities.CheckStatusSkipped
)

func TestDiagnose_HealthyServerPassesEveryStep(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) { o.JetStream, o.StoreDir = true, t.TempDir() })

	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}})

	require.True(t, res.Success)
	assert.Equal(t, []entities.ConnectionCheckStatus{ok, ok, ok, skipped, ok, ok}, statuses(res))
	assert.Contains(t, check(t, res, entities.CheckStepProtocol).Detail, "NATS")
	assert.True(t, res.JetstreamEnabled)
}

func TestDiagnose_NothingListening(t *testing.T) {
	t.Parallel()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())

	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{"nats://" + addr}})

	assert.False(t, res.Success)
	assert.Equal(t, []entities.ConnectionCheckStatus{ok, failed, skipped, skipped, skipped, skipped}, statuses(res))
	assert.Contains(t, check(t, res, entities.CheckStepTCP).Hint, "Is the NATS server running")
}

func TestDiagnose_UnknownHost(t *testing.T) {
	t.Parallel()
	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{"nats://no-such-host.invalid:4222"}})

	assert.False(t, res.Success)
	assert.Equal(t, failed, check(t, res, entities.CheckStepDNS).Status)
	assert.Equal(t, skipped, check(t, res, entities.CheckStepTCP).Status)
}

func TestDiagnose_HTTPPort(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(srv.Close)

	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{"nats://" + strings.TrimPrefix(srv.URL, "http://")}})

	assert.False(t, res.Success)
	proto := check(t, res, entities.CheckStepProtocol)
	assert.Equal(t, failed, proto.Status)
	assert.Contains(t, proto.Hint, "8222")
}

func TestDiagnose_WrongPassword(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) { o.Username, o.Password = "app", "right" })

	res := diagnoseURL(t, &entities.TestConnectionRequest{
		URLs: []string{url},
		Auth: &entities.AuthConfig{Method: entities.AuthMethodUserPass, Username: ptr.Wrap("app"), Password: ptr.Wrap("wrong")},
	})

	assert.False(t, res.Success)
	assert.Contains(t, check(t, res, entities.CheckStepProtocol).Detail, "authentication required")
	auth := check(t, res, entities.CheckStepAuth)
	assert.Equal(t, failed, auth.Status)
	assert.Contains(t, auth.Hint, "user name and password")
	assert.Equal(t, skipped, check(t, res, entities.CheckStepJetStream).Status)
}

func TestDiagnose_NoJetStream(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, nil)

	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}})

	require.True(t, res.Success)
	assert.Equal(t, warning, check(t, res, entities.CheckStepJetStream).Status)
	assert.False(t, res.JetstreamEnabled)
}

func TestDiagnose_JetStreamDomainNobodyServes(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) { o.JetStream, o.StoreDir = true, t.TempDir() })

	res := diagnoseURL(t, &entities.TestConnectionRequest{
		URLs:       []string{url},
		Connection: &entities.ConnectionConfig{JetstreamDomain: ptr.Wrap("nowhere")},
	})

	require.True(t, res.Success)
	js := check(t, res, entities.CheckStepJetStream)
	assert.Equal(t, failed, js.Status)
	assert.Contains(t, js.Hint, "domain")
}

func TestDiagnose_TLS(t *testing.T) {
	t.Parallel()
	ca := newTestCA(t)
	tlsServer := func(cert tls.Certificate, first bool) string {
		return startTestServer(t, func(o *server.Options) {
			o.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
			o.TLSTimeout = 2
			o.TLSHandshakeFirst = first
		})
	}

	t.Run("a trusted certificate passes and shows its expiry", func(t *testing.T) {
		t.Parallel()
		url := tlsServer(ca.serverCert(t, 90*24*time.Hour, "127.0.0.1"), false)
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem}})
		require.True(t, res.Success, res.Error)
		tlsCheck := check(t, res, entities.CheckStepTLS)
		assert.Equal(t, ok, tlsCheck.Status)
		assert.Contains(t, tlsCheck.Detail, "valid until")
	})

	t.Run("an unknown CA asks for the CA certificate", func(t *testing.T) {
		t.Parallel()
		url := tlsServer(ca.serverCert(t, 90*24*time.Hour, "127.0.0.1"), false)
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}})
		assert.False(t, res.Success)
		tlsCheck := check(t, res, entities.CheckStepTLS)
		assert.Equal(t, failed, tlsCheck.Status)
		assert.Contains(t, tlsCheck.Hint, "CA certificate")
	})

	t.Run("a certificate for another name says so", func(t *testing.T) {
		t.Parallel()
		url := tlsServer(ca.serverCert(t, 90*24*time.Hour, "other.example"), false)
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem}})
		assert.False(t, res.Success)
		tlsCheck := check(t, res, entities.CheckStepTLS)
		assert.Equal(t, failed, tlsCheck.Status)
		assert.Contains(t, tlsCheck.Detail, "other.example")
	})

	t.Run("a certificate about to expire is a warning", func(t *testing.T) {
		t.Parallel()
		url := tlsServer(ca.serverCert(t, 3*24*time.Hour, "127.0.0.1"), false)
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem}})
		require.True(t, res.Success, res.Error)
		tlsCheck := check(t, res, entities.CheckStepTLS)
		assert.Equal(t, warning, tlsCheck.Status)
		assert.Contains(t, tlsCheck.Detail, "expires in")
	})

	t.Run("a TLS-first server without TLS first", func(t *testing.T) {
		t.Parallel()
		url := tlsServer(ca.serverCert(t, 90*24*time.Hour, "127.0.0.1"), true)
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem}})
		assert.False(t, res.Success)
		proto := check(t, res, entities.CheckStepProtocol)
		assert.Equal(t, failed, proto.Status)
		assert.Contains(t, proto.Hint, "TLS handshake first")
	})

	t.Run("a TLS-first server with TLS first", func(t *testing.T) {
		t.Parallel()
		url := tlsServer(ca.serverCert(t, 90*24*time.Hour, "127.0.0.1"), true)
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem, TlsFirst: true}})
		require.True(t, res.Success, res.Error)
		assert.Equal(t, ok, check(t, res, entities.CheckStepTLS).Status)
		assert.Equal(t, ok, check(t, res, entities.CheckStepProtocol).Status)
	})
}

func TestDiagnose_LoopbackInsideAContainerPointsAtTheHost(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())
	restore := inContainer
	inContainer = func() bool { return true }
	t.Cleanup(func() { inContainer = restore })

	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{"nats://" + addr}})

	assert.Contains(t, check(t, res, entities.CheckStepTCP).Hint, "host.docker.internal")
}
