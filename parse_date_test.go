package gotime

import (
	"testing"
	"time"
)

func TestParse_Date_ISO(t *testing.T) {
	tests := []struct {
		input   string
		wantY   int
		wantM   time.Month
		wantD   int
		wantErr bool
	}{
		{"2026-03-27", 2026, time.March, 27, false},
		{"20260327", 2026, time.March, 27, false},
		{"2026-03", 2026, time.March, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if tt.wantErr {
				if r.Status != StatusInvalid {
					t.Fatalf("status = %v, want Invalid", r.Status)
				}
				return
			}
			if r.Status != StatusResolved {
				t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
			}
			if r.Kind != KindDate {
				t.Fatalf("kind = %v, want KindDate", r.Kind)
			}
			d, _ := r.Date()
			if d.Year() != tt.wantY || d.Month() != tt.wantM || d.Day() != tt.wantD {
				t.Errorf("date = %v, want %04d-%02d-%02d", d, tt.wantY, int(tt.wantM), tt.wantD)
			}
		})
	}
}

func TestParse_Date_Ordinal(t *testing.T) {
	// 2026-079 = day 79 of 2026 = March 20
	r := Parse("2026-079")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDate {
		t.Fatalf("kind = %v, want KindDate", r.Kind)
	}
	d, _ := r.Date()
	if d.Year() != 2026 || d.Month() != time.March || d.Day() != 20 {
		t.Errorf("ordinal date = %v, want 2026-03-20", d)
	}
}

func TestParse_Date_WeekDate(t *testing.T) {
	// 2026-W13-5 = ISO week 13, Friday of 2026 = March 27, 2026
	r := Parse("2026-W13-5")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDate {
		t.Fatalf("kind = %v, want KindDate", r.Kind)
	}
	d, _ := r.Date()
	if d.Year() != 2026 || d.Month() != time.March || d.Day() != 27 {
		t.Errorf("week date = %v, want 2026-03-27", d)
	}
}

func TestParse_Date_Invalid(t *testing.T) {
	tests := []struct {
		input    string
		wantCode ErrorCode
	}{
		{"2026-13-01", CodeInvalidDate}, // month 13
		{"2026-00-01", CodeInvalidDate}, // month 0
		{"2026-03-32", CodeInvalidDate}, // day 32
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error == nil {
				t.Fatal("Error must not be nil when status is Invalid")
			}
			if r.Error.Code != tt.wantCode {
				t.Errorf("error code = %q, want %q", r.Error.Code, tt.wantCode)
			}
			if r.Error.Hint == "" {
				t.Error("error Hint must not be empty")
			}
		})
	}
}

func TestParseDateComponents(t *testing.T) {
	tests := []struct {
		input string
		year  int
		mon   int
		day   int
	}{
		{input: "2026-03-27", year: 2026, mon: 3, day: 27},
		{input: "20260327"},
		{input: "2026-03"},
		{input: "2026-03-27-extra"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			year, mon, day := parseDateComponents(tt.input)
			if year != tt.year || mon != tt.mon || day != tt.day {
				t.Fatalf("parseDateComponents(%q) = (%d, %d, %d), want (%d, %d, %d)",
					tt.input, year, mon, day, tt.year, tt.mon, tt.day)
			}
		})
	}
}

func TestParse_Input_Preserved(t *testing.T) {
	input := "2026-03-27"
	r := Parse(input)
	if r.Input != input {
		t.Errorf("Input = %q, want %q", r.Input, input)
	}
}
