// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"sync"

	"github.com/zalando/go-keyring"

	atlasmaps "github.com/altessa-s/go-atlas/core/collections/maps"
	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// Vault stores per-entity secret values outside the database. A namespace plus
// entity id addresses one set of named secrets (field name -> value).
type Vault interface {
	// Put replaces the stored secrets for (namespace, id). Empty values are
	// dropped; if nothing remains the entry is removed.
	Put(ctx context.Context, namespace, id string, secrets map[string]string) error
	// Get returns the stored secrets, or an empty map when none exist.
	Get(ctx context.Context, namespace, id string) (map[string]string, error)
	// Delete removes all secrets for (namespace, id); a missing entry is not an
	// error.
	Delete(ctx context.Context, namespace, id string) error
}

// Keyring is a Vault backed by the OS keychain. Each entity's secrets are stored
// as one JSON blob under a per-entity account.
type Keyring struct {
	service string
}

// NewKeyring returns a keychain-backed vault under the given service name (use a
// stable app identifier such as "natscope").
func NewKeyring(service string) *Keyring {
	return &Keyring{service: service}
}

// Put stores the non-empty secrets as one JSON blob, or clears the entry when
// none remain.
func (k *Keyring) Put(ctx context.Context, namespace, id string, secrets map[string]string) error {
	clean := nonEmpty(secrets)
	if len(clean) == 0 {
		return k.Delete(ctx, namespace, id)
	}
	blob, err := json.Marshal(clean)
	if err != nil {
		return coreerrs.WrapOperation(err, "marshal secrets")
	}
	if err := keyring.Set(k.service, account(namespace, id), string(blob)); err != nil {
		return coreerrs.WrapOperation(err, "store secrets in keychain")
	}
	return nil
}

// Get reads and decodes the entity's secret blob; a missing entry yields an
// empty map.
func (k *Keyring) Get(_ context.Context, namespace, id string) (map[string]string, error) {
	raw, err := keyring.Get(k.service, account(namespace, id))
	if errors.Is(err, keyring.ErrNotFound) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "read secrets from keychain")
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, coreerrs.WrapOperation(err, "unmarshal secrets")
	}
	return out, nil
}

// Available reports whether the OS keychain actually works here — headless
// Linux and containers often lack a Secret Service. It round-trips a probe
// entry; any failure means "use another backend".
func (k *Keyring) Available() bool {
	const probeAccount = "secret/_probe/_probe"
	if err := keyring.Set(k.service, probeAccount, "ok"); err != nil {
		return false
	}
	_ = keyring.Delete(k.service, probeAccount) //nolint:errcheck // best-effort cleanup of the probe entry
	return true
}

// Delete removes the entity's secret entry; a missing entry is ignored.
func (k *Keyring) Delete(_ context.Context, namespace, id string) error {
	if err := keyring.Delete(k.service, account(namespace, id)); err != nil &&
		!errors.Is(err, keyring.ErrNotFound) {
		return coreerrs.WrapOperation(err, "delete secrets from keychain")
	}
	return nil
}

// Memory is an in-memory Vault for tests.
type Memory struct {
	mu   sync.Mutex
	data map[string]map[string]string
}

// NewMemory returns an empty in-memory vault.
func NewMemory() *Memory {
	return &Memory{data: map[string]map[string]string{}}
}

// Put stores a copy of the non-empty secrets, or clears the entry.
func (m *Memory) Put(_ context.Context, namespace, id string, secrets map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clean := nonEmpty(secrets)
	if len(clean) == 0 {
		delete(m.data, account(namespace, id))
		return nil
	}
	m.data[account(namespace, id)] = clean
	return nil
}

// Get returns a copy of the entity's secrets, or an empty map.
func (m *Memory) Get(_ context.Context, namespace, id string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v := m.data[account(namespace, id)]; v != nil {
		return maps.Clone(v), nil
	}
	return map[string]string{}, nil
}

// Delete drops the entity's secrets.
func (m *Memory) Delete(_ context.Context, namespace, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, account(namespace, id))
	return nil
}

// account is the keychain account (or memory key) for one entity's secrets.
func account(namespace, id string) string {
	return "secret/" + namespace + "/" + id
}

// nonEmpty returns a fresh map with the blank-valued entries removed.
func nonEmpty(in map[string]string) map[string]string {
	return atlasmaps.FilterMap(in, func(_, v string) bool { return v != "" })
}

var (
	_ Vault = (*Keyring)(nil)
	_ Vault = (*Memory)(nil)
)
