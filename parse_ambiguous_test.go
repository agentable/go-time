package gotime

import (
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
	for _, c := range r.Candidates {
		if len(c.Warnings) == 0 || c.Warnings[0].Message == "" {
			t.Error("candidate Warnings must not be empty")
		}
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

func TestParse_SlashDate_NoLocale_BothInvalid_ReturnsAmbiguousWithInvalidCandidates(t *testing.T) {
	t.Parallel()

	// "31/02/2026" — neither (Feb 31) nor (31st month) is valid; surface both
	// invalid candidates so callers can inspect why.
	r := Parse("31/02/2026")
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v, want Ambiguous", r.Status)
	}
	if len(r.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(r.Candidates))
	}
	invalidCount := 0
	for _, candidate := range r.Candidates {
		if candidate.Status != StatusInvalid {
			continue
		}
		invalidCount++
		if candidate.Error == nil || candidate.Error.Code != CodeInvalidDate {
			t.Fatalf("candidate error = %#v, want invalid date", candidate.Error)
		}
	}
	if invalidCount == 0 {
		t.Fatalf("Candidates = %#v, want at least one invalid date candidate", r.Candidates)
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
