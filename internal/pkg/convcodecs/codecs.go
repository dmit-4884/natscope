// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package convcodecs holds custom go-atlas converter codecs. Each handles one
// type pair the converter lacks built-in support for, delegating to the next
// codec via CodecHandler when it doesn't match.
package convcodecs

import (
	"encoding/base64"
	"reflect"
	"strconv"
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

// StringSliceJoin joins a []string into one comma-separated string.
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

// DurationSaturating converts google.protobuf.Duration to time.Duration with AsDuration, which saturates on
// overflow and keeps an explicit zero as a non-nil pointer. Register it before durpb.New().
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

// DurationString renders a time.Duration as its String form, leaving zero as "".
var DurationString convcodec.Codec = func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
	if src.Type() != goDurationType || dst.Kind() != reflect.String {
		next(fieldName, src, dst)
		return
	}
	if d := time.Duration(src.Int()); d != 0 {
		dst.SetString(d.String())
	}
}

// EnumNames renders integer enums listed in names as the name at their value's index; unknown values render as the number.
func EnumNames(names map[reflect.Type][]string) convcodec.Codec {
	return func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		list, ok := names[src.Type()]
		if !ok || dst.Kind() != reflect.String {
			next(fieldName, src, dst)
			return
		}
		if i := src.Int(); i >= 0 && i < int64(len(list)) {
			dst.SetString(list[i])
			return
		}
		dst.SetString(strconv.FormatInt(src.Int(), 10))
	}
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
