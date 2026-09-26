package gotime

import (
	"fmt"
	"strings"

	"encoding/json/v2"
)

func unmarshalJSONWire(b []byte, out any) error {
	if err := json.Unmarshal(b, out, json.RejectUnknownMembers(true)); err != nil {
		return newTimeErrorWithCause(
			ErrInvalidFormat,
			err,
			"JSON does not match the required wire structure",
			"",
			"provide a JSON object with only the documented fields and matching JSON value types",
		)
	}
	return nil
}

func requireJSONKind(typeName, got, want string) error {
	if got == want {
		return nil
	}
	if got == "" {
		return newTimeError(
			ErrInvalidFormat,
			fmt.Sprintf("%s kind is required", typeName),
			"",
			fmt.Sprintf("encode %s JSON with kind %q", typeName, want),
		)
	}
	return newTimeError(
		ErrInvalidFormat,
		fmt.Sprintf("%s kind must be %q", typeName, want),
		got,
		fmt.Sprintf("encode %s JSON with kind %q", typeName, want),
	)
}

func requireJSONString(typeName, field, value string) error {
	if value != "" {
		return nil
	}
	return newTimeError(
		ErrInvalidFormat,
		fmt.Sprintf("%s %s is required", typeName, field),
		"",
		fmt.Sprintf("include a non-empty %s field in %s JSON", field, typeName),
	)
}

func requireJSONPrecision(value string) error {
	index := strings.IndexAny(value, ".,")
	if index < 0 {
		return nil
	}
	digits := 0
	for _, c := range value[index+1:] {
		if c < '0' || c > '9' {
			break
		}
		digits++
	}
	if digits <= 9 {
		return nil
	}
	return newTimeError(ErrInvalidFormat, "fractional seconds exceed nanosecond precision", value,
		"use at most 9 fractional second digits; wire decoding never truncates precision")
}
