package gotime

import (
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
	if i.UnixNano() != 0 {
		t.Errorf("UnixNano() = %d, want 0", i.UnixNano())
	}
	want := time.Unix(0, 0).UTC()
	if !i.Std().Equal(want) {
		t.Errorf("time mismatch: got %v, want %v", i.Std(), want)
	}
}

func TestUnixMillis(t *testing.T) {
	i := UnixMillis(1000)
	if i.UnixNano() != 1_000_000_000 {
		t.Errorf("UnixNano() = %d, want 1_000_000_000", i.UnixNano())
	}
	if i.UnixMilli() != 1000 {
		t.Errorf("UnixMilli() = %d, want 1000", i.UnixMilli())
	}
}

func TestUnixSeconds(t *testing.T) {
	t.Parallel()

	i := UnixSeconds(1234)
	want := time.Unix(1234, 0).UTC()
	if got := i.Std(); !got.Equal(want) {
		t.Errorf("Std() = %v, want %v", got, want)
	}
	if got := i.UnixNano(); got != 1_234_000_000_000 {
		t.Errorf("UnixNano() = %d, want 1_234_000_000_000", got)
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
	d := i2.Sub(i)
	if d.InHours() != 1.0 {
		t.Errorf("Sub after Add((1 * Hour)) = %v hours, want 1.0", d.InHours())
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
	dt := i.In(UTC)
	if !dt.Instant().Equal(i) {
		t.Error("In(UTC).Instant() should round-trip")
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
