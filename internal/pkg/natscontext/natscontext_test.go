// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natscontext_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natscontext"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func byName(t *testing.T, contexts []entities.CliContext, name string) entities.CliContext {
	t.Helper()
	for _, c := range contexts {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("context %q not found in %v", name, contexts)
	return entities.CliContext{}
}

func TestDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	dir, err := natscontext.Dir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/cfg", "nats", "context"), dir)

	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/alice")
	dir, err = natscontext.Dir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/alice", ".config", "nats", "context"), dir)
}

func TestRead(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "nats", "context")
	kp, err := nkeys.CreateUser()
	require.NoError(t, err)
	seed, err := kp.Seed()
	require.NoError(t, err)

	writeFile(t, filepath.Join(root, "keys", "ops.creds"), "-----BEGIN NATS USER JWT-----\nJWT\n------END NATS USER JWT------\n")
	writeFile(t, filepath.Join(root, "keys", "svc.nk"), "# service key\n"+string(seed)+"\n")
	writeFile(t, filepath.Join(root, "tls", "ca.pem"), "CA PEM")
	writeFile(t, filepath.Join(root, "tls", "cert.pem"), "CERT PEM")
	writeFile(t, filepath.Join(root, "tls", "key.pem"), "KEY PEM")

	writeFile(t, filepath.Join(dir, "prod.json"), `{
		"description": "production",
		"url": "tls://a.example:4222, tls://b.example:4222",
		"creds": "`+filepath.Join(root, "keys", "ops.creds")+`",
		"ca": "`+filepath.Join(root, "tls", "ca.pem")+`",
		"cert": "`+filepath.Join(root, "tls", "cert.pem")+`",
		"key": "`+filepath.Join(root, "tls", "key.pem")+`",
		"tls_first": true,
		"jetstream_domain": "hub",
		"inbox_prefix": "_INBOX_ops"
	}`)
	writeFile(t, filepath.Join(dir, "svc.json"), `{"url": "nats://svc:4222", "nkey": "`+filepath.Join(root, "keys", "svc.nk")+`", "jetstream_api_prefix": "JS.A.API"}`)
	writeFile(t, filepath.Join(dir, "dev.json"), `{"user": "dev", "password": "pw"}`)
	writeFile(t, filepath.Join(dir, "tok.json"), `{"url": "nats://t:4222", "token": "s3cret"}`)
	writeFile(t, filepath.Join(dir, "broken.json"), `{"url": "nats://b:4222", "creds": "`+filepath.Join(root, "missing.creds")+`", "nsc": "nsc://op/acc/user", "socks_proxy": "socks5://p:1080"}`)
	writeFile(t, filepath.Join(dir, "garbage.json"), `{not json`)
	writeFile(t, filepath.Join(dir, "notes.txt"), `ignored`)
	writeFile(t, filepath.Join(root, "nats", "context.txt"), "svc\n")

	contexts, err := natscontext.Read(dir)
	require.NoError(t, err)
	require.Len(t, contexts, 6)

	prod := byName(t, contexts, "prod")
	c := prod.Connection
	assert.Equal(t, "production", *c.Description)
	assert.Equal(t, []string{"tls://a.example:4222", "tls://b.example:4222"}, c.URLs)
	assert.Equal(t, entities.AuthMethodCredentials, c.Auth.Method)
	assert.Contains(t, *c.Auth.Credentials, "NATS USER JWT")
	assert.Equal(t, "CA PEM", *c.TLS.CaCert)
	assert.Equal(t, "CERT PEM", *c.TLS.ClientCert)
	assert.Equal(t, "KEY PEM", *c.TLS.ClientKey)
	assert.True(t, c.TLS.TlsFirst)
	assert.Equal(t, "hub", *c.Connection.JetstreamDomain)
	assert.Equal(t, "_INBOX_ops", *c.Connection.InboxPrefix)
	assert.Empty(t, prod.Warnings)
	assert.False(t, prod.Selected)

	svc := byName(t, contexts, "svc")
	assert.True(t, svc.Selected)
	assert.Equal(t, entities.AuthMethodNKey, svc.Connection.Auth.Method)
	assert.Equal(t, string(seed), *svc.Connection.Auth.NkeySeed)
	assert.Equal(t, "JS.A.API", *svc.Connection.Connection.JetstreamAPIPrefix)

	dev := byName(t, contexts, "dev")
	assert.Equal(t, []string{"nats://127.0.0.1:4222"}, dev.Connection.URLs)
	assert.Equal(t, entities.AuthMethodUserPass, dev.Connection.Auth.Method)
	assert.Equal(t, "pw", *dev.Connection.Auth.Password)

	assert.Equal(t, "s3cret", *byName(t, contexts, "tok").Connection.Auth.Token)

	broken := byName(t, contexts, "broken")
	assert.Nil(t, broken.Connection.Auth)
	assert.Len(t, broken.Warnings, 3)

	garbage := byName(t, contexts, "garbage")
	assert.Nil(t, garbage.Connection)
	assert.NotEmpty(t, garbage.Warnings)
}

func TestRead_MissingDirectoryIsEmpty(t *testing.T) {
	contexts, err := natscontext.Read(filepath.Join(t.TempDir(), "nope"))
	require.NoError(t, err)
	assert.Empty(t, contexts)
}

func TestParse_UploadsNeverReadLocalFiles(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "server-side.creds")
	writeFile(t, secret, "do not read")

	contexts := natscontext.Parse([]entities.CliContextFile{
		{Name: "remote.json", Content: []byte(`{"url": "nats://r:4222", "creds": "` + secret + `", "user": "u", "password": "p"}`)},
	})

	require.Len(t, contexts, 1)
	c := contexts[0]
	assert.Equal(t, "remote", c.Name)
	assert.Equal(t, entities.AuthMethodUserPass, c.Connection.Auth.Method)
	assert.Nil(t, c.Connection.Auth.Credentials)
	require.Len(t, c.Warnings, 1)
	assert.Contains(t, c.Warnings[0], "server-side.creds")
}

func TestParse_UserJWTAndSeed(t *testing.T) {
	kp, err := nkeys.CreateUser()
	require.NoError(t, err)
	seed, err := kp.Seed()
	require.NoError(t, err)

	contexts := natscontext.Parse([]entities.CliContextFile{
		{Name: "jwt.json", Content: []byte(`{"url": "nats://a:4222", "user_jwt": "eyJ.jwt", "user_seed": "` + string(seed) + `"}`)},
		{Name: "lonely.json", Content: []byte(`{"url": "nats://a:4222", "user_jwt": "eyJ.jwt"}`)},
	})

	jwt := byName(t, contexts, "jwt")
	require.NotNil(t, jwt.Connection.Auth)
	assert.Equal(t, entities.AuthMethodCredentials, jwt.Connection.Auth.Method)
	assert.Equal(t, "eyJ.jwt", *jwt.Connection.Auth.JWT)
	assert.Equal(t, string(seed), *jwt.Connection.Auth.NkeySeed)

	lonely := byName(t, contexts, "lonely")
	assert.Nil(t, lonely.Connection.Auth)
	assert.Contains(t, lonely.Warnings, "the user JWT comes without a seed: add the seed or a credentials file to the connection")
}

func TestParse_KeepsWhatTheEditFormAccepts(t *testing.T) {
	contexts := natscontext.Parse([]entities.CliContextFile{
		{Name: "both.json", Content: []byte(`{"url": "nats://a:4222", "jetstream_domain": "hub", "jetstream_api_prefix": "JS.A.API"}`)},
		{Name: "inbox.json", Content: []byte(`{"url": "nats://a:4222", "inbox_prefix": "bad inbox *"}`)},
		{Name: "domain.json", Content: []byte(`{"url": "nats://a:4222", "jetstream_domain": "a.b"}`)},
		{Name: "prefix.json", Content: []byte(`{"url": "nats://a:4222", "jetstream_api_prefix": "JS..API"}`)},
		{Name: ".json", Content: []byte(`{"url": "nats://a:4222"}`)},
		{Name: "odd‮name.json", Content: []byte(`{"url": "nats://a:4222"}`)},
		{Name: "long.json", Content: []byte(`{"url": "nats://a:4222", "description": "` + strings.Repeat("d", 5000) + `"}`)},
	})

	both := byName(t, contexts, "both")
	require.NotNil(t, both.Connection)
	assert.Equal(t, "hub", *both.Connection.Connection.JetstreamDomain)
	assert.Nil(t, both.Connection.Connection.JetstreamAPIPrefix)
	assert.NotEmpty(t, both.Warnings)

	for _, name := range []string{"inbox", "domain", "prefix", "", "odd‮name"} {
		c := byName(t, contexts, name)
		assert.Nil(t, c.Connection, "%q is not importable", name)
		assert.NotEmpty(t, c.Warnings, name)
	}

	long := byName(t, contexts, "long")
	require.NotNil(t, long.Connection)
	assert.Len(t, *long.Connection.Description, 4096)
}

func TestParse_SameNameTwice(t *testing.T) {
	contexts := natscontext.Parse([]entities.CliContextFile{
		{Name: "dup.json", Content: []byte(`{"url": "nats://first:4222"}`)},
		{Name: "dup.json", Content: []byte(`{"url": "nats://second:4222"}`)},
	})

	require.Len(t, contexts, 2)
	require.NotNil(t, contexts[0].Connection)
	assert.Equal(t, []string{"nats://first:4222"}, contexts[0].Connection.URLs)
	assert.Nil(t, contexts[1].Connection)
	assert.NotEmpty(t, contexts[1].Warnings)
}
