package natural

import (
	"testing"
	"time"
)

func enCtx() Context {
	return Context{
		Locale:     "en",
		RelativeTo: time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC), // Monday
	}
}

func TestEn_RelativeDate(t *testing.T) {
	ctx := enCtx()
	loc := mustLoadLocation(t, "America/New_York")
	ref := time.Date(2026, 3, 30, 12, 0, 0, 0, loc)
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, loc)

	tests := []struct {
		input string
		want  time.Time
	}{
		{"today", today},
		{"tomorrow", today.AddDate(0, 0, 1)},
		{"yesterday", today.AddDate(0, 0, -1)},
		{"day after tomorrow", today.AddDate(0, 0, 2)},
		{"day before yesterday", today.AddDate(0, 0, -2)},
		// Case insensitive
		{"Tomorrow", today.AddDate(0, 0, 1)},
		{"TOMORROW", today.AddDate(0, 0, 1)},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, ok := Parse(tt.input, ctx)
			if !ok {
				t.Fatalf("Parse(%q) returned ok=false", tt.input)
			}
			if r.Kind != KindDate {
				t.Fatalf("Kind = %v, want KindDate", r.Kind)
			}
			if !equalCivil(r, tt.want) {
				t.Errorf("Time = %v, want %v", civilTime(r), tt.want)
			}
		})
	}
}

func TestEn_WeekRelative(t *testing.T) {
	// RelativeTo = 2026-03-30 (Monday)
	ctx := enCtx()
	loc := mustLoadLocation(t, "America/New_York")

	tests := []struct {
		input    string
		wantDate time.Time
	}{
		{"next Friday", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		{"this Friday", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		{"last Wednesday", time.Date(2026, 3, 25, 0, 0, 0, 0, loc)},
		// next Monday when today is Monday → +7 days
		{"next Monday", time.Date(2026, 4, 6, 0, 0, 0, 0, loc)},
		// Case insensitive
		{"Next Friday", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, ok := Parse(tt.input, ctx)
			if !ok {
				t.Fatalf("Parse(%q) returned ok=false", tt.input)
			}
			if r.Kind != KindDate {
				t.Fatalf("Kind = %v, want KindDate", r.Kind)
			}
			if !equalCivil(r, tt.wantDate) {
				t.Errorf("Time = %v, want %v", civilTime(r), tt.wantDate)
			}
		})
	}
}

func TestEn_DateTime(t *testing.T) {
	ctx := enCtx()
	loc := mustLoadLocation(t, "America/New_York")
	today := time.Date(2026, 3, 30, 0, 0, 0, 0, loc)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		input    string
		wantTime time.Time
	}{
		{"tomorrow at 3pm", time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 15, 0, 0, 0, loc)},
		{"today at 9:30am", time.Date(today.Year(), today.Month(), today.Day(), 9, 30, 0, 0, loc)},
		{"tomorrow at 12pm", time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 0, 0, 0, loc)},
		{"today at 12am", time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)},
		{"next Friday at 4:15pm", time.Date(2026, 4, 3, 16, 15, 0, 0, loc)},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, ok := Parse(tt.input, ctx)
			if !ok {
				t.Fatalf("Parse(%q) returned ok=false", tt.input)
			}
			if r.Kind != KindDateTime {
				t.Fatalf("Kind = %v, want KindDateTime", r.Kind)
			}
			if !equalCivil(r, tt.wantTime) {
				t.Errorf("Time = %v, want %v", civilTime(r), tt.wantTime)
			}
		})
	}
}

func TestEn_Duration(t *testing.T) {
	ctx := enCtx()
	hour := int64(time.Hour)
	minute := int64(time.Minute)

	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"in 2 hours", 2 * hour},
		{"in 1 hour", hour},
		{"30 minutes ago", -30 * minute},
		{"in 1 minute", minute},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, ok := Parse(tt.input, ctx)
			if !ok {
				t.Fatalf("Parse(%q) returned ok=false", tt.input)
			}
			if r.Kind != KindDuration {
				t.Fatalf("Kind = %v, want KindDuration", r.Kind)
			}
			if r.DurNanos != tt.wantNanos {
				t.Errorf("DurNanos = %d, want %d", r.DurNanos, tt.wantNanos)
			}
		})
	}
}

func TestEn_Period(t *testing.T) {
	ctx := enCtx()
	tests := []struct {
		input    string
		wantDays int32
	}{
		{"in 3 days", 3},
		{"1 day ago", -1},
		{"in 2 weeks", 14},
		{"3 weeks ago", -21},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, ok := Parse(tt.input, ctx)
			if !ok {
				t.Fatalf("Parse(%q) returned ok=false", tt.input)
			}
			if r.Kind != KindPeriod {
				t.Fatalf("Kind = %v, want KindPeriod", r.Kind)
			}
			if r.PeriodDays != tt.wantDays {
				t.Errorf("PeriodDays = %d, want %d", r.PeriodDays, tt.wantDays)
			}
		})
	}
}

func TestEn_NonMatch(t *testing.T) {
	ctx := enCtx()
	tests := []string{
		"今天",
		"明天",
		"2026-03-30",
		"",
		"hello world",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, ok := Parse(input, ctx)
			if ok {
				t.Errorf("Parse(%q) returned ok=true, want false", input)
			}
		})
	}
}

func TestEn_NonEnLocale(t *testing.T) {
	ctx := Context{Locale: "zh-Hans", RelativeTo: time.Now()}
	_, ok := Parse("today", ctx)
	if ok {
		t.Error("en parser should not handle 'zh-Hans' locale")
	}
}

func TestEn_WeekRelative_AdditionalWeekdays(t *testing.T) {
	t.Parallel()

	ctx := enCtx()
	loc := mustLoadLocation(t, "America/New_York")
	tests := []struct {
		input    string
		wantDate time.Time
	}{
		{input: "this Tuesday", wantDate: time.Date(2026, 3, 31, 0, 0, 0, 0, loc)},
		{input: "this Thursday", wantDate: time.Date(2026, 4, 2, 0, 0, 0, 0, loc)},
		{input: "this Saturday", wantDate: time.Date(2026, 4, 4, 0, 0, 0, 0, loc)},
		{input: "this Sunday", wantDate: time.Date(2026, 4, 5, 0, 0, 0, 0, loc)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			r, ok := Parse(tt.input, ctx)
			if !ok {
				t.Fatalf("Parse(%q) returned ok=false", tt.input)
			}
			if r.Kind != KindDate {
				t.Fatalf("Kind = %v, want KindDate", r.Kind)
			}
			if !equalCivil(r, tt.wantDate) {
				t.Errorf("Time = %v, want %v", civilTime(r), tt.wantDate)
			}
		})
	}
}
