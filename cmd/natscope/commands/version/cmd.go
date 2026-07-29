// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"github.com/altessa-s/go-atlas/core/runtime/appinfo"
)

type Command struct {
	*cobra.Command
}

// New creates and returns the version command
func New() *cobra.Command {
	version := &Command{
		Command: &cobra.Command{
			Use:   "version",
			Short: "Display version information",
			Long: heredoc.Doc(`
				Display version information for natscope.

				Prints the version number by default. Use --full for build details
				(commit, build time, project). Use --output json for scripts.
			`),
			Example: heredoc.Doc(`
				# Display the current version
				natscope version

				# Display full version information
				natscope version --full

				# Display version information in JSON format
				natscope version --output json

				# Display full version information in JSON format
				natscope version --full --output json
			`),
			SilenceUsage: true,
		},
	}

	version.configure()
	return version.Command
}

func (c *Command) configure() {
	c.Flags().BoolP("full", "f", false, "show build time, commit, project, and env prefix")
	c.Flags().StringP("output", "o", "text", "output format: text (default) or json")

	c.PersistentPreRunE = c.preRun
	c.RunE = c.run
}

// preRun validates the --output flag.
func (c *Command) preRun(cmd *cobra.Command, _ []string) error {
	output, _ := cmd.Flags().GetString("output") //nolint:errcheck
	switch output {
	case "text", "json":
		return nil
	default:
		return fmt.Errorf("invalid output format: %s. Valid formats are: text, json", output)
	}
}

func (c *Command) run(cmd *cobra.Command, _ []string) error {
	output, _ := cmd.Flags().GetString("output") //nolint:errcheck

	if output == "json" {
		printJSON(cmd.Flags().Lookup("full").Value.String() == "true")
		return nil
	}
	printText(cmd.Flags().Lookup("full").Value.String() == "true")
	return nil
}

// printText prints version information in text format
func printText(full bool) {
	if full {
		fmt.Printf("Version:    %s\n", appinfo.Version)
		fmt.Printf("Build Time: %s\n", appinfo.BuildTime)
		fmt.Printf("Commit:     %s\n", appinfo.Commit)
		fmt.Printf("Project:    %s\n", appinfo.Project)
		fmt.Printf("Env Prefix: %s\n", appinfo.EnvPrefix)
	} else {
		fmt.Println(appinfo.Version)
	}
}

// printJSON prints version information in JSON format
func printJSON(full bool) {
	if full {
		versionInfo := struct {
			Version   string `json:"version"`
			BuildTime string `json:"build_time"`
			Commit    string `json:"commit"`
			Project   string `json:"project"`
			EnvPrefix string `json:"env_prefix"`
		}{
			Version:   appinfo.Version,
			BuildTime: appinfo.BuildTime,
			Commit:    appinfo.Commit,
			Project:   appinfo.Project,
			EnvPrefix: appinfo.EnvPrefix,
		}

		jsonData, err := json.Marshal(versionInfo)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(string(jsonData))
		return
	}

	versionInfo := struct {
		Version string `json:"version"`
	}{Version: appinfo.Version}

	jsonData, err := json.Marshal(versionInfo)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}
