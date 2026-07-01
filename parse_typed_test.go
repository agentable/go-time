package gotime

import (
	"errors"
	"testing"

	"golang.org/x/text/language"
)

func TestParseInstant_Resolves(t *testing.T) {
	i, err := ParseInstant("2026-03-27T13:00:00+09:00")
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
	var te *TimeError
	if !errors.As(err, &te) {
		t.Fatalf("expected *TimeError, got %T", err)
	}
	if te.Hint != "input parsed as date, call ParseDate instead" {
		t.Errorf("TimeError.Hint = %q, want ParseDate guidance", te.Hint)
	}
}

func TestParseDateTime_Resolves(t *testing.T) {
	dt, err := ParseDateTime("2026-03-27T13:00:00", WithZone(MustLoadZone("Asia/Tokyo")))
	if err != nil {
		t.Fatalf("ParseDateTime error: %v", err)
	}
	if dt.IsZero() {
		t.Error("ParseDateTime returned zero DateTime")
	}
}

func TestParseDateTime_RejectsLocalDateTimeWithoutZone(t *testing.T) {
	_, err := ParseDateTime("2026-03-27T13:00:00")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrIncompatibleTypes) {
		t.Errorf("ParseDateTime() error = %v, want ErrIncompatibleTypes", err)
	}
}

func TestParseLocalDateTime_Resolves(t *testing.T) {
	ldt, err := ParseLocalDateTime("2026-03-27T13:00:00.12345")
	if err != nil {
		t.Fatalf("ParseLocalDateTime error: %v", err)
	}
	if got := ldt.String(); got != "2026-03-27T13:00:00.12345" {
		t.Errorf("ParseLocalDateTime() = %s, want 2026-03-27T13:00:00.12345", got)
	}
}

func TestParseLocalDateTime_RejectsExplicitOffset(t *testing.T) {
	_, err := ParseLocalDateTime("2026-03-27T13:00:00+09:00")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrIncompatibleTypes) {
		t.Errorf("ParseLocalDateTime() error = %v, want ErrIncompatibleTypes", err)
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

func TestTypedParsers_ReturnErrorSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parse    func() error
		wantErr  error
		wantCode ErrorCode
	}{
		{
			name: "duplicate local datetime",
			parse: func() error {
				_, err := ParseDateTime("2026-11-01T01:30:00", WithZone(MustLoadZone("America/New_York")))
				return err
			},
			wantErr:  ErrDuplicateTime,
			wantCode: CodeDuplicateTime,
		},
		{
			name: "non-hour duplicate local datetime",
			parse: func() error {
				_, err := ParseDateTime("2026-04-05T01:45:00", WithZone(MustLoadZone("Australia/Lord_Howe")))
				return err
			},
			wantErr:  ErrDuplicateTime,
			wantCode: CodeDuplicateTime,
		},
		{
			name: "interval duplicate local start",
			parse: func() error {
				_, err := ParseInterval(
					"2026-11-01T01:30:00/2026-11-01T03:00:00",
					WithZone(MustLoadZone("America/New_York")),
				)
				return err
			},
			wantErr:  ErrDuplicateTime,
			wantCode: CodeDuplicateTime,
		},
		{
			name: "invalid clock time",
			parse: func() error {
				_, err := ParseTime("25:00")
				return err
			},
			wantErr:  ErrInvalidTime,
			wantCode: CodeInvalidTime,
		},
		{
			name: "date-only interval endpoints",
			parse: func() error {
				_, err := ParseInterval("2026-03-27/2026-03-28")
				return err
			},
			wantErr:  ErrIncompatibleTypes,
			wantCode: CodeIncompatibleTypes,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.parse()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			var te *TimeError
			if !errors.As(err, &te) {
				t.Fatalf("expected *TimeError, got %T", err)
			}
			if te.Code != tc.wantCode {
				t.Errorf("TimeError.Code = %q, want %q", te.Code, tc.wantCode)
			}
			if te.Hint == "" {
				t.Error("TimeError.Hint must not be empty")
			}
		})
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

func TestParseResultError_InvalidWithoutErrorHasHint(t *testing.T) {
	err := parseResultError(ParseResult{Status: StatusInvalid, Input: "??"}, KindDate)
	if err == nil {
		t.Fatal("parseResultError() error = nil, want error")
	}
	if !errors.Is(err, ErrUnparseable) {
		t.Fatalf("parseResultError() error = %v, want ErrUnparseable", err)
	}
	var te *TimeError
	if !errors.As(err, &te) {
		t.Fatalf("expected *TimeError, got %T", err)
	}
	if te.Hint == "" {
		t.Fatal("TimeError.Hint is empty")
	}
}
