package gotime

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/text/language"
)

func TestParse_SlashDate_NoLocale_BothValid_ReturnsAmbiguous(t *testing.T) {
	// With no locale and both month-first and day-first interpretations valid,
	// return Ambiguous candidates so the caller can disambiguate.
	r := Parse("04/05/2026")
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v, want Ambiguous", r.Status)
	}
	if len(r.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(r.Candidates))
	}
	dates := make([]Date, 0, len(r.Candidates))
	for _, c := range r.Candidates {
		if c.Status != StatusResolved || c.Kind != KindDate {
			t.Fatalf("candidate status/kind = %q/%q, want resolved/date", c.Status, c.Kind)
		}
		d, ok := c.Date()
		if !ok {
			t.Fatal("candidate Date() ok=false, want true")
		}
		dates = append(dates, d)
		if len(c.Warnings) == 0 || c.Warnings[0].Message == "" {
			t.Error("candidate Warnings must not be empty")
		}
	}
	if dates[0].Equal(dates[1]) {
		t.Fatalf("candidate dates are equal: %v", dates[0])
	}
}

func TestParse_SlashDate_LocaleFirstUS(t *testing.T) {
	// en-US → month-first → April 5
	r := Parse("04/05/2026", WithInputLocale(language.MustParse("en-US")))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindDate {
		t.Fatalf("kind = %v, want KindDate", r.Kind)
	}
	d, ok := r.Date()
	if !ok {
		t.Fatalf("Date() ok=false, want true")
	}
	if d.Month() != time.April || d.Day() != 5 {
		t.Errorf("date = %v, want April 5 (month-first for en-US)", d)
	}
}

func TestParse_SlashDate_LocaleFirstGB(t *testing.T) {
	// en-GB → day-first → May 4
	r := Parse("04/05/2026", WithInputLocale(language.MustParse("en-GB")))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	d, ok := r.Date()
	if !ok {
		t.Fatalf("Date() ok=false, want true")
	}
	if d.Month() != time.May || d.Day() != 4 {
		t.Errorf("date = %v, want May 4 (day-first for en-GB)", d)
	}
}

func TestParse_SlashDate_ClosedLocalePolicy(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		want      Status
		wantMonth time.Month
		wantDay   int
	}{
		{name: "US month first", tag: "en-US", want: StatusResolved, wantMonth: time.April, wantDay: 5},
		{name: "GB day first", tag: "en-GB", want: StatusResolved, wantMonth: time.May, wantDay: 4},
		{name: "AU day first", tag: "en-AU", want: StatusResolved, wantMonth: time.May, wantDay: 4},
		{name: "CA uses no slash policy", tag: "en-CA", want: StatusAmbiguous},
		{name: "bare English", tag: "en", want: StatusAmbiguous},
		{name: "unknown locale", tag: "fr-FR", want: StatusAmbiguous},
		{name: "canonical case and Unicode extension", tag: "EN-us-u-ca-gregory", want: StatusResolved, wantMonth: time.April, wantDay: 5},
		{name: "explicit Latin script", tag: "en-Latn-AU", want: StatusResolved, wantMonth: time.May, wantDay: 4},
		{name: "unrelated script", tag: "en-Cyrl-US", want: StatusAmbiguous},
		{name: "private extension", tag: "en-US-x-private", want: StatusAmbiguous},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Parse("04/05/2026", WithInputLocale(language.MustParse(tc.tag)))
			if r.Status != tc.want {
				t.Fatalf("Parse(locale %q) status = %v, want %v", tc.tag, r.Status, tc.want)
			}
			if tc.want != StatusResolved {
				return
			}
			d, ok := r.Date()
			if !ok {
				t.Fatalf("Parse(locale %q).Date() ok = false", tc.tag)
			}
			if d.Month() != tc.wantMonth || d.Day() != tc.wantDay {
				t.Fatalf("Parse(locale %q) date = %v, want %s %d", tc.tag, d, tc.wantMonth, tc.wantDay)
			}
		})
	}
}

func TestParse_SlashDate_UnsupportedLocaleUsesValidityInference(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Status
	}{
		{name: "one valid interpretation", input: "13/02/2026", want: StatusResolved},
		{name: "no valid interpretation", input: "31/02/2026", want: StatusInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Parse(tc.input, WithInputLocale(language.MustParse("fr-FR")))
			if r.Status != tc.want {
				t.Fatalf("Parse(%q, fr-FR) status = %v, want %v", tc.input, r.Status, tc.want)
			}
		})
	}
}

func TestParse_SlashDate_NoLocale_OnlyOneValid_AutoResolves(t *testing.T) {
	t.Parallel()

	// When only one of month-first/day-first parses as a valid calendar date,
	// the parser resolves automatically without requiring a locale hint.
	cases := []struct {
		name      string
		input     string
		wantMonth time.Month
		wantDay   int
	}{
		{name: "day first only", input: "13/02/2026", wantMonth: time.February, wantDay: 13},
		{name: "month first only", input: "02/13/2026", wantMonth: time.February, wantDay: 13},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
			}
			d, ok := r.Date()
			if !ok {
				t.Fatalf("Date() ok=false, want true")
			}
			if d.Month() != tc.wantMonth || d.Day() != tc.wantDay {
				t.Fatalf("date = %v, want %s %d", d, tc.wantMonth, tc.wantDay)
			}
			if hasWarning(r.Warnings, WarnInferredCalendar) {
				t.Fatalf("warnings = %v, do not want WarnInferredCalendar", r.Warnings)
			}
		})
	}
}

func TestParse_SlashDate_InvalidDateReportsActionableError(t *testing.T) {
	t.Parallel()

	cases := []string{"31/02/2026", "13/13/2026"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			r := Parse(input, WithInputLocale(language.MustParse("en-GB")))
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error == nil {
				t.Fatal("Error is nil, want TimeError")
			}
			if r.Error.Code != CodeInvalidDate {
				t.Fatalf("error code = %q, want %q", r.Error.Code, CodeInvalidDate)
			}
			if r.Error.Hint == "" {
				t.Fatal("error Hint is empty")
			}
		})
	}
}

func TestParse_SlashDate_NoLocale_BothInvalid_ReturnsInvalid(t *testing.T) {
	t.Parallel()

	r := Parse("31/02/2026")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error == nil || r.Error.Code != CodeInvalidDate {
		t.Fatalf("error = %#v, want CodeInvalidDate", r.Error)
	}
	if r.Error.Hint == "" {
		t.Fatal("error Hint is empty")
	}
	if len(r.Candidates) != 0 {
		t.Fatalf("len(Candidates) = %d, want 0", len(r.Candidates))
	}
}

func TestParseDate_SlashDateValidInterpretationCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "zero", input: "31/02/2026", wantErr: ErrInvalidDate},
		{name: "one", input: "13/02/2026"},
		{name: "two", input: "04/05/2026", wantErr: ErrAmbiguousDate},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseDate(tc.input)
			if tc.wantErr == nil && err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tc.input, err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseDate(%q) error = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestParse_Invalid_Unparseable(t *testing.T) {
	r := Parse("not a time")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error == nil {
		t.Fatal("Error must not be nil")
	}
	if r.Error.Code != CodeUnparseable {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeUnparseable)
	}
	if r.Error.Hint == "" {
		t.Error("error Hint must not be empty")
	}
}

func TestParse_Invalid_Empty(t *testing.T) {
	r := Parse("")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeEmptyInput {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeEmptyInput)
	}
}

func TestParse_Invalid_InputPreserved(t *testing.T) {
	input := "not a time"
	r := Parse(input)
	if r.Input != input {
		t.Errorf("Input = %q, want %q", r.Input, input)
	}
	if r.Error.Input != input {
		t.Errorf("Error.Input = %q, want %q", r.Error.Input, input)
	}
}

func TestParse_Invalid_HintNeverEmpty(t *testing.T) {
	inputs := []string{"", "xyz", "99:99", "2026-99-01"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				return // not invalid, skip
			}
			if r.Error.Hint == "" {
				t.Errorf("Parse(%q) error Hint is empty", input)
			}
		})
	}
}
