// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// CliContextFile is an uploaded nats CLI context file.
type CliContextFile struct {
	Name    string
	Content []byte
}

// CliContext is a nats CLI context translated into a connection to create.
type CliContext struct {
	// Name is the context name, the file name without .json.
	Name string

	// Selected marks the context the CLI currently uses.
	Selected bool

	// Exists reports a saved connection with the same name.
	Exists bool

	// Connection is the connection the context becomes, credentials included; nil when the file is not a context.
	Connection *SavedConnectionCreate

	// Warnings name the settings that could not be carried over.
	Warnings []string
}

// CliContexts is the outcome of reading nats CLI contexts.
type CliContexts struct {
	// Dir is the directory the contexts were read from; empty for uploads.
	Dir string

	Contexts []CliContext
}

// CliContextSkip names a context that was not imported and why.
type CliContextSkip struct {
	Name   string
	Reason string
}

// CliContextImport is the outcome of importing nats CLI contexts.
type CliContextImport struct {
	Created SavedConnections
	Skipped []CliContextSkip
}
