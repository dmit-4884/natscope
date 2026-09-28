// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package convcodecs holds custom go-atlas converter codecs. Each handles one
// type pair the converter lacks built-in support for, delegating to the next
// codec via CodecHandler when it doesn't match.
package convcodecs

import (
	"encoding/base64"
	"reflect"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

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

// StringSliceJoin reduces []string to a single comma-separated string,
// preserving every value; used where a header value can repeat.
var StringSliceJoin convcodec.Codec = func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
	if src.Kind() == reflect.Slice && src.Type().Elem().Kind() == reflect.String && dst.Kind() == reflect.String {
		if src.Len() > 0 {
			values := make([]string, src.Len())
			for i := range values {
				values[i] = src.Index(i).String()
			}
			dst.SetString(strings.Join(values, ", "))
		}
		return
	}
	next(fieldName, src, dst)
}

var (
	durationPBType = reflect.TypeFor[durationpb.Duration]()
	goDurationType = reflect.TypeFor[time.Duration]()
)

// DurationSaturating intercepts google.protobuf.Duration -> time.Duration
// conversions and computes the result via the standard library's
// AsDuration(), which saturates to [math.MinInt64, math.MaxInt64] nanoseconds
// on overflow instead of silently wrapping modulo 2^64 (go-atlas's durpb
// codec multiplies seconds*time.Second with no overflow check). Must be
// registered before durpb.New() in the codec chain so it gets first refusal;
// every other type pair falls through unchanged. It also happens to preserve
// an explicit-zero Duration (present but {0,0}) as a non-nil *time.Duration
// rather than treating it the same as "absent" the way durpb's
// WithIgnoreZero does — which is exactly the presence semantics a mutable
// Update* request field needs to be able to reset a value to zero.
//
// The reverse direction (time.Duration -> Duration) can't overflow, since a
// valid time.Duration already fits the int64 nanosecond range, so it is left
// to durpb — which keeps response formatting (zero duration -> absent field)
// unchanged.
var DurationSaturating convcodec.Codec = func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
	srcValue := reflect.Indirect(src)
	if !srcValue.IsValid() || indirectType(src.Type()) != durationPBType || indirectType(dst.Type()) != goDurationType {
		next(fieldName, src, dst)
		return
	}

	secField := srcValue.FieldByName("Seconds")
	nanField := srcValue.FieldByName("Nanos")
	if !secField.IsValid() || !nanField.IsValid() {
		next(fieldName, src, dst)
		return
	}

	goDur := (&durationpb.Duration{Seconds: secField.Int(), Nanos: int32(nanField.Int())}).AsDuration()

	if dst.Kind() == reflect.Pointer {
		dst.Set(reflect.New(goDurationType))
		dst = dst.Elem()
	}
	dst.SetInt(int64(goDur))
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
