// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package filewatcher tracks one local directory per source (keyed by opaque
// sourceID) and reports .proto changes via a single debounced ChangeCallback.
// Lifecycle: SetCallback before Start; Watch/Unwatch between Start and Stop.
// Implementations must be concurrency-safe and debounce rapid event bursts so
// the callback fires once per quiescent state.
package filewatcher
