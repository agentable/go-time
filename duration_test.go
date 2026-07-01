package gotime

import (
	"math"
	"testing"
	"time"
)

func TestDuration_Constants(t *testing.T) {
	if Nanosecond != 1 {
		t.Errorf("Nanosecond = %d, want 1", int64(Nanosecond))
	}
	if Microsecond != 1000*Nanosecond {
		t.Errorf("Microsecond mismatch")
	}
	if Millisecond != 1000*Microsecond {
		t.Errorf("Millisecond mismatch")
	}
	if Second != 1000*Millisecond {
		t.Errorf("Second mismatch")
	}
	if Minute != 60*Second {
		t.Errorf("Minute mismatch")
	}
	if Hour != 60*Minute {
		t.Errorf("Hour mismatch")
	}
}

func TestDuration_ConstArithmetic(t *testing.T) {
	d := 5 * Minute
	if d.Std() != 5*time.Minute {
		t.Errorf("5*Minute = %v, want 5m", d.Std())
	}
	d2 := 2 * Hour
	if d2.InHours() != 2.0 {
		t.Errorf("2*Hour.InHours() = %v, want 2.0", d2.InHours())
	}
}

func TestDuration_UnitConversions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		d            Duration
		milliseconds int64
		seconds      float64
	}{
		{
			name:         "positive fractional second truncates milliseconds",
			d:            1500*Millisecond + 999*Microsecond,
			milliseconds: 1500,
			seconds:      1.500999,
		},
		{
			name:         "negative fractional second truncates milliseconds toward zero",
			d:            -(1500*Millisecond + 500*Microsecond),
			milliseconds: -1500,
			seconds:      -1.5005,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.d.Milliseconds(); got != tc.milliseconds {
				t.Errorf("Milliseconds() = %d, want %d", got, tc.milliseconds)
			}
			if got := tc.d.InSeconds(); math.Abs(got-tc.seconds) > 1e-12 {
				t.Errorf("InSeconds() = %v, want %v", got, tc.seconds)
			}
		})
	}
}

func TestDuration_IsZero(t *testing.T) {
	if !Duration(0).IsZero() {
		t.Error("Duration(0).IsZero() should be true")
	}
	if (5 * Minute).IsZero() {
		t.Error("5*Minute.IsZero() should be false")
	}
}

func TestDuration_IsNegative(t *testing.T) {
	if !(-1 * Minute).IsNegative() {
		t.Error("-1*Minute.IsNegative() should be true")
	}
	if (5 * Minute).IsNegative() {
		t.Error("5*Minute.IsNegative() should be false")
	}
	if Duration(0).IsNegative() {
		t.Error("Duration(0).IsNegative() should be false")
	}
}

func TestDuration_Abs(t *testing.T) {
	d := -5 * Minute
	if d.Abs() != 5*Minute {
		t.Errorf("(-5*Minute).Abs() = %v, want 5m", d.Abs())
	}
	d2 := 5 * Minute
	if d2.Abs() != 5*Minute {
		t.Errorf("(5*Minute).Abs() = %v, want 5m", d2.Abs())
	}
}

func TestDuration_ISO8601(t *testing.T) {
	tests := []struct {
		d    Duration
		want string
	}{
		{Duration(0), "PT0S"},
		{1 * Hour, "PT1H"},
		{90 * Minute, "PT1H30M"},
		{45 * Minute, "PT45M"},
		{30 * Second, "PT30S"},
		{Nanosecond, "PT0.000000001S"},
		{Microsecond, "PT0.000001S"},
		{Millisecond, "PT0.001S"},
		{Second + Nanosecond, "PT1.000000001S"},
		{-30 * Minute, "-PT30M"},
		{-Nanosecond, "-PT0.000000001S"},
		{2*Hour + 15*Minute, "PT2H15M"},
	}
	for _, tc := range tests {
		if got := tc.d.ISO8601(); got != tc.want {
			t.Errorf("(%v).ISO8601() = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestDuration_String(t *testing.T) {
	tests := []struct {
		name string
		d    Duration
		want string
	}{
		{name: "compound minutes", d: 90 * Minute, want: "1h30m0s"},
		{name: "zero", d: Duration(0), want: "0s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.d.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDuration_Decompose(t *testing.T) {
	c := (90*Minute + 5*Second).Decompose()
	if c.Hours != 1 || c.Minutes != 30 || c.Seconds != 5 {
		t.Errorf("Decompose() = %+v, want {Hours:1 Minutes:30 Seconds:5}", c)
	}
}

func TestDuration_Decompose_NegativeIsSigned(t *testing.T) {
	d := -(90 * Minute)
	c := d.Decompose()
	if c.Hours != -1 || c.Minutes != -30 {
		t.Errorf("Decompose() of negative = %+v, want {Hours:-1 Minutes:-30}", c)
	}
	if !d.IsNegative() {
		t.Error("IsNegative should be true")
	}
}

func TestDuration_Decompose_SubSecond(t *testing.T) {
	d := 1*Second + 500*Millisecond + 250*Microsecond + 125*Nanosecond
	c := d.Decompose()
	if c.Seconds != 1 || c.Milliseconds != 500 || c.Microseconds != 250 || c.Nanoseconds != 125 {
		t.Errorf("Decompose() sub-second slots = %+v", c)
	}
}

func TestDuration_Std_RoundTrip(t *testing.T) {
	d := 90 * Minute
	std := d.Std()
	if std != 90*time.Minute {
		t.Errorf("Std() = %v, want 90m", std)
	}
	back := Duration(std)
	if back != d {
		t.Errorf("round trip mismatch: %v vs %v", back, d)
	}
}
