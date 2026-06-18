package gotime

import (
	"fmt"

	"github.com/go-json-experiment/json"
)

const iso8601Calendar = "iso8601"

func unmarshalJSONWire(b []byte, out any) error {
	return json.Unmarshal(b, out, json.RejectUnknownMembers(true))
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

func requireJSONCalendar(typeName, calendar string) error {
	if calendar == iso8601Calendar {
		return nil
	}
	if calendar == "" {
		return newTimeError(
			ErrInvalidFormat,
			fmt.Sprintf("%s calendar is required", typeName),
			"",
			fmt.Sprintf("encode %s JSON with calendar %q", typeName, iso8601Calendar),
		)
	}
	return newTimeError(
		ErrInvalidFormat,
		fmt.Sprintf("%s calendar must be %q", typeName, iso8601Calendar),
		calendar,
		fmt.Sprintf("encode %s JSON with calendar %q", typeName, iso8601Calendar),
	)
}
