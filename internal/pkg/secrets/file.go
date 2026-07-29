// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"cmp"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/altessa-s/go-atlas/core/runtime/appinfo"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

const (
	fileVaultName = "secrets.vault"
	fileKeyName   = "vault.key"

	// fileMagic versions the on-disk layout: magic || 12-byte nonce || ciphertext.
	fileMagic = "NGV1"

	filePerm = 0o600
	dirPerm  = 0o700

	keyLen = 32 // AES-256

	// fileKeyEnv supplies the hex AES key out-of-band, so it need not sit on
	// disk beside the ciphertext. Empty -> co-located key file.
	fileKeyEnv = "SECRETS__FILE_KEY"
)

// ErrVaultKeyMismatch is returned when the key cannot decrypt the vault file.
var ErrVaultKeyMismatch = errors.New("vault key does not match the vault file")

// ErrVaultCorrupt is returned when the vault file is truncated or not a natscope
// vault (missing magic).
var ErrVaultCorrupt = errors.New("vault file is corrupt or not a natscope vault")

// ErrVaultKeyInvalid is returned when the AES key material (env or key file) is
// not the expected hex-encoded 32 bytes.
var ErrVaultKeyInvalid = errors.New("vault key material is malformed")

// File is a Vault persisted as one AES-256-GCM-encrypted JSON file. It backs
// installs without a usable OS keychain (headless Linux, containers). The key
// file beside it is the trust anchor — both live under the data dir with
// user-only permissions.
type File struct {
	mu   sync.Mutex
	path string
	aead cipher.AEAD
	data map[string]map[string]string
}

// NewFile opens (or initializes) the encrypted vault under dir: a missing key
// file is generated, a missing vault file yields an empty vault.
func NewFile(dir string) (*File, error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, coreerrs.WrapOperation(err, "create vault directory")
	}
	key, err := loadOrCreateKey(filepath.Join(dir, fileKeyName))
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "init vault cipher")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "init vault cipher")
	}

	f := &File{path: filepath.Join(dir, fileVaultName), aead: aead}
	if err := f.load(); err != nil {
		return nil, err
	}
	return f, nil
}

// Put stores the non-empty secrets and persists, or clears the entry when
// none remain.
func (f *File) Put(ctx context.Context, namespace, id string, secrets map[string]string) error {
	clean := nonEmpty(secrets)
	if len(clean) == 0 {
		return f.Delete(ctx, namespace, id)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[account(namespace, id)] = clean
	return f.persist()
}

// Get returns a copy of the entity's secrets, or an empty map.
func (f *File) Get(_ context.Context, namespace, id string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v := f.data[account(namespace, id)]; v != nil {
		return maps.Clone(v), nil
	}
	return map[string]string{}, nil
}

// Delete drops the entity's secrets and persists; a missing entry is ignored.
func (f *File) Delete(_ context.Context, namespace, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.data[account(namespace, id)]; !ok {
		return nil
	}
	delete(f.data, account(namespace, id))
	return f.persist()
}

// load decrypts the vault file into memory; a missing file is an empty vault.
func (f *File) load() error {
	raw, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		f.data = map[string]map[string]string{}
		return nil
	}
	if err != nil {
		return coreerrs.WrapOperation(err, "read vault file")
	}
	nonceLen := f.aead.NonceSize()
	if len(raw) < len(fileMagic)+nonceLen || string(raw[:len(fileMagic)]) != fileMagic {
		return fmt.Errorf("vault file %s: %w", f.path, ErrVaultCorrupt)
	}
	nonce := raw[len(fileMagic) : len(fileMagic)+nonceLen]
	plain, err := f.aead.Open(nil, nonce, raw[len(fileMagic)+nonceLen:], nil)
	if err != nil {
		return fmt.Errorf(
			"decrypt vault file %s: %w — restore the original key or delete both "+
				"files to start over (stored secrets will need re-entering)",
			f.path, ErrVaultKeyMismatch)
	}
	data := map[string]map[string]string{}
	if err := json.Unmarshal(plain, &data); err != nil {
		return coreerrs.WrapOperation(err, "unmarshal vault")
	}
	f.data = data
	return nil
}

// persist encrypts and atomically replaces the vault file. Callers hold f.mu.
func (f *File) persist() error {
	plain, err := json.Marshal(f.data)
	if err != nil {
		return coreerrs.WrapOperation(err, "marshal vault")
	}
	nonce := make([]byte, f.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return coreerrs.WrapOperation(err, "generate vault nonce")
	}
	blob := append([]byte(fileMagic), nonce...)
	blob = f.aead.Seal(blob, nonce, plain, nil)

	tmp := f.path + ".tmp"
	if err := writeFileSync(tmp, blob, filePerm); err != nil {
		_ = os.Remove(tmp) //nolint:errcheck // best-effort cleanup
		return coreerrs.WrapOperation(err, "write vault file")
	}
	if err := os.Rename(tmp, f.path); err != nil {
		_ = os.Remove(tmp) //nolint:errcheck // best-effort cleanup
		return coreerrs.WrapOperation(err, "replace vault file")
	}
	syncDir(filepath.Dir(f.path))
	return nil
}

// writeFileSync writes and fsyncs before closing, so the rename that follows
// cannot publish a file whose contents are still only in the page cache.
func writeFileSync(path string, data []byte, perm os.FileMode) error {
	fh, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := fh.Write(data); err != nil {
		_ = fh.Close() //nolint:errcheck
		return err
	}
	if err := fh.Sync(); err != nil {
		_ = fh.Close() //nolint:errcheck
		return err
	}
	return fh.Close()
}

// syncDir fsyncs a directory so the rename entry is durable. Best-effort:
// unsupported on some platforms, and the file contents are already durable.
func syncDir(dir string) {
	fh, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = fh.Sync()  //nolint:errcheck // best-effort
	_ = fh.Close() //nolint:errcheck // read-only handle
}

// loadOrCreateKey returns the AES key: an out-of-band hex key from fileKeyEnv
// wins; otherwise a co-located key file is read or generated.
func loadOrCreateKey(path string) ([]byte, error) {
	// Honor the derived env prefix (as the config loader does) so
	// <PREFIX>SECRETS__FILE_KEY works; fall back to the bare name.
	if env := strings.TrimSpace(cmp.Or(os.Getenv(appinfo.EnvPrefix+fileKeyEnv), os.Getenv(fileKeyEnv))); env != "" {
		key, decErr := hex.DecodeString(env)
		if decErr != nil || len(key) != keyLen {
			return nil, fmt.Errorf("%s must be %d hex-encoded bytes: %w", fileKeyEnv, keyLen, ErrVaultKeyInvalid)
		}
		return key, nil
	}
	// Co-located key is obfuscation only: whoever reads the vault file can read
	// the key beside it. Set fileKeyEnv for real at-rest protection.
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		key, decErr := hex.DecodeString(strings.TrimSpace(string(raw)))
		if decErr != nil || len(key) != keyLen {
			return nil, fmt.Errorf(
				"vault key file %s is malformed — expected %d hex-encoded bytes: %w", path, keyLen, ErrVaultKeyInvalid)
		}
		return key, nil
	case os.IsNotExist(err):
		key := make([]byte, keyLen)
		if _, rErr := rand.Read(key); rErr != nil {
			return nil, coreerrs.WrapOperation(rErr, "generate vault key")
		}
		if wErr := os.WriteFile(path, []byte(hex.EncodeToString(key)+"\n"), filePerm); wErr != nil {
			return nil, coreerrs.WrapOperation(wErr, "write vault key file")
		}
		return key, nil
	default:
		return nil, coreerrs.WrapOperation(err, "read vault key file")
	}
}

var _ Vault = (*File)(nil)
