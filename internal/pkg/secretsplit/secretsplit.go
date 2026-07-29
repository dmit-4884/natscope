// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package secretsplit maps a persistence document's secret fields to and from an
// out-of-database vault. A secret field is tagged `behavior:"input_only"` and
// carries a `secret:"<vault key>"` tag naming its vault key. Split lifts those
// fields' values into a map keyed by that tag and blanks them on the document;
// Merge refills them from such a map. This keeps secrets out of the plaintext
// store while the document round-trips through it.
package secretsplit

import (
	"fmt"
	"reflect"

	"github.com/altessa-s/go-atlas/domain/behavior"
)

// secretTag is the struct tag holding a field's vault key.
const secretTag = "secret"

// Split returns doc's input_only-tagged secrets keyed by their `secret` tag and
// blanks those fields on doc, so the document can be persisted without them. A
// nil or empty secret is omitted from the map. doc must be a non-nil pointer.
func Split[D any](doc *D) (map[string]string, error) {
	root, err := behavior.Resolve(doc, behavior.WithKinds(behavior.InputOnly))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := collect(root, out); err != nil {
		return nil, err
	}
	// Blank the captured fields via the framework's canonical strip so the
	// document never carries a secret into the store.
	if err := behavior.Strip(doc, behavior.WithKinds(behavior.InputOnly)); err != nil {
		return nil, err
	}
	return out, nil
}

// Merge refills doc's input_only-tagged fields from secs (keyed by `secret` tag).
// A missing or empty entry leaves the field nil. doc must be a non-nil pointer.
func Merge[D any](doc *D, secs map[string]string) error {
	root, err := behavior.Resolve(doc, behavior.WithKinds(behavior.InputOnly))
	if err != nil {
		return err
	}
	return inject(root, secs)
}

// collect records each stripped (secret) leaf's value under its vault key,
// recursing into nested structs.
func collect(obj behavior.Object, out map[string]string) error {
	for i := range obj.Fields {
		f := &obj.Fields[i]
		if f.Strip {
			key := f.Tag.Get(secretTag)
			if key == "" {
				return fmt.Errorf("secretsplit: field %q is input_only but has no `secret` tag", f.Name)
			}
			s, ok, err := stringValue(f.Value)
			if err != nil {
				return fmt.Errorf("secretsplit: field %q (%s): %w", f.Name, key, err)
			}
			if ok && s != "" {
				out[key] = s
			}
			continue
		}
		if f.Nested != nil {
			if err := collect(*f.Nested, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// inject sets each stripped (secret) leaf from secs, recursing into nested structs.
func inject(obj behavior.Object, secs map[string]string) error {
	for i := range obj.Fields {
		f := &obj.Fields[i]
		if f.Strip {
			key := f.Tag.Get(secretTag)
			if key == "" {
				return fmt.Errorf("secretsplit: field %q is input_only but has no `secret` tag", f.Name)
			}
			if err := setString(f.Value, secs[key]); err != nil {
				return fmt.Errorf("secretsplit: field %q (%s): %w", f.Name, key, err)
			}
			continue
		}
		if f.Nested != nil {
			if err := inject(*f.Nested, secs); err != nil {
				return err
			}
		}
	}
	return nil
}

// stringValue reads a *string value; ok is false for a nil pointer.
func stringValue(v reflect.Value) (string, bool, error) {
	if v.Kind() != reflect.Pointer || v.Type().Elem().Kind() != reflect.String {
		return "", false, fmt.Errorf("unsupported secret target kind %s (want *string)", v.Kind())
	}
	if v.IsNil() {
		return "", false, nil
	}
	return v.Elem().String(), true, nil
}

// setString sets a *string field to s, or to nil when s is empty.
func setString(v reflect.Value, s string) error {
	if v.Kind() != reflect.Pointer || v.Type().Elem().Kind() != reflect.String {
		return fmt.Errorf("unsupported secret target kind %s (want *string)", v.Kind())
	}
	if !v.CanSet() {
		return fmt.Errorf("secret field is not settable (pass a pointer to the document)")
	}
	if s == "" {
		v.SetZero()
		return nil
	}
	p := reflect.New(v.Type().Elem())
	p.Elem().SetString(s)
	v.Set(p)
	return nil
}
