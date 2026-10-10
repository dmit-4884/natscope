// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package appconfig

// MCPConfig configures the Model Context Protocol endpoint served at /mcp on the Connect listener.
type MCPConfig struct {
	Enabled     *bool `yaml:"enabled"`
	AllowWrites bool  `yaml:"allowWrites"`
}

// MCPEnabled reports whether the MCP endpoint is served; on by default.
func (c *Config) MCPEnabled() bool {
	return c.MCP == nil || c.MCP.Enabled == nil || *c.MCP.Enabled
}

// MCPAllowWrites reports whether MCP tools that publish to NATS are registered; off by default.
func (c *Config) MCPAllowWrites() bool {
	return c.MCP != nil && c.MCP.AllowWrites
}
