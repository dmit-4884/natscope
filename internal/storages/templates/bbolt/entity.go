// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// templateDoc is the persistence model: the base plus the template fields, with
// headers and wildcards kept as nested JSON instead of child tables.
type templateDoc struct {
	bbstore.Base

	Name        string            `json:"name"`
	Subject     string            `json:"subject,omitempty"`
	MessageType string            `json:"messageType,omitempty"`
	Data        string            `json:"data,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Wildcards   []string          `json:"wildcards,omitempty"`
}
