package gotime

import (
	"bytes"
	"errors"
	"maps"
	"strings"
	"testing"
	"time"

	"encoding/json/jsontext"
	"encoding/json/v2"
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

func TestJSONZoneErrorsKeepStructuredInputOutOfText(t *testing.T) {
	t.Parallel()

	const sensitiveZone = "secret-zone-token\n\x1b[31m"
	tests := []struct {
		name   string
		value  any
		target any
	}{
		{
			name: "zone",
			value: struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			}{Kind: "zone", ID: sensitiveZone},
			target: new(Zone),
		},
		{
			name: "datetime",
			value: struct {
				Kind    string `json:"kind"`
				Instant string `json:"instant"`
				Zone    string `json:"zone"`
			}{Kind: "datetime", Instant: "2026-03-27T04:00:00Z", Zone: sensitiveZone},
			target: new(DateTime),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			err = json.Unmarshal(payload, test.target)
			if !errors.Is(err, ErrInvalidZone) {
				t.Fatalf("Unmarshal() error = %v, want ErrInvalidZone", err)
			}
			if strings.Contains(err.Error(), "secret-zone-token") {
				t.Fatalf("Unmarshal() error exposed structured input: %q", err)
			}
			var detail *TimeError
			if !errors.As(err, &detail) {
				t.Fatalf("Unmarshal() error = %T, want *TimeError", err)
			}
			if detail.Input != sensitiveZone {
				t.Errorf("TimeError.Input = %q, want original structured input", detail.Input)
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

func TestJSONClockPrecision(t *testing.T) {
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
		newJSONStructuralCase("instant", "instant", map[string]any{"iso": "2026-03-27T13:30:45%sZ"}, UnixNanos(1)),
		newJSONStructuralCase("time", "time", map[string]any{"value": "13:30:45%s"}, clock),
		newJSONStructuralCase("local", "local_datetime", map[string]any{"value": "2026-03-27T13:30:45%s"}, NewLocalDateTime(date, clock)),
		newJSONStructuralCase("datetime", "datetime", map[string]any{"instant": "2026-03-27T13:30:45%sZ", "zone": "UTC"}, mustDateTime(date, clock, UTC)),
		newJSONStructuralCase("interval", "interval", map[string]any{"start": "2026-03-27T13:30:45%sZ", "end": "2026-03-28T13:30:45%sZ"}, mustInterval(t, UnixNanos(0), UnixNanos(1))),
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, fraction := range []string{"", ".1", ".123456789", ".1234567890", ".1234567891"} {
				fields := maps.Clone(tc.fields)
				fields["kind"] = tc.kind
				for key, value := range tc.fields {
					fields[key] = strings.ReplaceAll(value.(string), "%s", fraction)
				}
				before := mustJSON(t, tc.target)
				err := json.Unmarshal(mustJSON(t, fields), tc.target)
				if len(fraction) > 10 {
					assertJSONStructuralError(t, err, "")
					if after := mustJSON(t, tc.target); !bytes.Equal(after, before) {
						t.Fatalf("failed decode changed receiver: %s -> %s", before, after)
					}
					continue
				}
				if err != nil {
					t.Fatalf("fraction %q: %v", fraction, err)
				}
				encoded := mustJSON(t, tc.target)
				if err := json.Unmarshal(encoded, tc.target); err != nil {
					t.Fatal(err)
				}
				if again := mustJSON(t, tc.target); !bytes.Equal(again, encoded) {
					t.Fatalf("unstable round trip: %s -> %s", encoded, again)
				}
			}
		})
	}
}

func TestIntervalJSONRejectsPrecisionCollapse(t *testing.T) {
	t.Parallel()
	for _, endpoints := range [][2]string{
		{".1234567891", ".1234567892"},
		{".1234567891", ".2"},
		{".1", ".1234567892"},
	} {
		original := mustInterval(t, UnixNanos(0), UnixNanos(1))
		got := original
		input := map[string]string{"kind": "interval", "start": "2026-03-27T13:30:45" + endpoints[0] + "Z", "end": "2026-03-27T13:30:45" + endpoints[1] + "Z"}
		assertJSONStructuralError(t, json.Unmarshal(mustJSON(t, input), &got), "")
		if got != original {
			t.Fatal("failed precision decode changed interval")
		}
	}
}

func TestJSONUTCWireDomain(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		iso      string
		overflow bool
	}{
		{"0000-01-01T00:00:00+01:00", true},
		{"9999-12-31T23:59:59-01:00", true},
		{"0000-01-01T00:00:00Z", false},
		{"9999-12-31T23:59:59Z", false},
		{"0000-01-01T01:00:00+01:00", false},
		{"9999-12-31T22:59:59-01:00", false},
	} {
		t.Run(tc.iso, func(t *testing.T) {
			instant := UnixNanos(1)
			interval := mustInterval(t, UnixNanos(0), UnixNanos(1))
			for _, c := range []struct {
				value  map[string]string
				target any
			}{
				{map[string]string{"kind": "instant", "iso": tc.iso}, &instant},
				{map[string]string{"kind": "interval", "start": tc.iso, "end": tc.iso}, &interval},
				{map[string]string{"kind": "interval", "start": "0000-01-01T00:00:00Z", "end": tc.iso}, &interval},
				{map[string]string{"kind": "interval", "start": tc.iso, "end": "9999-12-31T23:59:59Z"}, &interval},
			} {
				before := mustJSON(t, c.target)
				err := json.Unmarshal(mustJSON(t, c.value), c.target)
				if tc.overflow {
					var detail *TimeError
					if !errors.Is(err, ErrOverflow) || !errors.As(err, &detail) || detail.Hint == "" {
						t.Fatalf("decode %v: %v, want overflow with hint", c.value, err)
					}
					if after := mustJSON(t, c.target); !bytes.Equal(after, before) {
						t.Fatal("overflow changed receiver")
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					encoded := mustJSON(t, c.target)
					if err := json.Unmarshal(encoded, c.target); err != nil {
						t.Fatal(err)
					}
					if again := mustJSON(t, c.target); !bytes.Equal(again, encoded) {
						t.Fatal("unstable encoding")
					}
				}
			}
		})
	}
}
