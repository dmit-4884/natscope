// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package convcodecs holds custom go-atlas converter codecs. Each handles one
// type pair the converter lacks built-in support for, delegating to the next
// codec via CodecHandler when it doesn't match.
package convcodecs

import (
	"encoding/base64"
	"reflect"

	convcodec "github.com/altessa-s/go-atlas/domain/converter/codec"
)

// BytesBase64 encodes []byte to base64 string, else falls through. Don't
// pair with BytesPlain — both match []byte -> string; first registered wins.
var BytesBase64 convcodec.Codec = func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
	if src.Kind() == reflect.Slice && src.Type().Elem().Kind() == reflect.Uint8 && dst.Kind() == reflect.String {
		dst.SetString(base64.StdEncoding.EncodeToString(src.Bytes()))
		return
	}
	next(fieldName, src, dst)
}

// StringSliceFirst reduces []string to its first element (zero value if
// empty). Every key lands in dest map; callers must skip empty entries.
var StringSliceFirst convcodec.Codec = func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
	if src.Kind() == reflect.Slice && src.Type().Elem().Kind() == reflect.String && dst.Kind() == reflect.String {
		if src.Len() > 0 {
			dst.SetString(src.Index(0).String())
		}
		return
	}
	next(fieldName, src, dst)
}
