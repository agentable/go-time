package gotime

import "testing"

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
