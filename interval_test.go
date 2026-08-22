package gotime

import (
	"errors"
	"testing"
	"time"

	"encoding/json/v2"
)

var (
	ivStart = UnixNanos(0)
	ivEnd   = UnixNanos((9 * Hour).Nanoseconds())
)

func mustInterval(t *testing.T, start, end Instant) Interval {
	t.Helper()

	iv, err := NewInterval(start, end)
	if err != nil {
		t.Fatalf("NewInterval(%v, %v): %v", start, end, err)
	}
	return iv
}

func mustIntervalLength(t *testing.T, iv Interval) Duration {
	t.Helper()

	length, err := iv.Length()
	if err != nil {
		t.Fatalf("Length() error = %v", err)
	}
	return length
}

func TestNewInterval_Valid(t *testing.T) {
	iv, err := NewInterval(ivStart, ivEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !iv.Start().Equal(ivStart) || !iv.End().Equal(ivEnd) {
		t.Error("Start/End should match inputs")
	}
}

func TestNewInterval_EndBeforeStart(t *testing.T) {
	_, err := NewInterval(ivEnd, ivStart)
	if !errors.Is(err, ErrIntervalReversed) {
		t.Errorf("expected ErrIntervalReversed, got %v", err)
	}
}

func TestNewInterval_SameStartEnd(t *testing.T) {
	iv, err := NewInterval(ivStart, ivStart)
	if err != nil {
		t.Fatalf("zero-length interval should succeed: %v", err)
	}
	if !mustIntervalLength(t, iv).IsZero() {
		t.Error("zero-length interval should have zero Length()")
	}
}

func TestNewIntervalStartingAt(t *testing.T) {
	iv, err := NewIntervalStartingAt(ivStart, 9*Hour)
	if err != nil {
		t.Fatalf("NewIntervalStartingAt() error = %v", err)
	}
	if got := mustIntervalLength(t, iv).InHours(); got != 9.0 {
		t.Errorf("Length() = %v hours, want 9.0", got)
	}
}

func TestNewIntervalStartingAt_NegativeLength(t *testing.T) {
	_, err := NewIntervalStartingAt(ivStart, -1*Hour)
	if !errors.Is(err, ErrInvalidDuration) {
		t.Errorf("NewIntervalStartingAt() error = %v, want ErrInvalidDuration", err)
	}
}

func TestNewIntervalEndingAt(t *testing.T) {
	iv, err := NewIntervalEndingAt(ivEnd, 9*Hour)
	if err != nil {
		t.Fatalf("NewIntervalEndingAt() error = %v", err)
	}
	if !iv.Start().Equal(ivStart) {
		t.Errorf("Start() = %v, want %v", iv.Start(), ivStart)
	}
	if !iv.End().Equal(ivEnd) {
		t.Errorf("End() = %v, want %v", iv.End(), ivEnd)
	}
}

func TestNewIntervalEndingAt_NegativeLength(t *testing.T) {
	_, err := NewIntervalEndingAt(ivEnd, -1*Hour)
	if !errors.Is(err, ErrInvalidDuration) {
		t.Errorf("NewIntervalEndingAt() error = %v, want ErrInvalidDuration", err)
	}
}

func TestInterval_Length(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)
	if got := mustIntervalLength(t, iv).InHours(); got != 9.0 {
		t.Errorf("Length() = %v hours, want 9.0", got)
	}
}

func TestInterval_LengthRejectsOverflowWithoutInvalidatingInterval(t *testing.T) {
	t.Parallel()

	start := InstantFromTime(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC))
	end := InstantFromTime(time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC))
	iv := mustInterval(t, start, end)

	_, err := iv.Length()
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Length() error = %v, want ErrOverflow", err)
	}
	if _, err := json.Marshal(iv); err != nil {
		t.Fatalf("Marshal(long interval) error = %v", err)
	}
}

func TestInterval_Contains_HalfOpen(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)

	// Start is inclusive
	if !iv.Contains(ivStart) {
		t.Error("Should contain start (inclusive)")
	}
	// End is exclusive
	if iv.Contains(ivEnd) {
		t.Error("Should NOT contain end (half-open: end is exclusive)")
	}
	// Middle
	mid := UnixNanos((4 * Hour).Nanoseconds())
	if !iv.Contains(mid) {
		t.Error("Should contain midpoint")
	}
	// Just before end — still inside
	justBeforeEnd := ivEnd.Add(Duration(-1))
	if !iv.Contains(justBeforeEnd) {
		t.Error("Should contain instant just before end")
	}
	// Before start
	before := UnixNanos(-1)
	if iv.Contains(before) {
		t.Error("Should not contain instant before start")
	}
	// After end
	after := ivEnd.Add((1 * Second))
	if iv.Contains(after) {
		t.Error("Should not contain instant after end")
	}
}

func TestInterval_Overlaps_HalfOpen(t *testing.T) {
	iv1 := mustInterval(t, ivStart, ivEnd)
	// Overlapping
	iv2 := mustInterval(t, ivStart.Add((4 * Hour)), ivEnd.Add((4 * Hour)))
	if !iv1.Overlaps(iv2) {
		t.Error("overlapping intervals should overlap")
	}
	// Touching at endpoint — half-open: [0,9) and [9,10) don't share any moment
	iv3 := mustInterval(t, ivEnd, ivEnd.Add((1 * Hour)))
	if iv1.Overlaps(iv3) {
		t.Error("touching at endpoint should NOT overlap (half-open)")
	}
	// Disjoint
	iv4 := mustInterval(t, ivEnd.Add((1 * Second)), ivEnd.Add((1 * Hour)))
	if iv1.Overlaps(iv4) {
		t.Error("disjoint intervals should not overlap")
	}
}

func TestInterval_Adjacent(t *testing.T) {
	iv1 := mustInterval(t, ivStart, ivEnd)
	// [0,9) and [9,10) are adjacent
	iv2 := mustInterval(t, ivEnd, ivEnd.Add((1 * Hour)))
	if !iv1.Adjacent(iv2) {
		t.Error("[0,9) and [9,10) should be adjacent")
	}
	// Reverse direction
	if !iv2.Adjacent(iv1) {
		t.Error("[9,10) and [0,9) should be adjacent (symmetric)")
	}
	// Overlapping — not adjacent
	iv3 := mustInterval(t, ivStart.Add((4 * Hour)), ivEnd.Add((4 * Hour)))
	if iv1.Adjacent(iv3) {
		t.Error("overlapping intervals should not be adjacent")
	}
	// Gap — not adjacent
	iv4 := mustInterval(t, ivEnd.Add((1 * Hour)), ivEnd.Add((2 * Hour)))
	if iv1.Adjacent(iv4) {
		t.Error("intervals with a gap should not be adjacent")
	}
}

func TestInterval_Intersect(t *testing.T) {
	iv1 := mustInterval(t, ivStart, ivEnd)
	iv2 := mustInterval(t, ivStart.Add((4 * Hour)), ivEnd.Add((4 * Hour)))

	overlap, ok := iv1.Intersect(iv2)
	if !ok {
		t.Fatal("overlapping intervals should intersect")
	}
	if !overlap.Start().Equal(ivStart.Add((4 * Hour))) {
		t.Errorf("Intersect Start = %v, want %v", overlap.Start(), ivStart.Add((4 * Hour)))
	}
	if !overlap.End().Equal(ivEnd) {
		t.Errorf("Intersect End = %v, want %v", overlap.End(), ivEnd)
	}

	// Adjacent half-open intervals share no moment.
	iv3 := mustInterval(t, ivEnd, ivEnd.Add((1 * Hour)))
	_, ok = iv1.Intersect(iv3)
	if ok {
		t.Error("adjacent intervals should not intersect")
	}

	// Disjoint
	iv3 = mustInterval(t, ivEnd.Add((1 * Second)), ivEnd.Add((1 * Hour)))
	_, ok = iv1.Intersect(iv3)
	if ok {
		t.Error("disjoint intervals should not intersect")
	}
}

func TestInterval_Union(t *testing.T) {
	iv1 := mustInterval(t, ivStart, ivEnd)
	iv2 := mustInterval(t, ivStart.Add((4 * Hour)), ivEnd.Add((4 * Hour)))

	u, err := iv1.Union(iv2)
	if err != nil {
		t.Fatalf("overlapping union error: %v", err)
	}
	if !u.Start().Equal(ivStart) {
		t.Errorf("Union Start = %v, want %v", u.Start(), ivStart)
	}
	if !u.End().Equal(ivEnd.Add((4 * Hour))) {
		t.Errorf("Union End = %v, want %v", u.End(), ivEnd.Add((4 * Hour)))
	}

	// Adjacent intervals can be unioned: [0,9) ∪ [9,10)
	iv3 := mustInterval(t, ivEnd, ivEnd.Add((1 * Hour)))
	u2, err := iv1.Union(iv3)
	if err != nil {
		t.Fatalf("adjacent union error: %v", err)
	}
	if !u2.Start().Equal(ivStart) || !u2.End().Equal(ivEnd.Add((1 * Hour))) {
		t.Errorf("Union of adjacent = [%v, %v), want [%v, %v)", u2.Start(), u2.End(), ivStart, ivEnd.Add((1 * Hour)))
	}

	// Disjoint with gap → error
	iv4 := mustInterval(t, ivEnd.Add((1 * Second)), ivEnd.Add((1 * Hour)))
	_, err = iv1.Union(iv4)
	if err == nil {
		t.Error("disjoint intervals with gap should return error on Union")
	}
}

func TestInterval_Shift(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)
	shifted := iv.Shift((1 * Hour))
	if !shifted.Start().Equal(ivStart.Add((1 * Hour))) {
		t.Errorf("Shift Start = %v, want %v", shifted.Start(), ivStart.Add((1 * Hour)))
	}
	if !shifted.End().Equal(ivEnd.Add((1 * Hour))) {
		t.Errorf("Shift End = %v, want %v", shifted.End(), ivEnd.Add((1 * Hour)))
	}
}

func TestInterval_Expand(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)
	expanded, err := iv.Expand((15 * Minute), (15 * Minute))
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}
	if !expanded.Start().Equal(ivStart.Add((-15 * Minute))) {
		t.Errorf("Expand Start = %v, want %v", expanded.Start(), ivStart.Add((-15 * Minute)))
	}
	if !expanded.End().Equal(ivEnd.Add((15 * Minute))) {
		t.Errorf("Expand End = %v, want %v", expanded.End(), ivEnd.Add((15 * Minute)))
	}
}

func TestInterval_Expand_ZeroPreservesInterval(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)

	expanded, err := iv.Expand(0, 0)
	if err != nil {
		t.Fatalf("Expand(0, 0) error = %v", err)
	}
	if !expanded.Start().Equal(ivStart) {
		t.Errorf("Expand(0, 0).Start() = %v, want %v", expanded.Start(), ivStart)
	}
	if !expanded.End().Equal(ivEnd) {
		t.Errorf("Expand(0, 0).End() = %v, want %v", expanded.End(), ivEnd)
	}
}

func TestInterval_Expand_RejectsNegativeInputs(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)

	tests := []struct {
		name   string
		before Duration
		after  Duration
	}{
		{name: "negative before", before: -1 * Minute, after: 0},
		{name: "negative after", before: 0, after: -1 * Minute},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := iv.Expand(tc.before, tc.after)
			if !errors.Is(err, ErrInvalidDuration) {
				t.Errorf("Expand(%v, %v) error = %v, want ErrInvalidDuration", tc.before, tc.after, err)
			}
		})
	}
}

func TestInterval_Expand_RejectsShrinkPastZero(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)

	_, err := iv.Expand(-5*Hour, -5*Hour)
	if !errors.Is(err, ErrInvalidDuration) {
		t.Errorf("Expand(-5h, -5h) error = %v, want ErrInvalidDuration", err)
	}
}

func TestInterval_String(t *testing.T) {
	iv := mustInterval(t, ivStart, ivEnd)
	s := iv.String()
	// Should be <start>/<end> format
	if s == "" {
		t.Error("String() should not be empty")
	}
	// Should contain a slash
	found := false
	for _, c := range s {
		if c == '/' {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("String() = %q, expected '/' separator", s)
	}
}

func TestInterval_IsZero(t *testing.T) {
	var iv Interval
	if !iv.IsZero() {
		t.Error("zero Interval should be zero")
	}
}

func TestInterval_StdRange(t *testing.T) {
	iv, err := NewInterval(ivStart, ivEnd)
	if err != nil {
		t.Fatalf("NewInterval error: %v", err)
	}
	start, end := iv.StdRange()
	if !start.Equal(iv.Start().Std()) {
		t.Errorf("StdRange start = %v, want %v", start, iv.Start().Std())
	}
	if !end.Equal(iv.End().Std()) {
		t.Errorf("StdRange end = %v, want %v", end, iv.End().Std())
	}
	if start.Location() != time.UTC || end.Location() != time.UTC {
		t.Errorf("StdRange should be UTC, got start=%v end=%v", start.Location(), end.Location())
	}
}
