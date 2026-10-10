// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// SchemaUpload is a schema sent from the browser: .proto files with buf configs, or a compiled descriptor set.
type SchemaUpload struct {
	Files         []ProtoFileEntry
	DescriptorSet []byte
}
