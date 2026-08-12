package natural

import (
	"math"
	"testing"
	"time"
)

func TestRelativeUnitResult_CheckedBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		unit      string
		count     int64
		wantKind  Kind
		wantError ErrorKind
	}{
		{name: "zero exact duration", unit: "hour", count: 0, wantKind: KindDuration},
		{name: "seconds", unit: "second", count: 2, wantKind: KindDuration},
		{name: "year maximum", unit: "year", count: math.MaxInt32, wantKind: KindPeriod},
		{name: "year overflow", unit: "year", count: math.MaxInt32 + 1, wantKind: KindInvalid, wantError: ErrorOverflow},
		{name: "month minimum", unit: "month", count: math.MinInt32, wantKind: KindPeriod},
		{name: "month overflow", unit: "month", count: math.MinInt32 - 1, wantKind: KindInvalid, wantError: ErrorOverflow},
		{name: "week multiplication overflow", unit: "week", count: math.MaxInt64, wantKind: KindInvalid, wantError: ErrorOverflow},
		{name: "duration multiplication overflow", unit: "minute", count: math.MaxInt64, wantKind: KindInvalid, wantError: ErrorOverflow},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := relativeUnitResult(tc.unit, tc.count)
			if got.Kind != tc.wantKind || got.ErrorKind != tc.wantError {
				t.Fatalf("relativeUnitResult(%q, %d) = %#v, want kind/error %v/%v", tc.unit, tc.count, got, tc.wantKind, tc.wantError)
			}
			if got.Kind == KindInvalid && got.ErrHint == "" {
				t.Fatalf("relativeUnitResult(%q, %d).ErrHint is empty", tc.unit, tc.count)
			}
		})
	}
}

func TestNaturalCurrentWeekFromSunday(t *testing.T) {
	t.Parallel()

	ctx := Context{
		Locale:     "en",
		RelativeTo: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	}
	r, ok := Parse("this Monday", ctx)
	if !ok || r.Kind != KindDate || r.Year != 2026 || r.Month != time.March || r.Day != 30 {
		t.Fatalf("Parse(this Monday) = %#v/%v, want 2026-03-30", r, ok)
	}
}

func mustLoadLocation(t *testing.T, id string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(id)
	if err != nil {
		t.Fatalf("time.LoadLocation(%q): %v", id, err)
	}
	return loc
}

func equalCivil(r Result, want time.Time) bool {
	return r.Year == want.Year() &&
		r.Month == want.Month() &&
		r.Day == want.Day() &&
		r.Hour == want.Hour() &&
		r.Minute == want.Minute() &&
		r.Second == want.Second() &&
		r.Nanosecond == want.Nanosecond()
}

func civilTime(r Result) time.Time {
	return time.Date(r.Year, r.Month, r.Day, r.Hour, r.Minute, r.Second, r.Nanosecond, time.UTC)
}
