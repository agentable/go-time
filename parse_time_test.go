package gotime

import (
	"errors"
	"testing"
)

func TestParse_Time_24h(t *testing.T) {
	tests := []struct {
		input string
		h, m  int
		s     int
	}{
		{"15:00", 15, 0, 0},
		{"08:30:45", 8, 30, 45},
		{"00:00", 0, 0, 0},
		{"23:59:59", 23, 59, 59},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v, want Resolved", r.Status)
			}
			if r.Kind != KindTime {
				t.Fatalf("kind = %v, want KindTime", r.Kind)
			}
			ti, _ := r.Time()
			if ti.Hour() != tt.h || ti.Minute() != tt.m || ti.Second() != tt.s {
				t.Errorf("time = %v, want %02d:%02d:%02d", ti, tt.h, tt.m, tt.s)
			}
		})
	}
}

func TestParse_Time_12h(t *testing.T) {
	tests := []struct {
		input string
		h, m  int
	}{
		{"3pm", 15, 0},
		{"3:30 PM", 15, 30},
		{"3:30PM", 15, 30},
		{"12am", 0, 0},
		{"12pm", 12, 0},
		{"1AM", 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v, want Resolved", r.Status)
			}
			if r.Kind != KindTime {
				t.Fatalf("kind = %v, want KindTime", r.Kind)
			}
			ti, _ := r.Time()
			if ti.Hour() != tt.h || ti.Minute() != tt.m {
				t.Errorf("time = %v, want %02d:%02d", ti, tt.h, tt.m)
			}
		})
	}
}

func TestParse_Time_Invalid(t *testing.T) {
	r := Parse("25:00")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeInvalidTime {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeInvalidTime)
	}
}

func TestParseTimeNanosecondRoundTrip(t *testing.T) {
	t.Parallel()
	for _, tc := range [][4]int{{0, 0, 0, 0}, {12, 30, 45, 0}, {12, 30, 45, 1}, {12, 30, 45, 120000000}, {23, 59, 59, 999999999}} {
		clock, err := NewTimeNanos(tc[0], tc[1], tc[2], tc[3])
		if err != nil {
			t.Fatal(err)
		}
		got, err := ParseTime(clock.String())
		if err != nil || got != clock {
			t.Errorf("%s: %v %v", clock, got, err)
		}
	}
	r := Parse("12:30:45.1234567891")
	clock, ok := r.Time()
	if !ok || clock.Nanosecond() != 123456789 || !hasWarning(r.Warnings, WarnTruncatedPrecision) {
		t.Errorf("truncation: %v %v", clock, r.Error)
	}
	for _, input := range []string{"12:30.5", "12:30:45.1pm", "12:30:45."} {
		if Parse(input).Status != StatusInvalid {
			t.Errorf("accepted %s", input)
		}
	}
	for _, input := range []string{"25:00:00.1", "12:60:00.1", "12:30:60.1"} {
		if _, err := ParseTime(input); !errors.Is(err, ErrInvalidTime) {
			t.Errorf("%s: %v", input, err)
		}
	}
}
