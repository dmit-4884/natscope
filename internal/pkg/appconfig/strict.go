// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/altessa-s/go-atlas/core/runtime/appinfo"
)

// validConfigExtensions mirrors the yaml3 loader backend: only .yaml/.yml is
// ever decoded. A single file with any other extension is silently skipped
// by the loader (zero files loaded, so every field keeps its default) —
// validate it up front instead of starting on an unintentionally empty
// configuration. Directory mode already documents this skip behavior, so it
// is left alone here.
func validateConfigPath(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		// Missing path or directory mode: let the real loader report a
		// missing-path error, or fall through to the documented per-file skip.
		return nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("unsupported config file extension %q (expected .yaml or .yml): %s", filepath.Ext(filePath), filePath)
	}
	return nil
}

// checkKnownFields re-decodes the same YAML file(s) with strict field
// checking, purely to catch a typo'd or misnamed key (e.g. "datadir" instead
// of "dataDir", "webauth" instead of "webAuth") that the real loader — whose
// yaml3 backend does not enable yaml.Decoder.KnownFields — otherwise accepts
// and silently drops.
func checkKnownFields(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil // the real loader already reported (or will report) this.
	}
	if !info.IsDir() {
		return checkKnownFieldsFile(filePath)
	}
	entries, err := os.ReadDir(filePath)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if err := checkKnownFieldsFile(filepath.Join(filePath, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// checkKnownFieldsFile decodes one file into a throwaway Config with strict
// field checking. Env-var substitution is irrelevant here — it rewrites
// values, never keys — so the raw file content is fine for this check.
func checkKnownFieldsFile(path string) error {
	f, err := os.Open(path) //nolint:gosec // path comes from the operator-supplied --config flag/env, not untrusted input
	if err != nil {
		return nil // unreadable; the real loader's error already covers this.
	}
	defer f.Close() //nolint:errcheck

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var probe Config
	if err := dec.Decode(&probe); err != nil && err != io.EOF { //nolint:errorlint // yaml.v3 returns the sentinel directly, not wrapped
		return fmt.Errorf("config file %s: %w", path, err)
	}
	return nil
}

// emptyOverridableEnvKeys lists the documented flat/nested env vars whose
// empty value should be treated as "unset" rather than as an explicit
// override to the field's zero value — a docker-compose/systemd template
// like "LOGGER__LEVEL: ${LOG_LEVEL}" with an unset LOG_LEVEL must not
// silently blank a value already set in the config file.
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
}

// clearEmptyEnvOverrides unsets any of emptyOverridableEnvKeys (with the
// build's env prefix applied) that are present but set to the empty string.
func clearEmptyEnvOverrides() {
	for _, key := range emptyOverridableEnvKeys {
		name := EnvPrefix() + key
		if v, ok := os.LookupEnv(name); ok && v == "" {
			_ = os.Unsetenv(name)
		}
	}
}

// boolEnvKeys are the boolean config keys settable through the environment.
var boolEnvKeys = []string{"ALLOW_REMOTE", "ALLOW_INSECURE", "LOGGER__COLORIZED"}

// normalizeBoolEnv rewrites the YAML 1.1 boolean words the config file accepts
// (yes/no, on/off, y/n) to true/false in boolean env vars, so both sources
// follow the same rules.
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

// EnvPrefix returns the build's env var prefix joined with "_" (or "" for an
// unprefixed build), so config keys use the same PREFIX_KEY form as LIB_DIR,
// VAR_DIR and CONFIG_FILE.
func EnvPrefix() string {
	if appinfo.EnvPrefix == "" {
		return ""
	}
	return strings.TrimRight(appinfo.EnvPrefix, "_") + "_"
}
