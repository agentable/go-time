package gotime

import (
	"fmt"
	"strings"
	"time"

	"encoding/json/v2"
)

// Time represents a clock time without a date or timezone.
type Time struct {
	hour       int
	minute     int
	second     int
	nanosecond int
}

// NewTime creates a Time from hour, minute, and second.
func NewTime(hour, minute, second int) (Time, error) {
	return NewTimeNanos(hour, minute, second, 0)
}

// NewTimeNanos creates a Time with sub-second precision.
func NewTimeNanos(hour, minute, second, nanosecond int) (Time, error) {
	if msg := validateTimeComponents(hour, minute, second, nanosecond); msg != "" {
		return Time{}, newTimeError(
			ErrInvalidTime,
			msg,
			fmt.Sprintf("hour=%d minute=%d second=%d nanosecond=%d", hour, minute, second, nanosecond),
			"provide a clock time with hour 0-23, minute/second 0-59, and nanosecond 0-999999999",
		)
	}
	return timeFromComponents(hour, minute, second, nanosecond), nil
}

func timeFromComponents(hour, minute, second, nanosecond int) Time {
	return Time{hour: hour, minute: minute, second: second, nanosecond: nanosecond}
}

// TimeFromTime extracts the clock time from a time.Time.
func TimeFromTime(t time.Time) Time {
	return timeFromComponents(t.Hour(), t.Minute(), t.Second(), t.Nanosecond())
}

// Hour returns the hour component (0-23).
func (t Time) Hour() int { return t.hour }

// Minute returns the minute component (0-59).
func (t Time) Minute() int { return t.minute }

// Second returns the second component (0-59).
func (t Time) Second() int { return t.second }

// Nanosecond returns the nanosecond component (0-999999999).
func (t Time) Nanosecond() int { return t.nanosecond }

// IsZero reports whether t is the zero value (midnight, 00:00:00.000000000).
func (t Time) IsZero() bool {
	return t.hour == 0 && t.minute == 0 && t.second == 0 && t.nanosecond == 0
}

// toNano converts the clock time to nanoseconds since midnight for comparison.
func (t Time) toNano() int64 {
	return int64(t.hour)*int64(time.Hour) +
		int64(t.minute)*int64(time.Minute) +
		int64(t.second)*int64(time.Second) +
		int64(t.nanosecond)
}

// Equal reports whether t and other represent the same clock time.
func (t Time) Equal(other Time) bool {
	return t.hour == other.hour && t.minute == other.minute &&
		t.second == other.second && t.nanosecond == other.nanosecond
}

// Before reports whether t is before other.
func (t Time) Before(other Time) bool {
	return t.toNano() < other.toNano()
}

// After reports whether t is after other.
func (t Time) After(other Time) bool {
	return t.toNano() > other.toNano()
}

// String returns the canonical clock time as "HH:MM:SS[.fraction]".
func (t Time) String() string {
	if t.nanosecond == 0 {
		return fmt.Sprintf("%02d:%02d:%02d", t.hour, t.minute, t.second)
	}
	frac := strings.TrimRight(fmt.Sprintf("%09d", t.nanosecond), "0")
	return fmt.Sprintf("%02d:%02d:%02d.%s", t.hour, t.minute, t.second, frac)
}

// MarshalJSON encodes t as {"kind":"time","value":"HH:MM:SS[.fraction]"}.
func (t Time) MarshalJSON() ([]byte, error) {
	if msg := validateTimeComponents(t.hour, t.minute, t.second, t.nanosecond); msg != "" {
		return nil, newTimeError(
			ErrInvalidTime,
			msg,
			t.String(),
			"marshal a Time constructed with NewTime or NewTimeNanos",
		)
	}
	return json.Marshal(struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}{Kind: "time", Value: t.String()})
}

// UnmarshalJSON decodes t from {"kind":"time","value":"HH:MM:SS[.nnnnnnnnn]"}.
func (t *Time) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("time", wire.Kind, "time"); err != nil {
		return err
	}
	if err := requireJSONString("time", "value", wire.Value); err != nil {
		return err
	}
	parsed, err := time.Parse("15:04:05", wire.Value)
	if err != nil {
		return newTimeErrorWithCause(
			ErrInvalidTime,
			err,
			"time value is not a valid clock time",
			wire.Value,
			"use a clock time such as 13:30:45 or 13:30:45.123",
		)
	}
	parsedTime := TimeFromTime(parsed)
	if canonical := parsedTime.String(); wire.Value != canonical {
		return newTimeError(
			ErrInvalidFormat,
			"time value is not canonical",
			wire.Value,
			fmt.Sprintf("use %q for this time value", canonical),
		)
	}
	*t = parsedTime
	return nil
}

func validateTimeComponents(hour, minute, second, nanosecond int) string {
	switch {
	case hour < 0 || hour > 23:
		return fmt.Sprintf("invalid hour %d: must be 0-23", hour)
	case minute < 0 || minute > 59:
		return fmt.Sprintf("invalid minute %d: must be 0-59", minute)
	case second < 0 || second > 59:
		return fmt.Sprintf("invalid second %d: must be 0-59", second)
	case nanosecond < 0 || nanosecond > 999_999_999:
		return fmt.Sprintf("invalid nanosecond %d: must be 0-999999999", nanosecond)
	default:
		return ""
	}
}
