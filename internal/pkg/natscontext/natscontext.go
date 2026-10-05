// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natscontext

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

const (
	contextExt = ".json"

	// maxReferencedFile bounds a credentials, key or certificate file a context points at.
	maxReferencedFile = 1 << 20

	// maxName and maxDescription are the lengths a saved connection accepts.
	maxName        = 256
	maxDescription = 4096
)

// cliContext is the part of a nats CLI context file Natscope understands.
type cliContext struct {
	Description      string `json:"description"`
	URL              string `json:"url"`
	Token            string `json:"token"`
	User             string `json:"user"`
	Password         string `json:"password"`
	Creds            string `json:"creds"`
	NKey             string `json:"nkey"`
	UserJWT          string `json:"user_jwt"`
	UserSeed         string `json:"user_seed"`
	Cert             string `json:"cert"`
	Key              string `json:"key"`
	CA               string `json:"ca"`
	NSC              string `json:"nsc"`
	JetStreamDomain  string `json:"jetstream_domain"`
	JetStreamAPI     string `json:"jetstream_api_prefix"`
	InboxPrefix      string `json:"inbox_prefix"`
	SocksProxy       string `json:"socks_proxy"`
	WindowsCertStore string `json:"windows_cert_store"`
	TLSFirst         bool   `json:"tls_first"`
}

// readFunc reads a file a context points at; nil when the files are out of reach.
type readFunc func(path string) ([]byte, error)

// Dir returns the nats CLI context directory: $XDG_CONFIG_HOME/nats/context, else ~/.config/nats/context.
func Dir() (string, error) {
	if cfg := os.Getenv("XDG_CONFIG_HOME"); cfg != "" {
		return filepath.Join(cfg, "nats", "context"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", coreerrs.WrapOperation(err, "find home directory")
	}
	return filepath.Join(home, ".config", "nats", "context"), nil
}

// Read reads every context in dir, with the files they point at; a missing dir has no contexts.
func Read(dir string) ([]entities.CliContext, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "read nats context directory")
	}

	selected := selectedContext(filepath.Dir(dir))
	contexts := make([]entities.CliContext, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != contextExt {
			continue
		}
		content, readErr := readLimited(filepath.Join(dir, e.Name()))
		c := parse(e.Name(), content, readErr, readReferenced)
		c.Selected = c.Name == selected
		contexts = append(contexts, c)
	}
	return contexts, nil
}

// Parse translates uploaded context files; the files they point at are reported, never read from this host. A file
// whose name an earlier one already took is not importable.
func Parse(files []entities.CliContextFile) []entities.CliContext {
	contexts := make([]entities.CliContext, 0, len(files))
	seen := map[string]bool{}
	for _, f := range files {
		c := parse(f.Name, f.Content, nil, nil)
		if seen[c.Name] {
			c.Connection = nil
			c.Warnings = append(c.Warnings, "another uploaded file has the same name")
		}
		seen[c.Name] = true
		contexts = append(contexts, c)
	}
	return contexts
}

func selectedContext(natsDir string) string {
	content, err := readLimited(filepath.Join(natsDir, "context.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func parse(fileName string, content []byte, readErr error, read readFunc) entities.CliContext {
	c := entities.CliContext{Name: strings.TrimSuffix(filepath.Base(fileName), contextExt)}
	if readErr != nil {
		c.Warnings = append(c.Warnings, fmt.Sprintf("the context file could not be read: %v", readErr))
		return c
	}
	var cli cliContext
	if err := json.Unmarshal(content, &cli); err != nil {
		c.Warnings = append(c.Warnings, "this is not a nats CLI context: "+err.Error())
		return c
	}

	t := translator{read: read}
	conn := &entities.SavedConnectionCreate{
		Name:        c.Name,
		Description: ptr.WrapNonZero(t.description(cli.Description)),
		URLs:        urls(cli.URL),
		Auth:        t.auth(cli),
		TLS:         t.tls(cli),
		Connection:  t.connection(cli),
	}
	if cli.NSC != "" {
		t.warn("nsc references are not supported: add the user's credentials to the connection")
	}
	if cli.SocksProxy != "" {
		t.warn("SOCKS proxies are not supported")
	}
	if cli.WindowsCertStore != "" {
		t.warn("the Windows certificate store is not supported: add the certificate files to the connection")
	}
	c.Warnings = t.warnings
	if problem := unusable(c.Name, conn.Connection); problem != "" {
		c.Warnings = append(c.Warnings, problem)
		return c
	}
	c.Connection = conn
	return c
}

// unusable names why a context would make a connection the edit form refuses to save, or returns "".
func unusable(name string, cfg *entities.ConnectionConfig) string {
	switch {
	case strings.TrimSpace(name) == "":
		return "the context has no name"
	case strings.ContainsFunc(name, func(r rune) bool { return unicode.In(r, unicode.Cc, unicode.Cf) }):
		return "the context name holds control or formatting characters"
	case utf8.RuneCountInString(name) > maxName:
		return fmt.Sprintf("the context name is longer than %d characters", maxName)
	case cfg == nil:
		return ""
	}
	if domain := ptr.Unwrap(cfg.JetstreamDomain, ""); strings.ContainsAny(domain, " \t\r\n\f.*>") {
		return fmt.Sprintf("the JetStream domain %q may not contain spaces, dots, * or >", domain)
	}
	if prefix := ptr.Unwrap(cfg.JetstreamAPIPrefix, ""); prefix != "" && !plainSubject(prefix) {
		return fmt.Sprintf("the JetStream API prefix %q is not a subject without wildcards", prefix)
	}
	if inbox := ptr.Unwrap(cfg.InboxPrefix, ""); inbox != "" && !plainSubject(inbox) {
		return fmt.Sprintf("the inbox prefix %q is not a subject without wildcards", inbox)
	}
	return ""
}

// plainSubject reports whether s is a subject of non-empty tokens without spaces or wildcards.
func plainSubject(s string) bool {
	if strings.ContainsFunc(s, unicode.IsSpace) || strings.ContainsAny(s, "*>") {
		return false
	}
	return !slices.Contains(strings.Split(s, "."), "")
}

func urls(raw string) []string {
	var out []string
	for u := range strings.SplitSeq(raw, ",") {
		if u = strings.TrimSpace(u); u != "" {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		return []string{nats.DefaultURL}
	}
	return out
}

// translator reads the files a context points at and collects what it could not carry over.
type translator struct {
	read     readFunc
	warnings []string
}

func (t *translator) warn(msg string) {
	if !slices.Contains(t.warnings, msg) {
		t.warnings = append(t.warnings, msg)
	}
}

// file returns the content of a referenced file, or nil with a warning.
func (t *translator) file(what, path string) []byte {
	if path == "" {
		return nil
	}
	if t.read == nil {
		t.warn(fmt.Sprintf("the %s file %s stays on the machine the context came from: add it to the connection", what, path))
		return nil
	}
	content, err := t.read(path)
	if err != nil {
		t.warn(fmt.Sprintf("the %s file %s could not be read: %v", what, path, err))
		return nil
	}
	return content
}

func (t *translator) seed(path string) []byte {
	content := t.file("NKey", path)
	if content == nil {
		return nil
	}
	return t.seedFrom(content, "the NKey file "+path)
}

// seedFrom returns the NKey seed in content, or nil with a warning naming where it came from.
func (t *translator) seedFrom(content []byte, source string) []byte {
	kp, err := nkeys.ParseDecoratedNKey(content)
	if err != nil {
		t.warn(fmt.Sprintf("%s holds no seed: %v", source, err))
		return nil
	}
	seed, err := kp.Seed()
	if err != nil {
		t.warn(fmt.Sprintf("%s holds no seed: %v", source, err))
		return nil
	}
	return seed
}

// userSeed is the seed that signs for a user JWT: the context's own user seed, else its NKey file.
func (t *translator) userSeed(cli cliContext) []byte {
	if cli.UserSeed != "" {
		return t.seedFrom([]byte(cli.UserSeed), "the user seed")
	}
	return t.seed(cli.NKey)
}

func (t *translator) auth(cli cliContext) *entities.AuthConfig {
	if creds := t.file("credentials", cli.Creds); creds != nil {
		return &entities.AuthConfig{Method: entities.AuthMethodCredentials, Credentials: ptr.Wrap(string(creds))}
	}
	if cli.UserJWT != "" {
		if seed := t.userSeed(cli); seed != nil {
			return &entities.AuthConfig{Method: entities.AuthMethodCredentials, JWT: ptr.Wrap(cli.UserJWT), NkeySeed: ptr.Wrap(string(seed))}
		}
		t.warn("the user JWT comes without a seed: add the seed or a credentials file to the connection")
	} else if seed := t.seed(cli.NKey); seed != nil {
		return &entities.AuthConfig{Method: entities.AuthMethodNKey, NkeySeed: ptr.Wrap(string(seed))}
	}
	if cli.Token != "" {
		return &entities.AuthConfig{Method: entities.AuthMethodToken, Token: ptr.Wrap(cli.Token)}
	}
	if cli.User != "" {
		return &entities.AuthConfig{Method: entities.AuthMethodUserPass, Username: ptr.Wrap(cli.User), Password: ptr.WrapNonZero(cli.Password)}
	}
	return nil
}

func (t *translator) tls(cli cliContext) *entities.TlsConfig {
	cfg := &entities.TlsConfig{
		CaCert:     wrapBytes(t.file("CA certificate", cli.CA)),
		ClientCert: wrapBytes(t.file("client certificate", cli.Cert)),
		ClientKey:  wrapBytes(t.file("client key", cli.Key)),
		TlsFirst:   cli.TLSFirst,
	}
	if cfg.IsEmpty() {
		return nil
	}
	return cfg
}

// connection carries the inbox prefix and JetStream target; with both a domain and an API prefix, the domain wins.
func (t *translator) connection(cli cliContext) *entities.ConnectionConfig {
	if cli.InboxPrefix == "" && cli.JetStreamDomain == "" && cli.JetStreamAPI == "" {
		return nil
	}
	prefix := cli.JetStreamAPI
	if cli.JetStreamDomain != "" && prefix != "" {
		t.warn("the JetStream API prefix is ignored: the context also sets a JetStream domain")
		prefix = ""
	}
	return &entities.ConnectionConfig{
		InboxPrefix:        ptr.WrapNonZero(cli.InboxPrefix),
		JetstreamDomain:    ptr.WrapNonZero(cli.JetStreamDomain),
		JetstreamAPIPrefix: ptr.WrapNonZero(prefix),
	}
}

// description keeps as much of the context's description as a saved connection holds.
func (t *translator) description(text string) string {
	runes := []rune(text)
	if len(runes) <= maxDescription {
		return text
	}
	t.warn(fmt.Sprintf("the description was shortened to %d characters", maxDescription))
	return string(runes[:maxDescription])
}

func wrapBytes(b []byte) *string {
	if b == nil {
		return nil
	}
	return ptr.Wrap(string(b))
}

// readReferenced reads a file a local context points at, expanding a leading ~.
func readReferenced(path string) ([]byte, error) {
	if rest, ok := strings.CutPrefix(path, "~"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, coreerrs.WrapOperation(err, "find home directory")
		}
		path = filepath.Join(home, rest)
	}
	return readLimited(path)
}

// readLimited reads a regular file of at most maxReferencedFile bytes. It checks the file type before opening it,
// as opening a FIFO waits for a writer.
func readLimited(path string) ([]byte, error) {
	path = filepath.Clean(path)
	before, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	content, err := io.ReadAll(io.LimitReader(f, maxReferencedFile+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxReferencedFile {
		return nil, errors.New("larger than 1 MiB")
	}
	return content, nil
}
