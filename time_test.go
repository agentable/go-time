package gotime

import (
	"testing"
	gtime "time"
)

func TestNewTime(t *testing.T) {
	ct, err := NewTime(15, 0, 0)
	if err != nil {
		t.Fatalf("NewTime() error: %v", err)
	}
	if ct.Hour() != 15 {
		t.Errorf("Hour() = %d, want 15", ct.Hour())
	}
	if ct.Minute() != 0 {
		t.Errorf("Minute() = %d, want 0", ct.Minute())
	}
	if ct.Second() != 0 {
		t.Errorf("Second() = %d, want 0", ct.Second())
	}
	if ct.Nanosecond() != 0 {
		t.Errorf("Nanosecond() = %d, want 0", ct.Nanosecond())
	}
}

func TestNewTime_Invalid(t *testing.T) {
	if _, err := NewTime(24, 0, 0); err == nil {
		t.Fatal("NewTime(24,0,0) error = nil, want error")
	}
}

func TestNewTimeNanos(t *testing.T) {
	ct, err := NewTimeNanos(13, 30, 0, 500_000_000)
	if err != nil {
		t.Fatalf("NewTimeNanos() error: %v", err)
	}
	if ct.Hour() != 13 {
		t.Errorf("Hour() = %d, want 13", ct.Hour())
	}
	if ct.Minute() != 30 {
		t.Errorf("Minute() = %d, want 30", ct.Minute())
	}
	if ct.Second() != 0 {
		t.Errorf("Second() = %d, want 0", ct.Second())
	}
	if ct.Nanosecond() != 500_000_000 {
		t.Errorf("Nanosecond() = %d, want 500_000_000", ct.Nanosecond())
	}
}

func TestNewTimeNanos_Invalid(t *testing.T) {
	if _, err := NewTimeNanos(13, 30, 0, 1_000_000_000); err == nil {
		t.Fatal("NewTimeNanos(invalid nanosecond) error = nil, want error")
	}
}

func TestTimeFromTime(t *testing.T) {
	ts := gtime.Date(2026, 3, 27, 15, 30, 45, 123456789, gtime.UTC)
	ct := TimeFromTime(ts)
	if ct.Hour() != 15 || ct.Minute() != 30 || ct.Second() != 45 || ct.Nanosecond() != 123456789 {
		t.Errorf("TimeFromTime() = %v, want 15:30:45.123456789", ct)
	}
}

func TestTime_Equal(t *testing.T) {
	t1 := mustTime(15, 0, 0)
	t2 := mustTime(15, 0, 0)
	t3 := mustTime(16, 0, 0)
	if !t1.Equal(t2) {
		t.Error("same times should be equal")
	}
	if t1.Equal(t3) {
		t.Error("different times should not be equal")
	}
}

func TestTime_Before(t *testing.T) {
	t1 := mustTime(9, 0, 0)
	t2 := mustTime(17, 0, 0)
	if !t1.Before(t2) {
		t.Error("9:00 should be Before 17:00")
	}
	if t2.Before(t1) {
		t.Error("17:00 should not be Before 9:00")
	}
}

func TestTime_After(t *testing.T) {
	t1 := mustTime(9, 0, 0)
	t2 := mustTime(17, 0, 0)
	if !t2.After(t1) {
		t.Error("17:00 should be After 9:00")
	}
	if t1.After(t2) {
		t.Error("9:00 should not be After 17:00")
	}
}

func TestTime_IsZero(t *testing.T) {
	var ct Time
	if !ct.IsZero() {
		t.Error("zero Time should be zero")
	}
	if mustTime(0, 0, 1).IsZero() {
		t.Error("non-zero Time should not be zero")
	}
	// Midnight 00:00:00 is the zero value
	if !mustTime(0, 0, 0).IsZero() {
		t.Error("midnight 00:00:00 should be zero")
	}
}

func TestTime_String(t *testing.T) {
	ct := mustTime(15, 0, 0)
	if ct.String() != "15:00:00" {
		t.Errorf("String() = %q, want %q", ct.String(), "15:00:00")
	}
}

func TestTime_StringFractional(t *testing.T) {
	tests := []struct {
		name string
		tm   Time
		want string
	}{
		{name: "tenths trimmed", tm: mustTimeNanos(12, 30, 0, 120_000_000), want: "12:30:00.12"},
		{name: "microsecond precision", tm: mustTimeNanos(12, 30, 0, 1_000), want: "12:30:00.000001"},
		{name: "nanosecond precision", tm: mustTimeNanos(12, 30, 0, 123_456_789), want: "12:30:00.123456789"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tm.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTime_StringLeadingZeros(t *testing.T) {
	ct := mustTime(9, 5, 3)
	if ct.String() != "09:05:03" {
		t.Errorf("String() = %q, want %q", ct.String(), "09:05:03")
	}
}
