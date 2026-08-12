package gotime

import (
	"errors"
	"testing"

	"golang.org/x/text/language"
)

func TestParse_Interval_InvalidEndpointPreservesSemanticError(t *testing.T) {
	t.Parallel()

	const valid = "2026-03-27T00:00:00"
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantCode ErrorCode
	}{
		{name: "invalid start date", input: "2026-02-30T00:00:00/" + valid, wantErr: ErrInvalidDate, wantCode: CodeInvalidDate},
		{name: "invalid end date", input: valid + "/2026-02-30T00:00:00", wantErr: ErrInvalidDate, wantCode: CodeInvalidDate},
		{name: "invalid start time", input: "2026-03-27T25:00:00/" + valid, wantErr: ErrInvalidTime, wantCode: CodeInvalidTime},
		{name: "invalid end time", input: valid + "/2026-03-28T25:00:00", wantErr: ErrInvalidTime, wantCode: CodeInvalidTime},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input, WithZone(UTC))
			if r.Status != StatusInvalid {
				t.Fatalf("Parse(%q).Status = %s, want %s", tc.input, r.Status, StatusInvalid)
			}
			if !errors.Is(r.Error, tc.wantErr) {
				t.Fatalf("Parse(%q).Error = %v, want %v", tc.input, r.Error, tc.wantErr)
			}
			if r.Error.Code != tc.wantCode {
				t.Errorf("Parse(%q).Error.Code = %q, want %q", tc.input, r.Error.Code, tc.wantCode)
			}
			if r.Error.Input != tc.input {
				t.Errorf("Parse(%q).Error.Input = %q, want complete interval", tc.input, r.Error.Input)
			}
			if r.Error.Hint == "" {
				t.Errorf("Parse(%q).Error.Hint is empty", tc.input)
			}
		})
	}
}

func TestParse_Interval_InvalidScenarios(t *testing.T) {
	t.Parallel()

	zone := MustLoadZone("America/New_York")
	tests := []struct {
		name     string
		input    string
		opts     []Option
		wantErr  error
		wantCode ErrorCode
	}{
		{
			name:     "nonexistent start",
			input:    "2026-03-08T02:30:00/2026-03-08T04:00:00",
			opts:     []Option{WithZone(zone)},
			wantErr:  ErrNonexistentTime,
			wantCode: CodeNonexistentTime,
		},
		{
			name:     "nonexistent end",
			input:    "2026-03-08T00:30:00/2026-03-08T02:30:00",
			opts:     []Option{WithZone(zone)},
			wantErr:  ErrNonexistentTime,
			wantCode: CodeNonexistentTime,
		},
		{name: "malformed start", input: "not-a-date/2026-03-27T00:00:00Z", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "malformed end", input: "2026-03-27T00:00:00Z/not-a-date", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "period is not exact duration", input: "2026-03-27T00:00:00Z/P1D", wantErr: ErrIncompatibleTypes, wantCode: CodeIncompatibleTypes},
		{name: "two durations", input: "PT1H/PT2H", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "duration overflow", input: "2026-03-27T00:00:00Z/PT999999999999999999999H", wantErr: ErrOverflow, wantCode: CodeOverflow},
		{name: "malformed start before duration", input: "not-a-date/PT1H", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "overflow duration before end", input: "PT999999999999999999999H/2026-03-27T00:00:00Z", wantErr: ErrOverflow, wantCode: CodeOverflow},
		{name: "malformed end after duration", input: "PT1H/not-a-date", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input, tc.opts...)
			if r.Status != StatusInvalid || r.Error == nil {
				t.Fatalf("Parse(%q) = %#v, want invalid result", tc.input, r)
			}
			if !errors.Is(r.Error, tc.wantErr) {
				t.Fatalf("Parse(%q).Error = %v, want %v", tc.input, r.Error, tc.wantErr)
			}
			if r.Error.Code != tc.wantCode {
				t.Errorf("Parse(%q).Error.Code = %q, want %q", tc.input, r.Error.Code, tc.wantCode)
			}
			if r.Error.Input != tc.input || r.Error.Hint == "" {
				t.Errorf("Parse(%q) error input/hint = %q/%q", tc.input, r.Error.Input, r.Error.Hint)
			}
		})
	}
}

func TestParse_Interval_DateToDate(t *testing.T) {
	r := Parse("2026-03-27/2026-03-28")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeIncompatibleTypes {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeIncompatibleTypes)
	}
}

func TestParse_Interval_DatetimeToDatetime(t *testing.T) {
	r := Parse("2026-03-27T00:00:00Z/2026-03-28T00:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindInterval {
		t.Fatalf("kind = %v, want KindInterval", r.Kind)
	}
}

func TestParse_Interval_StartDuration(t *testing.T) {
	r := Parse("2026-03-27T00:00:00Z/PT12H")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindInterval {
		t.Fatalf("kind = %v, want KindInterval", r.Kind)
	}
	iv, _ := r.Interval()
	length, err := iv.Length()
	if err != nil {
		t.Fatalf("Length() error = %v", err)
	}
	if length.InHours() != 12 {
		t.Errorf("length = %v hours, want 12", length.InHours())
	}
}

func TestParse_Interval_DurationEnd(t *testing.T) {
	r := Parse("PT12H/2026-03-28T00:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	iv, _ := r.Interval()
	length, err := iv.Length()
	if err != nil {
		t.Fatalf("Length() error = %v", err)
	}
	if length.InHours() != 12 {
		t.Errorf("length = %v hours, want 12", length.InHours())
	}
}

func TestParse_Interval_EndBeforeStart(t *testing.T) {
	r := Parse("2026-03-28T00:00:00Z/2026-03-27T00:00:00Z")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeIntervalReversed {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeIntervalReversed)
	}
}

func TestParse_Interval_AmbiguousEndpoint(t *testing.T) {
	r := Parse(
		"2026-11-01T01:30:00/2026-11-01T03:00:00",
		WithZone(MustLoadZone("America/New_York")),
	)
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v (err=%v), want Ambiguous", r.Status, r.Error)
	}
}

func TestParse_Interval_RejectsNaturalLanguageEndpoint(t *testing.T) {
	r := Parse(
		"tomorrow/2026-03-28T00:00:00Z",
		WithInputLocale(language.English),
		WithReference(fixedNow),
	)
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
}
