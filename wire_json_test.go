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
		newJSONStructuralCase("period", "period", map[string]any{"value": "P1D"}, Period{Years: 1}),
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
			unknown := maps.Clone(valid)
			unknown["unexpected"] = true
			legacy := maps.Clone(valid)
			legacy["kind"] = tc.kind

			var wrongType any = false
			var missing any = ""
			if tc.kind == "datetime" || tc.kind == "interval" {
				wrong := maps.Clone(valid)
				incomplete := maps.Clone(valid)
				for field := range valid {
					wrong[field] = false
					delete(incomplete, field)
					break
				}
				wrongType = wrong
				missing = incomplete
			}

			cases := []struct {
				name      string
				input     []byte
				causeType string
			}{
				{name: "null", input: []byte(`null`)},
				{name: "boolean", input: []byte(`false`), causeType: "semantic"},
				{name: "array", input: []byte(`[]`), causeType: "semantic"},
				{name: "malformed", input: []byte(`{"kind":`), causeType: "syntax"},
				{name: "wrong field type", input: mustJSON(t, wrongType), causeType: "semantic"},
				{name: "unknown member", input: mustJSON(t, unknown), causeType: "semantic"},
				{name: "missing required field", input: mustJSON(t, missing)},
				{name: "legacy object", input: mustJSON(t, legacy)},
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
			`"2026-02-30"`,
			ErrInvalidDate,
			date,
		),
		newJSONSemanticCase(
			"time",
			`"25:00:00"`,
			ErrInvalidTime,
			clock,
		),
		newJSONSemanticCase(
			"local datetime date",
			`"2026-02-30T13:30:45"`,
			ErrInvalidDate,
			NewLocalDateTime(date, clock),
		),
		newJSONSemanticCase(
			"local datetime time",
			`"2026-03-27T25:30:45"`,
			ErrInvalidTime,
			NewLocalDateTime(date, clock),
		),
		newJSONSemanticCase(
			"instant",
			`"not-an-instant"`,
			ErrInvalidFormat,
			UnixNanos(1),
		),
		newJSONSemanticCase(
			"datetime",
			`{"instant":"not-an-instant","zone":"UTC"}`,
			ErrInvalidFormat,
			mustDateTime(date, clock, UTC),
		),
		newJSONSemanticCase(
			"duration",
			`"P1D"`,
			ErrInvalidDuration,
			90*Minute,
		),
		newJSONSemanticCase(
			"period",
			`"PT1H"`,
			ErrInvalidPeriod,
			Period{Years: 1},
		),
		newJSONSemanticCase(
			"interval",
			`{"start":"not-an-instant","end":"1970-01-01T00:00:01Z"}`,
			ErrInvalidFormat,
			mustInterval(t, UnixNanos(0), UnixNanos(1)),
		),
		newJSONSemanticCase(
			"zone",
			`"Mars/Olympus"`,
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
			name:   "zone",
			value:  sensitiveZone,
			target: new(Zone),
		},
		{
			name: "datetime",
			value: struct {
				Instant string `json:"instant"`
				Zone    string `json:"zone"`
			}{Instant: "2026-03-27T04:00:00Z", Zone: sensitiveZone},
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
				for key, value := range tc.fields {
					fields[key] = strings.ReplaceAll(value.(string), "%s", fraction)
				}
				var input any = fields
				if len(fields) == 1 {
					for _, value := range fields {
						input = value
					}
				}
				before := mustJSON(t, tc.target)
				err := json.Unmarshal(mustJSON(t, input), tc.target)
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
		input := map[string]string{"start": "2026-03-27T13:30:45" + endpoints[0] + "Z", "end": "2026-03-27T13:30:45" + endpoints[1] + "Z"}
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
				value  any
				target any
			}{
				{tc.iso, &instant},
				{map[string]string{"start": tc.iso, "end": tc.iso}, &interval},
				{map[string]string{"start": "0000-01-01T00:00:00Z", "end": tc.iso}, &interval},
				{map[string]string{"start": tc.iso, "end": "9999-12-31T23:59:59Z"}, &interval},
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

func TestJSONInstantProfile(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		"2026-10-19T12:00:00+24:00",
		"2026-10-19T12:00:00-24:00",
		"2026-10-19T12:00:00+00:60",
		"2026-10-19T12:00:00-00:60",
		"2026-10-19T1:00:00Z",
		"2026-10-19T12:00:00,1Z",
		"2026-10-19T12:00Z",
		"2026-10-19T12:00:00.1234567890Z",
		"2026-10-19T12:00:00+0100",
		"2026-10-19 12:00:00Z",
		"2026-10-19t12:00:00z",
		" 2026-10-19T12:00:00Z",
		"2026-10-19T12:00:00Z\n",
	} {
		t.Run(input, func(t *testing.T) {
			instant := UnixNanos(1)
			interval := mustInterval(t, UnixNanos(0), UnixNanos(1))
			datetime := mustDateTime(mustDate(2026, time.March, 27), mustTime(13, 30, 45), UTC)
			for _, tc := range []struct {
				name   string
				wire   any
				target any
			}{
				{"instant", input, &instant},
				{"interval start", map[string]string{"start": input, "end": "9999-12-31T23:59:59Z"}, &interval},
				{"interval end", map[string]string{"start": "0000-01-01T00:00:00Z", "end": input}, &interval},
				{"datetime", map[string]string{"instant": input, "zone": "UTC"}, &datetime},
			} {
				t.Run(tc.name, func(t *testing.T) {
					before := mustJSON(t, tc.target)
					err := json.Unmarshal(mustJSON(t, tc.wire), tc.target)
					var detail *TimeError
					if !errors.Is(err, ErrInvalidFormat) || !errors.As(err, &detail) || detail.Hint == "" {
						t.Errorf("Unmarshal(%q) = %v, want ErrInvalidFormat with hint", input, err)
					}
					if after := mustJSON(t, tc.target); !bytes.Equal(before, after) {
						t.Errorf("failed decode changed receiver: %s -> %s", before, after)
					}
				})
			}
		})
	}
}

func TestJSONInstantNormalization(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ input, want string }{
		{"2026-10-19T12:00:00+23:59", "2026-10-18T12:01:00Z"},
		{"2026-10-19T12:00:00-23:59", "2026-10-20T11:59:00Z"},
		{"2026-10-19T12:00:00.100Z", "2026-10-19T12:00:00.1Z"},
		{"2026-10-19T12:00:00-00:00", "2026-10-19T12:00:00Z"},
		{"0000-02-29T12:00:00Z", "0000-02-29T12:00:00Z"},
		{"0000-01-01T23:59:00+23:59", "0000-01-01T00:00:00Z"},
		{"9999-12-31T00:00:59.999999999-23:59", "9999-12-31T23:59:59.999999999Z"},
		{"2026-10-19T12:00:00.123456789Z", "2026-10-19T12:00:00.123456789Z"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			var instant Instant
			if err := json.Unmarshal(mustJSON(t, tc.input), &instant); err != nil {
				t.Fatal(err)
			}
			if got := instant.String(); got != tc.want {
				t.Fatalf("instant = %s, want %s", got, tc.want)
			}
			var interval Interval
			if err := json.Unmarshal(mustJSON(t, map[string]string{"start": tc.input, "end": tc.input}), &interval); err != nil {
				t.Fatal(err)
			}
			if !interval.Start().Equal(instant) || !interval.End().Equal(instant) {
				t.Fatalf("interval endpoints differ from %s: %s", instant, interval)
			}
			for _, value := range []any{&instant, &interval} {
				encoded := mustJSON(t, value)
				if err := json.Unmarshal(encoded, value); err != nil {
					t.Fatal(err)
				}
				if again := mustJSON(t, value); !bytes.Equal(encoded, again) {
					t.Fatalf("unstable wire: %s -> %s", encoded, again)
				}
			}
			datetime := mustDateTime(mustDate(2026, time.March, 27), mustTime(13, 30, 45), UTC)
			before := mustJSON(t, datetime)
			err := json.Unmarshal(mustJSON(t, map[string]string{"instant": tc.input, "zone": "UTC"}), &datetime)
			if tc.input != tc.want {
				if !errors.Is(err, ErrInvalidFormat) {
					t.Fatalf("noncanonical DateTime input error = %v", err)
				}
				if after := mustJSON(t, datetime); !bytes.Equal(before, after) {
					t.Fatal("failed DateTime decode changed receiver")
				}
			} else if err != nil || !datetime.Instant().Equal(instant) {
				t.Fatalf("canonical DateTime = %s, %v", datetime, err)
			}
		})
	}
}

func TestJSONInstantCalendarErrorKeepsCause(t *testing.T) {
	t.Parallel()
	const invalid = "2026-02-30T12:00:00Z"
	for _, tc := range []struct {
		wire   any
		target any
	}{
		{invalid, new(Instant)},
		{map[string]string{"start": invalid, "end": "2026-03-01T00:00:00Z"}, new(Interval)},
		{map[string]string{"start": "2026-01-01T00:00:00Z", "end": invalid}, new(Interval)},
		{map[string]string{"instant": invalid, "zone": "UTC"}, new(DateTime)},
	} {
		err := json.Unmarshal(mustJSON(t, tc.wire), tc.target)
		var cause *time.ParseError
		if !errors.Is(err, ErrInvalidFormat) || !errors.As(err, &cause) {
			t.Fatalf("Unmarshal(%v) = %v, want format error and stdlib parse cause", tc.wire, err)
		}
	}
}

func TestJSONInstantProfileLeavesHumanGrammarUnchanged(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"2026-10-19T12:00Z", "2026-10-19T12:00:00,1Z", " 2026-10-19T12:00:00Z "} {
		if _, err := ParseInstant(input); err != nil {
			t.Errorf("human ParseInstant(%q): %v", input, err)
		}
	}
	if _, err := ParseTime("09:30"); err != nil {
		t.Errorf("human ParseTime shorthand: %v", err)
	}
}

func TestConcreteValuesUseDirectJSON(t *testing.T) {
	d := mustDate(2026, time.October, 20)
	clock := mustTimeNanos(12, 30, 0, 100_000_000)
	instant := InstantFromTime(time.Date(2026, time.October, 20, 12, 30, 0, 0, time.UTC))
	for _, tc := range []struct {
		name   string
		value  any
		target any
		wire   string
	}{
		{"date", d, new(Date), `"2026-10-20"`},
		{"time", clock, new(Time), `"12:30:00.1"`},
		{"instant", instant, new(Instant), `"2026-10-20T12:30:00Z"`},
		{"local", NewLocalDateTime(d, clock), new(LocalDateTime), `"2026-10-20T12:30:00.1"`},
		{"duration", 1400 * Millisecond, new(Duration), `"PT1.4S"`},
		{"period", Period{Months: 1}, new(Period), `"P1M"`},
		{"zone", UTC, new(Zone), `"UTC"`},
		{"datetime", mustDateTime(d, mustTime(12, 30, 0), UTC), new(DateTime), `{"instant":"2026-10-20T12:30:00Z","zone":"UTC"}`},
		{"interval", mustInterval(t, instant, instant), new(Interval), `{"start":"2026-10-20T12:30:00Z","end":"2026-10-20T12:30:00Z"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.value)
			if err != nil || string(data) != tc.wire {
				t.Fatalf("Marshal = %s, %v; want %s", data, err, tc.wire)
			}
			if err := json.Unmarshal([]byte(tc.wire), tc.target); err != nil {
				t.Fatalf("Unmarshal = %v", err)
			}
			again, err := json.Marshal(tc.target)
			if err != nil || string(again) != tc.wire {
				t.Fatalf("round trip = %s, %v; want %s", again, err, tc.wire)
			}
		})
	}
	data, err := json.Marshal(struct {
		Due Date `json:"due"`
	}{Due: d})
	if err != nil || string(data) != `{"due":"2026-10-20"}` {
		t.Fatalf("record = %s, %v", data, err)
	}
}
