// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package appconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/altessa-s/go-atlas/core/runtime/appinfo"
)

// validateConfigPath rejects a single config file whose extension the yaml3 loader would skip.
func validateConfigPath(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		// Missing path or directory mode: the real loader handles both.
		return nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("unsupported config file extension %q (expected .yaml or .yml): %s", filepath.Ext(filePath), filePath)
	}
	return nil
}

// emptyOverridableEnvKeys lists the env vars whose empty value counts as unset.
var emptyOverridableEnvKeys = []string{
	"GRPC_WEB_ADDRESS",
	"ALLOW_REMOTE",
	"ALLOW_INSECURE",
	"ALLOWED_HOSTS",
	"STORAGE__LOCAL__DATA_DIR",
	"LOGGER__LEVEL",
	"LOGGER__OUTPUT_FORMAT",
	"LOGGER__COLORIZED",
	"WEB_AUTH__USERNAME",
	"WEB_AUTH__PASSWORD",
	"SECRETS__BACKEND",
	"SECRETS__FILE_KEY",
	"MCP__ENABLED",
	"MCP__ALLOW_WRITES",
}

// clearEmptyEnvOverrides unsets every prefixed emptyOverridableEnvKeys entry set to "".
func clearEmptyEnvOverrides() {
	for _, key := range emptyOverridableEnvKeys {
		name := EnvPrefix() + key
		if v, ok := os.LookupEnv(name); ok && v == "" {
			_ = os.Unsetenv(name)
		}
	}
}

// boolEnvKeys are the boolean config keys settable through the environment.
var boolEnvKeys = []string{"ALLOW_REMOTE", "ALLOW_INSECURE", "LOGGER__COLORIZED", "MCP__ENABLED", "MCP__ALLOW_WRITES"}

// normalizeBoolEnv rewrites yes/no, on/off and y/n in boolEnvKeys to true/false, as the config file accepts them.
func normalizeBoolEnv() {
	for _, key := range boolEnvKeys {
		name := EnvPrefix() + key
		v, ok := os.LookupEnv(name)
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "yes", "y", "on":
			_ = os.Setenv(name, "true")
		case "no", "n", "off":
			_ = os.Setenv(name, "false")
		}
	}
}

// EnvPrefix returns the build's env var prefix followed by "_", or "" for an unprefixed build.
func EnvPrefix() string {
	if appinfo.EnvPrefix == "" {
		return ""
	}
	return strings.TrimRight(appinfo.EnvPrefix, "_") + "_"
}
