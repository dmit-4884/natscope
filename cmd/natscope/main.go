// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/joho/godotenv"

	"github.com/dmit-4884/natscope/cmd/natscope/commands"
)

// dotenvOptInVar gates loading .env from the current working directory. Off
// by default: an unrelated .env in whatever directory natscope happens to be
// launched from (a project checkout, a home directory) would otherwise
// silently override GRPC_WEB_ADDRESS, ALLOW_REMOTE, STORAGE__LOCAL__DATA_DIR
// and friends with no indication a .env was even read.
const dotenvOptInVar = "NATSCOPE_DOTENV"

func init() {
	if os.Getenv(dotenvOptInVar) != "" {
		_ = godotenv.Load(".env") //nolint:errcheck
	}
}

func main() {
	if err := commands.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
