package gotime

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"encoding/json/v2"
)

func unmarshalJSONWire(b []byte, out any) error {
	if err := json.Unmarshal(b, out, json.RejectUnknownMembers(true)); err != nil {
		return newTimeErrorWithCause(
			ErrInvalidFormat,
			err,
			"JSON does not match the required wire structure",
			"",
			"provide the documented JSON string or closed object with matching value types",
		)
	}
	return nil
}

func requireJSONString(typeName, field, value string) error {
	if value != "" {
		return nil
	}
	if field == "" {
		return newTimeError(ErrInvalidFormat, typeName+" requires a non-empty string", "",
			"provide the documented value as a non-empty JSON string")
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

// jsonInstantPattern owns the wire shape; calendar validity stays with time.Parse.
var jsonInstantPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?(Z|[+-][0-9]{2}:[0-9]{2})$`)

func parseJSONInstant(value string) (Instant, error) {
	// Preserve stdlib parse causes before checking the narrower wire profile.
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return Instant{}, newTimeErrorWithCause(ErrInvalidFormat, err,
			"instant is not valid RFC3339", value,
			"use an RFC3339 instant such as 2026-03-27T04:00:00Z")
	}
	match := jsonInstantPattern.FindStringSubmatch(value)
	if match == nil {
		return Instant{}, newTimeError(ErrInvalidFormat,
			"instant does not match the JSON wire grammar", value,
			"use YYYY-MM-DDTHH:MM:SS with optional decimal-point seconds and Z or a signed HH:MM offset")
	}
	if match[1] != "Z" {
		if _, err := parseOffsetLocation(match[1]); err != nil {
			return Instant{}, newTimeErrorWithCause(ErrInvalidFormat, err,
				"instant offset is outside the JSON wire range", value,
				"use offset hours 00-23 and minutes 00-59")
		}
	}
	if err := requireJSONPrecision(value); err != nil {
		return Instant{}, err
	}
	return InstantFromTime(t), nil
}

var jsonClockPattern = regexp.MustCompile(`^[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,9})?$`)

func requireJSONClock(value string) error {
	if jsonClockPattern.MatchString(value) {
		return nil
	}
	return newTimeError(ErrInvalidFormat, "clock does not match the JSON wire grammar", value,
		"use HH:MM:SS with at most nine decimal-point fraction digits and no offset")
}
