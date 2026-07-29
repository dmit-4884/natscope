// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"github.com/go-ozzo/ozzo-validation/v4"
)

// StorageConfig is the storage backend configuration.
type StorageConfig struct {
	// Local data directory holding the bbolt file; DataDir defaults to
	// ~/.natscope/data.
	Local *LocalStorageConfig `yaml:"local" default:"-"`
}

// LocalStorageConfig configures the local data directory.
type LocalStorageConfig struct {
	// DataDir is the data directory; defaults to ~/.natscope/data if empty.
	DataDir string `yaml:"dataDir"`
}

// Validate performs validation of the StorageConfig configuration.
func (c *StorageConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Local, validation.NilOrNotEmpty),
	)
}

// GetDataDir returns the local storage data directory, or empty string for default.
func (c *StorageConfig) GetDataDir() string {
	if c.Local != nil {
		return c.Local.DataDir
	}
	return ""
}

// Validate performs validation of the LocalStorageConfig configuration.
func (c *LocalStorageConfig) Validate() error {
	return validation.ValidateStruct(c)
}
