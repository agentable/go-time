package gotime

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestInstantFromTime_ForcesUTC(t *testing.T) {
	loc := time.FixedZone("JST", 9*3600)
	ts := time.Date(2026, 3, 27, 13, 0, 0, 0, loc)
	i := InstantFromTime(ts)
	if i.Std().Location() != time.UTC {
		t.Errorf("Location() should be UTC, got %v", i.Std().Location())
	}
}

func TestUnixNanos_Epoch(t *testing.T) {
	i := UnixNanos(0)
	got, err := i.UnixNano()
	if err != nil {
		t.Fatalf("UnixNano() error = %v", err)
	}
	if got != 0 {
		t.Errorf("UnixNano() = %d, want 0", got)
	}
	want := time.Unix(0, 0).UTC()
	if !i.Std().Equal(want) {
		t.Errorf("time mismatch: got %v, want %v", i.Std(), want)
	}
}

func TestUnixMillis(t *testing.T) {
	i := UnixMillis(1000)
	gotNanos, err := i.UnixNano()
	if err != nil {
		t.Fatalf("UnixNano() error = %v", err)
	}
	if gotNanos != 1_000_000_000 {
		t.Errorf("UnixNano() = %d, want 1_000_000_000", gotNanos)
	}
	gotMillis, err := i.UnixMilli()
	if err != nil {
		t.Fatalf("UnixMilli() error = %v", err)
	}
	if gotMillis != 1000 {
		t.Errorf("UnixMilli() = %d, want 1000", gotMillis)
	}
}

func TestUnixSeconds(t *testing.T) {
	t.Parallel()

	i := UnixSeconds(1234)
	want := time.Unix(1234, 0).UTC()
	if got := i.Std(); !got.Equal(want) {
		t.Errorf("Std() = %v, want %v", got, want)
	}
	got, err := i.UnixNano()
	if err != nil {
		t.Fatalf("UnixNano() error = %v", err)
	}
	if got != 1_234_000_000_000 {
		t.Errorf("UnixNano() = %d, want 1_234_000_000_000", got)
	}
}

func TestInstant_UnixNanoBoundaries(t *testing.T) {
	t.Parallel()

	for _, want := range []int64{math.MinInt64, math.MaxInt64} {
		instant := UnixNanos(want)
		got, err := instant.UnixNano()
		if err != nil {
			t.Fatalf("UnixNanos(%d).UnixNano() error = %v", want, err)
		}
		if got != want {
			t.Errorf("UnixNanos(%d).UnixNano() = %d, want %d", want, got, want)
		}
		if !UnixNanos(got).Equal(instant) {
			t.Errorf("UnixNanos(%d) did not round-trip", got)
		}
	}
}

func TestInstant_UnixNanoRejectsOverflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		instant Instant
	}{
		{name: "below minimum", instant: UnixNanos(math.MinInt64).Add(-Nanosecond)},
		{name: "above maximum", instant: UnixNanos(math.MaxInt64).Add(Nanosecond)},
		{name: "zero value", instant: Instant{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.instant.UnixNano()
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("UnixNano() error = %v, want ErrOverflow", err)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("UnixNano() error = %#v, want TimeError with hint", err)
			}
		})
	}
}

func TestInstant_UnixMilliBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		instant Instant
		want    int64
	}{
		{name: "minimum", instant: UnixMillis(math.MinInt64), want: math.MinInt64},
		{
			name:    "maximum",
			instant: UnixMillis(math.MaxInt64).Add(Millisecond - Nanosecond),
			want:    math.MaxInt64,
		},
	}
	for _, tc := range tests {
		got, err := tc.instant.UnixMilli()
		if err != nil {
			t.Fatalf("%s UnixMilli() error = %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s UnixMilli() = %d, want %d", tc.name, got, tc.want)
		}
		if !UnixMillis(got).Equal(InstantFromTime(tc.instant.Std().Truncate(time.Millisecond))) {
			t.Errorf("%s UnixMillis(%d) did not round-trip to millisecond precision", tc.name, got)
		}
	}
}

func TestInstant_UnixMilliRejectsOverflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		instant Instant
	}{
		{name: "below minimum", instant: UnixMillis(math.MinInt64).Add(-Nanosecond)},
		{name: "above maximum", instant: UnixMillis(math.MaxInt64).Add(Millisecond)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.instant.UnixMilli()
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("UnixMilli() error = %v, want ErrOverflow", err)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("UnixMilli() error = %#v, want TimeError with hint", err)
			}
		})
	}
}

func TestInstant_ZeroUnixMilliIsRepresentable(t *testing.T) {
	t.Parallel()

	var instant Instant
	got, err := instant.UnixMilli()
	if err != nil {
		t.Fatalf("UnixMilli() error = %v", err)
	}
	if want := instant.Std().UnixMilli(); got != want {
		t.Errorf("UnixMilli() = %d, want %d", got, want)
	}
}

func TestNow_InstantNonZero(t *testing.T) {
	i := Now()
	if i.IsZero() {
		t.Error("Now() should not be zero")
	}
	// Should be close to current time
	diff := time.Since(i.Std())
	if diff < 0 || diff > time.Second {
		t.Errorf("Now() is not close to current time: diff=%v", diff)
	}
}

func TestInstant_AddSub(t *testing.T) {
	i := UnixNanos(0)
	i2 := i.Add((1 * Hour))
	d, err := i2.Sub(i)
	if err != nil {
		t.Fatalf("Sub after Add((1 * Hour)) error = %v", err)
	}
	if d.InHours() != 1.0 {
		t.Errorf("Sub after Add((1 * Hour)) = %v hours, want 1.0", d.InHours())
	}

	back, err := i.Sub(i2)
	if err != nil {
		t.Fatalf("reverse Sub error = %v", err)
	}
	if back != -d {
		t.Errorf("reverse Sub = %v, want %v", back, -d)
	}
}

func TestInstant_SubExactDurationBoundaries(t *testing.T) {
	t.Parallel()

	start := UnixNanos(0)
	for _, want := range []Duration{Duration(math.MinInt64), Duration(math.MaxInt64)} {
		end := start.Add(want)
		got, err := end.Sub(start)
		if err != nil {
			t.Fatalf("Sub(%v) error = %v", want, err)
		}
		if got != want {
			t.Errorf("Sub(%v) = %v, want %v", want, got, want)
		}
		if !start.Add(got).Equal(end) {
			t.Errorf("start.Add(Sub) = %v, want %v", start.Add(got), end)
		}
	}
}

func TestInstant_SubRejectsOverflow(t *testing.T) {
	t.Parallel()

	start := UnixNanos(0)
	tests := []struct {
		name string
		end  Instant
	}{
		{name: "positive", end: start.Add(Duration(math.MaxInt64)).Add(Nanosecond)},
		{name: "negative", end: start.Add(Duration(math.MinInt64)).Add(-Nanosecond)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.end.Sub(start)
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("Sub() error = %v, want ErrOverflow", err)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("Sub() error = %#v, want TimeError with hint", err)
			}
		})
	}
}

func TestInstant_BeforeAfterEqual(t *testing.T) {
	past := UnixNanos(0)
	future := UnixNanos(1_000_000_000)

	if !past.Before(future) {
		t.Error("past.Before(future) should be true")
	}
	if past.After(future) {
		t.Error("past.After(future) should be false")
	}
	if past.Equal(future) {
		t.Error("past.Equal(future) should be false")
	}
	if !past.Equal(UnixNanos(0)) {
		t.Error("same instants should be equal")
	}
}

func TestInstant_Compare(t *testing.T) {
	i1 := UnixNanos(0)
	i2 := UnixNanos(1000)

	if i1.Compare(i2) != -1 {
		t.Errorf("Compare(before) = %d, want -1", i1.Compare(i2))
	}
	if i2.Compare(i1) != 1 {
		t.Errorf("Compare(after) = %d, want 1", i2.Compare(i1))
	}
	i1Copy := UnixNanos(0)
	if i1.Compare(i1Copy) != 0 {
		t.Errorf("Compare(equal) = %d, want 0", i1.Compare(i1Copy))
	}
}

func TestInstant_IsZero(t *testing.T) {
	var i Instant
	if !i.IsZero() {
		t.Error("zero Instant should be zero")
	}
	if UnixNanos(1).IsZero() {
		t.Error("non-zero Instant should not be zero")
	}
}

func TestInstant_In(t *testing.T) {
	i := UnixNanos(0)
	dt, err := i.In(UTC)
	if err != nil {
		t.Fatalf("In(UTC) error = %v", err)
	}
	if !dt.Instant().Equal(i) {
		t.Error("In(UTC).Instant() should round-trip")
	}
}

func TestInstant_InRejectsProjectedYearOutsideCivilDomain(t *testing.T) {
	tests := []struct {
		name    string
		instant Instant
		zone    Zone
	}{
		{
			name:    "below minimum after westward projection",
			instant: InstantFromTime(time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)),
			zone:    MustLoadZone("America/New_York"),
		},
		{
			name:    "above maximum after eastward projection",
			instant: InstantFromTime(time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)),
			zone:    MustLoadZone("Asia/Tokyo"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.instant.In(tc.zone)
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("Instant.In(%v) error = %v, want ErrOverflow", tc.zone, err)
			}
		})
	}
}

func TestInstant_String(t *testing.T) {
	i := UnixNanos(0)
	s := i.String()
	// Should be a valid RFC3339Nano UTC string
	if !strings.HasSuffix(s, "Z") && !strings.Contains(s, "+00:00") {
		t.Errorf("String() = %q, expected UTC suffix", s)
	}
	// Should contain the epoch year
	if !strings.Contains(s, "1970") {
		t.Errorf("String() = %q, expected epoch year 1970", s)
	}
}
