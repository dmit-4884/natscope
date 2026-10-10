// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package natscontext reads nats CLI contexts and translates them into connections to create: servers,
// credentials, TLS material, inbox prefix and JetStream domain or API prefix, with a warning for every
// setting Natscope cannot carry over.
package natscontext
