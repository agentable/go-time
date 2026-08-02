package gotime

import (
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func TestJSONUnmarshalStructuralErrors(t *testing.T) {
	t.Parallel()

	date := mustDate(2026, time.March, 27)
	clock := mustTime(13, 30, 45)
	tests := []struct {
		name      string
		kind      string
		fields    map[string]any
		target    any
		unchanged func() bool
	}{
		newJSONStructuralCase("date", "date", map[string]any{"value": "2026-03-27"}, date),
		newJSONStructuralCase("time", "time", map[string]any{"value": "13:30:45"}, clock),
		newJSONStructuralCase(
			"local datetime",
			"local_datetime",
			map[string]any{"value": "2026-03-27T13:30:45"},
			NewLocalDateTime(date, clock),
		),
		newJSONStructuralCase(
			"instant",
			"instant",
			map[string]any{"iso": "2026-03-27T13:30:45Z"},
			UnixNanos(1),
		),
		newJSONStructuralCase(
			"datetime",
			"datetime",
			map[string]any{"instant": "2026-03-27T13:30:45Z", "zone": "UTC"},
			mustDateTime(date, clock, UTC),
		),
		newJSONStructuralCase("duration", "duration", map[string]any{"iso": "PT1H"}, 90*Minute),
		newJSONStructuralCase("period", "period", map[string]any{"iso": "P1D"}, Period{Years: 1}),
		newJSONStructuralCase(
			"interval",
			"interval",
			map[string]any{"start": "1970-01-01T00:00:00Z", "end": "1970-01-01T00:00:01Z"},
			mustInterval(t, UnixNanos(0), UnixNanos(1)),
		),
		newJSONStructuralCase(
			"zone",
			"zone",
			map[string]any{"id": "UTC"},
			MustLoadZone("Asia/Tokyo"),
		),
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			valid := maps.Clone(tc.fields)
			valid["kind"] = tc.kind

			wrongType := maps.Clone(tc.fields)
			wrongType["kind"] = false

			unknown := maps.Clone(valid)
			unknown["unexpected"] = true

			wrongKind := maps.Clone(tc.fields)
			wrongKind["kind"] = "wrong"

			cases := []struct {
				name      string
				input     []byte
				causeType string
			}{
				{name: "malformed", input: []byte(`{"kind":`), causeType: "syntax"},
				{name: "wrong field type", input: mustJSON(t, wrongType), causeType: "semantic"},
				{name: "unknown member", input: mustJSON(t, unknown), causeType: "semantic"},
				{name: "missing required field", input: mustJSON(t, map[string]any{"kind": tc.kind})},
				{name: "wrong kind", input: mustJSON(t, wrongKind)},
			}
			for _, failure := range cases {
				t.Run(failure.name, func(t *testing.T) {
					err := json.Unmarshal(failure.input, tc.target)
					assertJSONStructuralError(t, err, failure.causeType)
					if !tc.unchanged() {
						t.Fatalf("Unmarshal(%s) changed the receiver", failure.input)
					}
				})
			}
		})
	}
}

func TestJSONUnmarshalSemanticErrors(t *testing.T) {
	t.Parallel()

	date := mustDate(2026, time.March, 27)
	clock := mustTime(13, 30, 45)
	tests := []struct {
		name      string
		input     string
		want      error
		target    any
		unchanged func() bool
	}{
		newJSONSemanticCase(
			"date",
			`{"kind":"date","value":"2026-02-30"}`,
			ErrInvalidDate,
			date,
		),
		newJSONSemanticCase(
			"time",
			`{"kind":"time","value":"25:00:00"}`,
			ErrInvalidTime,
			clock,
		),
		newJSONSemanticCase(
			"local datetime date",
			`{"kind":"local_datetime","value":"2026-02-30T13:30:45"}`,
			ErrInvalidDate,
			NewLocalDateTime(date, clock),
		),
		newJSONSemanticCase(
			"local datetime time",
			`{"kind":"local_datetime","value":"2026-03-27T25:30:45"}`,
			ErrInvalidTime,
			NewLocalDateTime(date, clock),
		),
		newJSONSemanticCase(
			"instant",
			`{"kind":"instant","iso":"not-an-instant"}`,
			ErrInvalidFormat,
			UnixNanos(1),
		),
		newJSONSemanticCase(
			"datetime",
			`{"kind":"datetime","instant":"not-an-instant","zone":"UTC"}`,
			ErrInvalidFormat,
			mustDateTime(date, clock, UTC),
		),
		newJSONSemanticCase(
			"duration",
			`{"kind":"duration","iso":"P1D"}`,
			ErrInvalidDuration,
			90*Minute,
		),
		newJSONSemanticCase(
			"period",
			`{"kind":"period","iso":"PT1H"}`,
			ErrInvalidPeriod,
			Period{Years: 1},
		),
		newJSONSemanticCase(
			"interval",
			`{"kind":"interval","start":"not-an-instant","end":"1970-01-01T00:00:01Z"}`,
			ErrInvalidFormat,
			mustInterval(t, UnixNanos(0), UnixNanos(1)),
		),
		newJSONSemanticCase(
			"zone",
			`{"kind":"zone","id":"Mars/Olympus"}`,
			ErrInvalidZone,
			MustLoadZone("Asia/Tokyo"),
		),
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := json.Unmarshal([]byte(tc.input), tc.target)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Unmarshal error = %v, want %v", err, tc.want)
			}
			var detail *TimeError
			if !errors.As(err, &detail) {
				t.Fatalf("Unmarshal error = %v, want *TimeError", err)
			}
			if detail.Hint == "" {
				t.Fatalf("Unmarshal detail = %#v, want non-empty hint", detail)
			}
			causes, ok := detail.Unwrap().(interface{ Unwrap() []error })
			if !ok || len(causes.Unwrap()) < 2 {
				t.Fatalf("Unmarshal detail = %#v, want sentinel and underlying cause", detail)
			}
			if !tc.unchanged() {
				t.Fatalf("Unmarshal(%s) changed the receiver", tc.input)
			}
		})
	}
}

func newJSONStructuralCase[T comparable](
	name, kind string,
	fields map[string]any,
	initial T,
) struct {
	name      string
	kind      string
	fields    map[string]any
	target    any
	unchanged func() bool
} {
	target := initial
	return struct {
		name      string
		kind      string
		fields    map[string]any
		target    any
		unchanged func() bool
	}{
		name:      name,
		kind:      kind,
		fields:    fields,
		target:    &target,
		unchanged: func() bool { return target == initial },
	}
}

func newJSONSemanticCase[T comparable](name, input string, want error, initial T) struct {
	name      string
	input     string
	want      error
	target    any
	unchanged func() bool
} {
	target := initial
	return struct {
		name      string
		input     string
		want      error
		target    any
		unchanged func() bool
	}{
		name:      name,
		input:     input,
		want:      want,
		target:    &target,
		unchanged: func() bool { return target == initial },
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%v) error = %v", value, err)
	}
	return b
}

func assertJSONStructuralError(t *testing.T, err error, causeType string) {
	t.Helper()

	if causeType == "syntax" {
		var cause *jsontext.SyntacticError
		if !errors.As(err, &cause) {
			t.Fatalf("Unmarshal error = %v, want *jsontext.SyntacticError", err)
		}
		var detail *TimeError
		if errors.As(err, &detail) {
			t.Fatalf("Unmarshal error = %v, malformed JSON is rejected before the type decoder", err)
		}
		return
	}

	if !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("Unmarshal error = %v, want ErrInvalidFormat", err)
	}
	var detail *TimeError
	if !errors.As(err, &detail) {
		t.Fatalf("Unmarshal error = %v, want *TimeError", err)
	}
	if detail.Code != CodeInvalidFormat || detail.Hint == "" {
		t.Fatalf("Unmarshal detail = %#v, want CodeInvalidFormat and non-empty hint", detail)
	}
	if causeType == "semantic" {
		var cause *json.SemanticError
		if !errors.As(err, &cause) {
			t.Fatalf("Unmarshal error = %v, want wrapped *json.SemanticError", err)
		}
	}
}
