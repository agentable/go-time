package natural

import (
	"testing"
	"time"
)

func koCtx(zoneID string) Context {
	loc := locForZone(zoneID)
	ref := time.Date(2026, 3, 30, 12, 0, 0, 0, loc) // Monday
	return Context{
		Locale:     "ko",
		ZoneID:     zoneID,
		RelativeTo: ref,
	}
}

func TestKo_RelativeDate(t *testing.T) {
	ctx := koCtx("Asia/Seoul")
	loc := locForZone("Asia/Seoul")
	ref := time.Date(2026, 3, 30, 12, 0, 0, 0, loc)
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, loc)

	tests := []struct {
		input string
		want  time.Time
	}{
		{"오늘", today},
		{"내일", today.AddDate(0, 0, 1)},
		{"어제", today.AddDate(0, 0, -1)},
		{"모레", today.AddDate(0, 0, 2)},
		{"그제", today.AddDate(0, 0, -2)},
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
			if !r.DateOnly {
				t.Error("DateOnly = false, want true")
			}
			if !r.Time.Equal(tt.want) {
				t.Errorf("Time = %v, want %v", r.Time, tt.want)
			}
		})
	}
}

func TestKo_WeekRelative(t *testing.T) {
	// RelativeTo = 2026-03-30 (Monday)
	ctx := koCtx("Asia/Seoul")
	loc := locForZone("Asia/Seoul")

	tests := []struct {
		input    string
		wantDate time.Time
	}{
		{"다음 주 월요일", time.Date(2026, 4, 6, 0, 0, 0, 0, loc)},
		{"이번 주 금요일", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		{"지난 주 수요일", time.Date(2026, 3, 25, 0, 0, 0, 0, loc)},
		// Without spaces
		{"다음주 월요일", time.Date(2026, 4, 6, 0, 0, 0, 0, loc)},
		{"이번주 금요일", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
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
			if !r.Time.Equal(tt.wantDate) {
				t.Errorf("Time = %v, want %v", r.Time, tt.wantDate)
			}
		})
	}
}

func TestKo_DateTime(t *testing.T) {
	ctx := koCtx("Asia/Seoul")
	loc := locForZone("Asia/Seoul")
	today := time.Date(2026, 3, 30, 0, 0, 0, 0, loc)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		input    string
		wantTime time.Time
	}{
		// 다음 주 금요일 오후 3시 → 2026-04-03 15:00
		{"다음 주 금요일 오후 3시", time.Date(2026, 4, 3, 15, 0, 0, 0, loc)},
		// 내일 오전 9시 → tomorrow 09:00
		{"내일 오전 9시", time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, loc)},
		// 오늘 오후 2시 30분 → today 14:30
		{"오늘 오후 2시 30분", time.Date(today.Year(), today.Month(), today.Day(), 14, 30, 0, 0, loc)},
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
			if !r.Time.Equal(tt.wantTime) {
				t.Errorf("Time = %v, want %v", r.Time, tt.wantTime)
			}
		})
	}
}

func TestKo_Duration(t *testing.T) {
	ctx := koCtx("")
	hour := int64(time.Hour)
	minute := int64(time.Minute)

	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"2시간 후", 2 * hour},
		{"30분 전", -30 * minute},
		{"1시간 후", hour},
		{"3일 후", 3 * int64(24*time.Hour)},
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

func TestKo_Period(t *testing.T) {
	ctx := koCtx("")
	r, ok := Parse("1년 후", ctx)
	if !ok {
		t.Fatal("Parse returned ok=false")
	}
	if r.Kind != KindPeriod {
		t.Fatalf("Kind = %v, want KindPeriod", r.Kind)
	}
	if r.PeriodYears != 1 {
		t.Errorf("PeriodYears = %d, want 1", r.PeriodYears)
	}
}

func TestKo_NonMatch(t *testing.T) {
	ctx := koCtx("")
	tests := []string{"today", "今天", "2026-03-30", ""}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, ok := Parse(input, ctx)
			if ok {
				t.Errorf("Parse(%q) returned ok=true, want false", input)
			}
		})
	}
}

func TestKo_NonKoLocale(t *testing.T) {
	ctx := Context{Locale: "en", ZoneID: "", RelativeTo: time.Now()}
	_, ok := Parse("오늘", ctx)
	if ok {
		t.Error("ko parser should not handle 'en' locale")
	}
}

func TestKo_WeekRelative_AdditionalWeekdays(t *testing.T) {
	t.Parallel()

	ctx := koCtx("Asia/Seoul")
	loc := locForZone("Asia/Seoul")
	tests := []struct {
		input    string
		wantDate time.Time
	}{
		{input: "이번 주 화요일", wantDate: time.Date(2026, 3, 31, 0, 0, 0, 0, loc)},
		{input: "이번 주 목요일", wantDate: time.Date(2026, 4, 2, 0, 0, 0, 0, loc)},
		{input: "이번 주 토요일", wantDate: time.Date(2026, 4, 4, 0, 0, 0, 0, loc)},
		{input: "이번 주 일요일", wantDate: time.Date(2026, 4, 5, 0, 0, 0, 0, loc)},
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
			if !r.Time.Equal(tt.wantDate) {
				t.Errorf("Time = %v, want %v", r.Time, tt.wantDate)
			}
		})
	}
}
