// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

// Package devtools pins dev-only tool dependencies so `go install ./...`
// from this directory installs reproducible versions for every developer.
// None of these are imported by production code.
package devtools

import (
	_ "github.com/daixiang0/gci"
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
)
