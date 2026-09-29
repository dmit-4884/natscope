// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/joho/godotenv"

	"github.com/dmit-4884/natscope/cmd/natscope/commands"
)

// dotenvOptInVar enables loading .env from the working directory; off by default.
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
