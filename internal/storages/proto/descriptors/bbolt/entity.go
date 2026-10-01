// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

type descriptorDoc struct {
	bbstore.Base

	SourceID      string   `json:"sourceId"`
	Revision      string   `json:"revision"`
	DescriptorSet []byte   `json:"descriptorSet,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
	MessageTypes  []string `json:"messageTypes,omitempty"`
	TargetFiles   []string `json:"targetFiles,omitempty"`
	CompiledAt    int64    `json:"compiledAt,omitempty"`

	CompileSettings string `json:"compileSettings,omitempty"`
}
