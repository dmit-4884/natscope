// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package registry is the per-snapshot proto descriptor cache keyed by
// (sourceID, tag), lazily populated via DescriptorsLookup and
// singleflight-deduped; invalidation is explicit (proto service drives it).
package registry
