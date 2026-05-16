package gotime

import (
	"errors"
	"testing"

	"golang.org/x/text/language"
)

func TestParseInstant_Resolves(t *testing.T) {
	i, err := ParseInstant("2026-03-27T04:00:00Z")
	if err != nil {
		t.Fatalf("ParseInstant error: %v", err)
	}
	if i.IsZero() {
		t.Error("ParseInstant returned zero Instant")
	}
}

func TestParseInstant_KindMismatch(t *testing.T) {
	// A bare date parses as KindDate; ParseInstant must reject with ErrIncompatibleTypes.
	_, err := ParseInstant("2026-03-27")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrIncompatibleTypes) {
		t.Errorf("expected ErrIncompatibleTypes, got %v", err)
	}
}

func TestParseDateTime_Resolves(t *testing.T) {
	dt, err := ParseDateTime("2026-03-27T13:00:00+09:00")
	if err != nil {
		t.Fatalf("ParseDateTime error: %v", err)
	}
	if dt.IsZero() {
		t.Error("ParseDateTime returned zero DateTime")
	}
}

func TestParseDate_Resolves(t *testing.T) {
	d, err := ParseDate("2026-03-27")
	if err != nil {
		t.Fatalf("ParseDate error: %v", err)
	}
	if d.Year() != 2026 {
		t.Errorf("Year() = %d, want 2026", d.Year())
	}
}

func TestParseDate_AmbiguousReturnsError(t *testing.T) {
	// 04/05/2026 with no locale → Ambiguous (both interpretations valid)
	_, err := ParseDate("04/05/2026")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAmbiguousDate) {
		t.Errorf("expected ErrAmbiguousDate, got %v", err)
	}
}

func TestParseDate_ResolvesWithLocale(t *testing.T) {
	d, err := ParseDate("04/05/2026", WithInputLocale(language.MustParse("en-US")))
	if err != nil {
		t.Fatalf("ParseDate error: %v", err)
	}
	if d.Month() != 4 || d.Day() != 5 {
		t.Errorf("date = %v, want April 5", d)
	}
}

func TestParseTime_Resolves(t *testing.T) {
	tm, err := ParseTime("13:45:30")
	if err != nil {
		t.Fatalf("ParseTime error: %v", err)
	}
	if tm.Hour() != 13 || tm.Minute() != 45 || tm.Second() != 30 {
		t.Errorf("time = %v, want 13:45:30", tm)
	}
}

func TestParseDuration_Resolves(t *testing.T) {
	d, err := ParseDuration("PT1H30M")
	if err != nil {
		t.Fatalf("ParseDuration error: %v", err)
	}
	if d != 90*Minute {
		t.Errorf("Duration = %v, want 90m", d)
	}
}

func TestParseDuration_RejectsPeriodInput(t *testing.T) {
	// P1Y parses as Period; ParseDuration must reject with ErrIncompatibleTypes.
	_, err := ParseDuration("P1Y")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrIncompatibleTypes) {
		t.Errorf("expected ErrIncompatibleTypes, got %v", err)
	}
}

func TestParsePeriod_Resolves(t *testing.T) {
	p, err := ParsePeriod("P1Y3M")
	if err != nil {
		t.Fatalf("ParsePeriod error: %v", err)
	}
	if p.Years != 1 || p.Months != 3 {
		t.Errorf("Period = %v, want P1Y3M", p)
	}
}

func TestParsePeriod_RejectsDurationInput(t *testing.T) {
	_, err := ParsePeriod("PT1H")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrIncompatibleTypes) {
		t.Errorf("expected ErrIncompatibleTypes, got %v", err)
	}
}

func TestParseInterval_Resolves(t *testing.T) {
	iv, err := ParseInterval("2026-03-27T00:00:00Z/2026-03-28T00:00:00Z")
	if err != nil {
		t.Fatalf("ParseInterval error: %v", err)
	}
	if iv.IsZero() {
		t.Error("ParseInterval returned zero Interval")
	}
}

func TestParseDate_InvalidPropagatesError(t *testing.T) {
	_, err := ParseDate("not a date")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var te *TimeError
	if !errors.As(err, &te) {
		t.Fatalf("expected *TimeError, got %T", err)
	}
	if te.Hint == "" {
		t.Error("error Hint must not be empty")
	}
}
