// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"bufio"
	"context"
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
	"strconv"
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

// clientCert issues a client certificate and returns it and its key as PEM.
func (ca *testCA) clientCert(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "natscope client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
}

func (ca *testCA) pool() *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(ca.cert)
	return p
}

// fakeNATS greets with info and answers the client's CONNECT with reply, then PONGs every PING.
func fakeNATS(t *testing.T, info, reply string) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = conn.Write([]byte("INFO " + info + "\r\n"))
				r := bufio.NewReader(conn)
				replied := false
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if !strings.HasPrefix(line, "PING") {
						continue
					}
					if replied {
						_, _ = conn.Write([]byte("PONG\r\n"))
						continue
					}
					replied = true
					_, _ = conn.Write([]byte(reply + "\r\n"))
				}
			}()
		}
	}()
	return "nats://" + lis.Addr().String()
}

func deadAddr(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())
	return "nats://" + addr
}

func freePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := lis.Addr().(*net.TCPAddr).Port
	require.NoError(t, lis.Close())
	return port
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

func TestDiagnose_SeveralURLs(t *testing.T) {
	t.Parallel()
	good := startTestServer(t, nil)
	locked := startTestServer(t, func(o *server.Options) { o.Username, o.Password = "app", "right" })
	wrongPassword := &entities.AuthConfig{Method: entities.AuthMethodUserPass, Username: ptr.Wrap("app"), Password: ptr.Wrap("wrong")}

	t.Run("a dead first server does not hide a working second one", func(t *testing.T) {
		t.Parallel()
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{deadAddr(t), good}})
		require.True(t, res.Success, res.Error)
		assert.Equal(t, ok, check(t, res, entities.CheckStepTCP).Status)
	})

	t.Run("the server that got furthest explains the failure", func(t *testing.T) {
		t.Parallel()
		for range 5 {
			res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{deadAddr(t), locked}, Auth: wrongPassword})
			require.False(t, res.Success)
			auth := check(t, res, entities.CheckStepAuth)
			assert.Equal(t, failed, auth.Status)
			assert.Contains(t, auth.Hint, "user name and password")
			assert.Equal(t, ok, check(t, res, entities.CheckStepTCP).Status)
			assert.NotContains(t, res.Error, "no servers available")
		}
	})
}

func TestDiagnose_RunningOutOfTimeIsNotAFailure(t *testing.T) {
	t.Parallel()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 1500*time.Millisecond)
	defer cancel()

	res, err := NewDialer().TestConnection(ctx, &entities.TestConnectionRequest{URLs: []string{"nats://" + lis.Addr().String()}})
	require.NoError(t, err)

	assert.False(t, res.Success)
	assert.Equal(t, ok, check(t, res, entities.CheckStepTCP).Status)
	proto := check(t, res, entities.CheckStepProtocol)
	assert.Equal(t, skipped, proto.Status)
	assert.Contains(t, proto.Detail, "out of time")
}

func TestDiagnose_NegativeConnectTimeoutUsesTheDefault(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, nil)

	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, ConnectTimeout: new(-time.Second)})

	require.True(t, res.Success, res.Error)
}

func TestDiagnose_CredentialsTheServerDoesNotCheck(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, nil)

	res := diagnoseURL(t, &entities.TestConnectionRequest{
		URLs: []string{url},
		Auth: &entities.AuthConfig{Method: entities.AuthMethodToken, Token: ptr.Wrap("secret")},
	})

	require.True(t, res.Success, res.Error)
	auth := check(t, res, entities.CheckStepAuth)
	assert.Equal(t, warning, auth.Status)
	assert.Contains(t, auth.Detail, "not checked")
}

func TestDiagnose_ServerRefusals(t *testing.T) {
	t.Parallel()
	info := `{"server_id":"FAKE","version":"2.11.0","max_payload":1048576,"auth_required":true}`
	for name, tc := range map[string]struct{ reply, hint string }{
		"connection limit":         {"-ERR 'maximum connections exceeded'", "max_connections"},
		"account connection limit": {"-ERR 'maximum account active connections exceeded'", "account"},
		"authentication timeout":   {"-ERR 'Authentication Timeout'", "auth_timeout"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			res := diagnoseURL(t, &entities.TestConnectionRequest{
				URLs: []string{fakeNATS(t, info, tc.reply)},
				Auth: &entities.AuthConfig{Method: entities.AuthMethodToken, Token: ptr.Wrap("secret")},
			})
			require.False(t, res.Success)
			auth := check(t, res, entities.CheckStepAuth)
			assert.Equal(t, failed, auth.Status)
			assert.Contains(t, auth.Hint, tc.hint)
		})
	}
}

func TestDiagnose_ServerTextIsCleaned(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 500)
	info := `{"server_id":"FAKE","version":"2.11‮.0","server_name":"evil\u0007` + long + `","max_payload":1048576}`
	url := fakeNATS(t, info, "-ERR 'odd‮ "+long+"'")

	res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}})

	for _, c := range res.Checks {
		for _, text := range []string{c.Detail, c.Hint} {
			assert.NotContains(t, text, "‮", c.Step)
			assert.NotContains(t, text, "\u0007", c.Step)
			assert.Less(t, len(text), 300, c.Step)
		}
	}
	assert.Less(t, len(res.Error), 300)
}

func TestDiagnose_JetStreamDomainAnswersDisabled(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) { o.JetStream, o.StoreDir = true, t.TempDir() })

	res := diagnoseURL(t, &entities.TestConnectionRequest{
		URLs:       []string{url},
		Connection: &entities.ConnectionConfig{JetstreamDomain: ptr.Wrap("nowhere")},
	})

	js := check(t, res, entities.CheckStepJetStream)
	assert.NotContains(t, strings.ToLower(js.Detail), "jetstream not enabled")
	assert.Contains(t, js.Detail, "domain nowhere")
}

func TestDiagnose_MutualTLS(t *testing.T) {
	t.Parallel()
	ca := newTestCA(t)
	other := newTestCA(t)
	mtlsServer := func(clientAuth tls.ClientAuthType, maxVersion uint16) string {
		return startTestServer(t, func(o *server.Options) {
			o.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{ca.serverCert(t, 90*24*time.Hour, "127.0.0.1")},
				ClientAuth:   clientAuth,
				ClientCAs:    ca.pool(),
				MinVersion:   tls.VersionTLS12,
				MaxVersion:   maxVersion,
			}
			o.TLSVerify = clientAuth == tls.RequireAndVerifyClientCert
			o.TLSTimeout = 2
		})
	}
	versions := map[string]uint16{"TLS 1.2": tls.VersionTLS12, "TLS 1.3": tls.VersionTLS13}

	for name, version := range versions {
		t.Run(name+": no client certificate", func(t *testing.T) {
			t.Parallel()
			url := mtlsServer(tls.RequireAndVerifyClientCert, version)
			res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem}})
			require.False(t, res.Success)
			tlsCheck := check(t, res, entities.CheckStepTLS)
			assert.Equal(t, failed, tlsCheck.Status)
			assert.Contains(t, tlsCheck.Detail, "client certificate")
			assert.Equal(t, skipped, check(t, res, entities.CheckStepAuth).Status)
		})

		t.Run(name+": a client certificate the server does not trust", func(t *testing.T) {
			t.Parallel()
			url := mtlsServer(tls.RequireAndVerifyClientCert, version)
			certPEM, keyPEM := other.clientCert(t)
			res := diagnoseURL(t, &entities.TestConnectionRequest{
				URLs: []string{url},
				TLS:  &entities.TlsConfig{CaCert: &ca.pem, ClientCert: &certPEM, ClientKey: &keyPEM},
			})
			require.False(t, res.Success)
			tlsCheck := check(t, res, entities.CheckStepTLS)
			assert.Equal(t, failed, tlsCheck.Status)
			assert.Contains(t, tlsCheck.Detail, "rejected the client certificate")
			assert.Equal(t, skipped, check(t, res, entities.CheckStepAuth).Status)
		})

		t.Run(name+": a trusted client certificate", func(t *testing.T) {
			t.Parallel()
			url := mtlsServer(tls.RequireAndVerifyClientCert, version)
			certPEM, keyPEM := ca.clientCert(t)
			res := diagnoseURL(t, &entities.TestConnectionRequest{
				URLs: []string{url},
				TLS:  &entities.TlsConfig{CaCert: &ca.pem, ClientCert: &certPEM, ClientKey: &keyPEM},
			})
			require.True(t, res.Success, res.Error)
			assert.Equal(t, ok, check(t, res, entities.CheckStepTLS).Status)
		})
	}

	t.Run("an optional client certificate", func(t *testing.T) {
		t.Parallel()
		url := mtlsServer(tls.VerifyClientCertIfGiven, tls.VersionTLS13)
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem}})
		require.True(t, res.Success, res.Error)
		assert.Equal(t, warning, check(t, res, entities.CheckStepTLS).Status)
	})
}

func TestDiagnose_MoreTLS(t *testing.T) {
	t.Parallel()
	ca := newTestCA(t)

	t.Run("TLS first against a server that greets first", func(t *testing.T) {
		t.Parallel()
		url := startTestServer(t, func(o *server.Options) {
			o.TLSConfig = &tls.Config{Certificates: []tls.Certificate{ca.serverCert(t, 90*24*time.Hour, "127.0.0.1")}, MinVersion: tls.VersionTLS12}
			o.TLSTimeout = 2
		})
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{CaCert: &ca.pem, TlsFirst: true}})
		require.False(t, res.Success)
		tlsCheck := check(t, res, entities.CheckStepTLS)
		assert.Equal(t, failed, tlsCheck.Status)
		assert.Contains(t, tlsCheck.Hint, "TLS handshake first")
	})

	t.Run("an expired certificate with verification off", func(t *testing.T) {
		t.Parallel()
		url := startTestServer(t, func(o *server.Options) {
			o.TLSConfig = &tls.Config{Certificates: []tls.Certificate{ca.serverCert(t, -30*time.Minute, "127.0.0.1")}, MinVersion: tls.VersionTLS12}
			o.TLSTimeout = 2
		})
		res := diagnoseURL(t, &entities.TestConnectionRequest{URLs: []string{url}, TLS: &entities.TlsConfig{SkipVerify: true}})
		require.True(t, res.Success, res.Error)
		tlsCheck := check(t, res, entities.CheckStepTLS)
		assert.Equal(t, warning, tlsCheck.Status)
		assert.Contains(t, tlsCheck.Detail, "expired on")
	})

	t.Run("WebSocket with TLS settings", func(t *testing.T) {
		t.Parallel()
		port := freePort(t)
		startTestServer(t, func(o *server.Options) {
			o.Websocket = server.WebsocketOpts{
				Host:      "127.0.0.1",
				Port:      port,
				TLSConfig: &tls.Config{Certificates: []tls.Certificate{ca.serverCert(t, 90*24*time.Hour, "127.0.0.1")}, MinVersion: tls.VersionTLS12},
			}
		})
		res := diagnoseURL(t, &entities.TestConnectionRequest{
			URLs: []string{"ws://127.0.0.1:" + strconv.Itoa(port)},
			TLS:  &entities.TlsConfig{CaCert: &ca.pem},
		})
		require.True(t, res.Success, res.Error)
		assert.Equal(t, ok, check(t, res, entities.CheckStepTLS).Status)
	})

	t.Run("the TLS-first probe sends no client certificate", func(t *testing.T) {
		t.Parallel()
		presented := make(chan bool, 10)
		lis, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
			Certificates: []tls.Certificate{ca.serverCert(t, 90*24*time.Hour, "127.0.0.1")},
			ClientAuth:   tls.RequestClientCert,
			MinVersion:   tls.VersionTLS12,
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = lis.Close() })
		go func() {
			for {
				conn, err := lis.Accept()
				if err != nil {
					return
				}
				go func() {
					defer conn.Close()
					tc, isTLS := conn.(*tls.Conn)
					if !isTLS || tc.Handshake() != nil {
						return
					}
					presented <- len(tc.ConnectionState().PeerCertificates) > 0
				}()
			}
		}()
		certPEM, keyPEM := ca.clientCert(t)

		diagnoseURL(t, &entities.TestConnectionRequest{
			URLs: []string{"nats://" + lis.Addr().String()},
			TLS:  &entities.TlsConfig{CaCert: &ca.pem, ClientCert: &certPEM, ClientKey: &keyPEM},
		})

		select {
		case sent := <-presented:
			assert.False(t, sent, "the probe must not present the client certificate")
		case <-time.After(5 * time.Second):
			t.Fatal("the probe never completed a handshake")
		}
	})
}
