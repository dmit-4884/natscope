// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	connbbolt "github.com/dmit-4884/natscope/internal/storages/connections/bbolt"
)

func newStorage(t *testing.T) (*connbbolt.Storage, *secrets.Memory, *bbstore.DB) {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	vault := secrets.NewMemory()
	s, err := connbbolt.New(t.Context(), db, vault)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	return s, vault, db
}

func sampleConn(name string) *entities.SavedConnection {
	return entities.SavedConnectionNew(func(c *entities.SavedConnection) {
		c.Name = name
		c.URLs = []string{"nats://localhost:4222"}
		c.Auth = &entities.AuthConfig{
			Method:   entities.AuthMethodUserPass,
			Username: ptr.Wrap("admin"),
			Password: ptr.Wrap("super-secret-pw"),
		}
	})
}

// TestConnections_AllGroupsRoundTrip exercises every sub-struct field so a
// dropped value (in the doc or the vault) surfaces here.
func TestConnections_AllGroupsRoundTrip(t *testing.T) {
	s, _, _ := newStorage(t)
	ctx := t.Context()

	in := entities.SavedConnectionNew(func(c *entities.SavedConnection) {
		c.Name = "full"
		c.Description = ptr.Wrap("desc")
		c.URLs = []string{"nats://a:4222", "nats://b:4222"}
		c.Auth = &entities.AuthConfig{
			Method:   entities.AuthMethodCredentials,
			Username: ptr.Wrap("u"), Password: ptr.Wrap("p"), Token: ptr.Wrap("tok"),
			NkeySeed: ptr.Wrap("seed"), Credentials: ptr.Wrap("creds"), JWT: ptr.Wrap("jwt"),
		}
		c.TLS = &entities.TlsConfig{
			CaCert: ptr.Wrap("ca"), ClientCert: ptr.Wrap("cc"), ClientKey: ptr.Wrap("ck"),
			SkipVerify: true, TlsFirst: true,
		}
		ct := 7 * time.Second
		c.Connection = &entities.ConnectionConfig{
			ConnectTimeout: &ct, ConnectionName: ptr.Wrap("cn"), InboxPrefix: ptr.Wrap("_INBOX"),
			NoEcho: true, NoRandomize: true, IgnoreDiscoveredServers: true,
		}
		rw := 3 * time.Second
		c.Reconnect = &entities.ReconnectConfig{
			MaxReconnects: ptr.Wrap(int32(10)), ReconnectWait: &rw,
			ReconnectBufSize: ptr.Wrap(int32(4096)), RetryOnFailedConnect: true,
		}
		pi := 2 * time.Minute
		c.Ping = &entities.PingConfig{PingInterval: &pi, MaxPingsOutstanding: ptr.Wrap(int32(3))}
		c.Meta = &entities.ConnectionMeta{
			LastTestedAt: time.UnixMilli(1700000000123).UTC(), LastSuccess: true,
			LastRTTMs: ptr.Wrap(int64(42)), ServerVersion: ptr.Wrap("2.12"), ServerName: ptr.Wrap("n1"),
			ServerID: ptr.Wrap("ID1"), ClusterName: ptr.Wrap("cl"), MaxPayload: ptr.Wrap(int64(1048576)),
			JetstreamEnabled: ptr.Wrap(true), ConnectedURL: ptr.Wrap("nats://a:4222"), LastError: ptr.Wrap("none"),
		}
	})
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if len(got.URLs) != 2 || got.URLs[1] != "nats://b:4222" {
		t.Fatalf("urls: %v", got.URLs)
	}
	if got.Auth.Method != entities.AuthMethodCredentials || *got.Auth.Username != "u" ||
		*got.Auth.Password != "p" || *got.Auth.Token != "tok" || *got.Auth.NkeySeed != "seed" ||
		*got.Auth.Credentials != "creds" || *got.Auth.JWT != "jwt" {
		t.Fatalf("auth mismatch: %+v", got.Auth)
	}
	if *got.TLS.CaCert != "ca" || *got.TLS.ClientCert != "cc" || *got.TLS.ClientKey != "ck" ||
		!got.TLS.SkipVerify || !got.TLS.TlsFirst {
		t.Fatalf("tls mismatch: %+v", got.TLS)
	}
	if *got.Connection.ConnectTimeout != 7*time.Second || *got.Connection.ConnectionName != "cn" ||
		*got.Connection.InboxPrefix != "_INBOX" || !got.Connection.NoEcho || !got.Connection.NoRandomize ||
		!got.Connection.IgnoreDiscoveredServers {
		t.Fatalf("connection mismatch: %+v", got.Connection)
	}
	if *got.Reconnect.MaxReconnects != 10 || *got.Reconnect.ReconnectWait != 3*time.Second ||
		*got.Reconnect.ReconnectBufSize != 4096 || !got.Reconnect.RetryOnFailedConnect {
		t.Fatalf("reconnect mismatch: %+v", got.Reconnect)
	}
	if *got.Ping.PingInterval != 2*time.Minute || *got.Ping.MaxPingsOutstanding != 3 {
		t.Fatalf("ping mismatch: %+v", got.Ping)
	}
	if got.Meta.LastTestedAt.UnixMilli() != 1700000000123 || !got.Meta.LastSuccess ||
		*got.Meta.LastRTTMs != 42 || *got.Meta.ServerVersion != "2.12" || *got.Meta.ServerName != "n1" ||
		*got.Meta.ServerID != "ID1" || *got.Meta.ClusterName != "cl" || *got.Meta.MaxPayload != 1048576 ||
		!*got.Meta.JetstreamEnabled || *got.Meta.ConnectedURL != "nats://a:4222" || *got.Meta.LastError != "none" {
		t.Fatalf("meta mismatch: %+v", got.Meta)
	}
}

// TestConnections_SecretsInVaultNotDB proves the credential leaves the
// database: readable via storage, present in the vault, absent from the raw doc.
func TestConnections_SecretsInVaultNotDB(t *testing.T) {
	s, vault, db := newStorage(t)
	ctx := t.Context()
	in := sampleConn("withsecret")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Auth == nil || got.Auth.Password == nil || *got.Auth.Password != "super-secret-pw" {
		t.Fatalf("password not rehydrated: %+v", got.Auth)
	}

	secs, err := vault.Get(ctx, "connections", in.Id)
	if err != nil {
		t.Fatalf("vault get: %v", err)
	}
	if secs["auth.password"] != "super-secret-pw" {
		t.Fatalf("vault missing password: %v", secs)
	}

	raw, err := db.GetRaw(ctx, "connections", in.Id)
	if err != nil {
		t.Fatalf("read raw doc: %v", err)
	}
	if strings.Contains(string(raw), "super-secret-pw") {
		t.Fatalf("secret leaked into DB doc: %s", raw)
	}
}

func TestConnections_NameUniqueness(t *testing.T) {
	s, _, _ := newStorage(t)
	ctx := t.Context()
	if err := s.Save(ctx, sampleConn("dup")); err != nil {
		t.Fatalf("save 1: %v", err)
	}
	if err := s.Save(ctx, sampleConn("dup")); !errors.Is(err, errs.ErrConnectionNameAlreadyInUse) {
		t.Fatalf("expected ErrConnectionNameAlreadyInUse, got %v", err)
	}
}

func TestConnections_Update(t *testing.T) {
	s, _, _ := newStorage(t)
	ctx := t.Context()
	in := sampleConn("c1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
		existing.Name = "c1-renamed"
		existing.Auth.Password = ptr.Wrap("new-pw")
		return false, false
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "c1-renamed" || *got.Auth.Password != "new-pw" {
		t.Fatalf("update not applied: %+v", got)
	}
}

// TestConnections_UpdatePreservesOmittedSecret verifies that an update carrying
// no password (the API never echoes stored secrets back) keeps the existing
// vault value instead of clearing it.
func TestConnections_UpdatePreservesOmittedSecret(t *testing.T) {
	s, _, _ := newStorage(t)
	ctx := t.Context()
	in := sampleConn("c1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
		existing.Name = "c1-renamed"
		existing.Auth.Password = nil // omitted on edit → must be preserved
		return false, false
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "c1-renamed" {
		t.Fatalf("name not updated: %+v", got)
	}
	if got.Auth == nil || got.Auth.Password == nil || *got.Auth.Password != "super-secret-pw" {
		t.Fatalf("omitted password not preserved: %+v", got.Auth)
	}
}

// TestConnections_UpdateAuthReplacePrunesStaleSecret checks that switching auth method drops the old method's secret.
func TestConnections_UpdateAuthReplacePrunesStaleSecret(t *testing.T) {
	s, vault, _ := newStorage(t)
	ctx := t.Context()
	in := sampleConn("c1") // AUTH_METHOD_USER_PASSWORD, password="super-secret-pw"
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Switch to token auth, as ApplyUpdate does for a request that replaces Auth.
	if _, err := s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
		existing.Auth = &entities.AuthConfig{Method: entities.AuthMethodToken, Token: ptr.Wrap("new-token")}
		return true, false
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Auth == nil || got.Auth.Token == nil || *got.Auth.Token != "new-token" {
		t.Fatalf("token not set: %+v", got.Auth)
	}
	if got.Auth.Password != nil {
		t.Fatalf("stale password survived method switch: %+v", got.Auth)
	}

	secs, err := vault.Get(ctx, "connections", in.Id)
	if err != nil {
		t.Fatalf("vault get: %v", err)
	}
	if _, ok := secs["auth.password"]; ok {
		t.Fatalf("stale auth.password key survived in vault: %v", secs)
	}
	if secs["auth.token"] != "new-token" {
		t.Fatalf("vault missing new token: %v", secs)
	}
}

// TestConnections_UpdateAuthReplaceToNoneClearsVault checks that clearing auth removes every auth.* vault key.
func TestConnections_UpdateAuthReplaceToNoneClearsVault(t *testing.T) {
	s, vault, _ := newStorage(t)
	ctx := t.Context()
	in := sampleConn("c1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
		existing.Auth = nil // mirrors SavedConnection.ApplyUpdate collapsing an empty AuthConfig
		return true, false
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	secs, err := vault.Get(ctx, "connections", in.Id)
	if err != nil {
		t.Fatalf("vault get: %v", err)
	}
	if len(secs) != 0 {
		t.Fatalf("expected empty vault entry after clearing auth, got: %v", secs)
	}
}

// TestConnections_UpdateTLSReplacePrunesStaleClientKey checks that replacing tls drops the old client key.
func TestConnections_UpdateTLSReplacePrunesStaleClientKey(t *testing.T) {
	s, vault, _ := newStorage(t)
	ctx := t.Context()
	in := sampleConn("c1")
	in.TLS = &entities.TlsConfig{ClientCert: ptr.Wrap("cc"), ClientKey: ptr.Wrap("super-secret-key")}
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
		existing.TLS = &entities.TlsConfig{SkipVerify: true} // replace with an empty-of-secrets block
		return false, true
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	secs, err := vault.Get(ctx, "connections", in.Id)
	if err != nil {
		t.Fatalf("vault get: %v", err)
	}
	if _, ok := secs["tls.clientKey"]; ok {
		t.Fatalf("stale tls.clientKey survived: %v", secs)
	}
	// tlsReplaced prunes only tls.* keys.
	if secs["auth.password"] != "super-secret-pw" {
		t.Fatalf("unrelated auth secret was affected: %v", secs)
	}
}

func TestConnections_SoftDelete(t *testing.T) {
	s, _, _ := newStorage(t)
	ctx := t.Context()
	in := sampleConn("c1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SoftDelete(ctx, &entities.SoftDelete{Id: in.Id, NewUpdatedAt: in.CreatedAt}); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err := s.Get(ctx, in.Id); !errors.Is(err, errs.ErrSavedConnectionNotFound) {
		t.Fatalf("expected not found for soft-deleted, got %v", err)
	}
	if _, err := s.Get(ctx, in.Id, true); err != nil {
		t.Fatalf("get includeDeleted: %v", err)
	}
	if err := s.Save(ctx, sampleConn("c1")); err != nil {
		t.Fatalf("re-use soft-deleted name: %v", err)
	}
}

// TestConnections_UpdateConcurrentPartialUpdatesDoNotLoseWrites checks that concurrent single-field updates all persist.
func TestConnections_UpdateConcurrentPartialUpdatesDoNotLoseWrites(t *testing.T) {
	s, _, _ := newStorage(t)
	ctx := t.Context()
	in := sampleConn("c1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	const rounds = 20
	for r := 1; r <= rounds; r++ {
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			_, _ = s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
				existing.Description = ptr.Wrap("d" + strconv.Itoa(r))
				return false, false
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
				if existing.Connection == nil {
					existing.Connection = &entities.ConnectionConfig{}
				}
				existing.Connection.ConnectionName = ptr.Wrap("cn" + strconv.Itoa(r))
				return false, false
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = s.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
				if existing.Ping == nil {
					existing.Ping = &entities.PingConfig{}
				}
				existing.Ping.MaxPingsOutstanding = ptr.Wrap(int32(r))
				return false, false
			})
		}()
		wg.Wait()

		got, err := s.Get(ctx, in.Id)
		if err != nil {
			t.Fatalf("round %d get: %v", r, err)
		}
		wantDesc := "d" + strconv.Itoa(r)
		wantConnName := "cn" + strconv.Itoa(r)
		if got.Description == nil || *got.Description != wantDesc {
			t.Fatalf("round %d: description lost, want %q got %v", r, wantDesc, got.Description)
		}
		if got.Connection == nil || got.Connection.ConnectionName == nil || *got.Connection.ConnectionName != wantConnName {
			t.Fatalf("round %d: connection name lost, want %q got %+v", r, wantConnName, got.Connection)
		}
		if got.Ping == nil || got.Ping.MaxPingsOutstanding == nil || *got.Ping.MaxPingsOutstanding != int32(r) {
			t.Fatalf("round %d: max pings outstanding lost, want %d got %+v", r, r, got.Ping)
		}
	}
}

func TestConnections_ListPagination(t *testing.T) {
	s, _, _ := newStorage(t)
	ctx := t.Context()
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		if err := s.Save(ctx, sampleConn(n)); err != nil {
			t.Fatalf("save %s: %v", n, err)
		}
	}
	page, err := s.List(ctx, &entities.SavedConnectionsList{
		ListBase: entities.ListBase{Limit: ptr.Wrap(int64(2)), IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 || page.Total == nil || *page.Total != 5 || page.NextCursor == nil {
		t.Fatalf("page1 = %d items total=%v cursor=%v", len(page.Items), page.Total, page.NextCursor)
	}
}
