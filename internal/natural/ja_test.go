package natural

import (
	"testing"
	"time"
)

func jaCtx(zoneID string) Context {
	loc := locForZone(zoneID)
	ref := time.Date(2026, 3, 30, 12, 0, 0, 0, loc) // Monday
	return Context{
		Locale:     "ja",
		ZoneID:     zoneID,
		RelativeTo: ref,
	}
}

func TestJa_RelativeDate(t *testing.T) {
	ctx := jaCtx("Asia/Tokyo")
	loc := locForZone("Asia/Tokyo")
	ref := time.Date(2026, 3, 30, 12, 0, 0, 0, loc)
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, loc)

	tests := []struct {
		input string
		want  time.Time
	}{
		{"今日", today},
		{"明日", today.AddDate(0, 0, 1)},
		{"昨日", today.AddDate(0, 0, -1)},
		{"明後日", today.AddDate(0, 0, 2)},
		{"一昨日", today.AddDate(0, 0, -2)},
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

func TestJa_WeekRelative(t *testing.T) {
	// RelativeTo = 2026-03-30 (Monday)
	ctx := jaCtx("Asia/Tokyo")
	loc := locForZone("Asia/Tokyo")

	tests := []struct {
		input    string
		wantDate time.Time
	}{
		// 来週月曜日 → 2026-04-06
		{"来週月曜日", time.Date(2026, 4, 6, 0, 0, 0, 0, loc)},
		// 今週金曜日 → 2026-04-03
		{"今週金曜日", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		// 先週水曜日 → 2026-03-25
		{"先週水曜日", time.Date(2026, 3, 25, 0, 0, 0, 0, loc)},
		// Without 日 suffix
		{"来週月曜", time.Date(2026, 4, 6, 0, 0, 0, 0, loc)},
		{"今週金曜", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
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

func TestJa_DateTime(t *testing.T) {
	ctx := jaCtx("Asia/Tokyo")
	loc := locForZone("Asia/Tokyo")
	today := time.Date(2026, 3, 30, 0, 0, 0, 0, loc)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		input    string
		wantTime time.Time
	}{
		// 来週金曜日の午後3時 → 2026-04-03 15:00
		{"来週金曜日の午後3時", time.Date(2026, 4, 3, 15, 0, 0, 0, loc)},
		// 明日の午前10時 → tomorrow 10:00
		{"明日の午前10時", time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, loc)},
		// 今日の午後2時30分 → today 14:30
		{"今日の午後2時30分", time.Date(today.Year(), today.Month(), today.Day(), 14, 30, 0, 0, loc)},
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

func TestJa_Duration(t *testing.T) {
	ctx := jaCtx("")
	hour := int64(time.Hour)
	minute := int64(time.Minute)

	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"2時間後", 2 * hour},
		{"30分前", -30 * minute},
		{"1時間後", hour},
		{"3日後", 3 * int64(24*time.Hour)},
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

func TestJa_Period(t *testing.T) {
	ctx := jaCtx("")
	r, ok := Parse("1年後", ctx)
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

func TestJa_NonMatch(t *testing.T) {
	ctx := jaCtx("")
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

func TestJa_NonJaLocale(t *testing.T) {
	ctx := Context{Locale: "en", ZoneID: "", RelativeTo: time.Now()}
	_, ok := Parse("今日", ctx)
	if ok {
		t.Error("ja parser should not handle 'en' locale")
	}
}

func TestJa_WeekRelative_AdditionalWeekdays(t *testing.T) {
	t.Parallel()

	ctx := jaCtx("Asia/Tokyo")
	loc := locForZone("Asia/Tokyo")
	tests := []struct {
		input    string
		wantDate time.Time
	}{
		{input: "今週火曜日", wantDate: time.Date(2026, 3, 31, 0, 0, 0, 0, loc)},
		{input: "今週木曜日", wantDate: time.Date(2026, 4, 2, 0, 0, 0, 0, loc)},
		{input: "今週土曜日", wantDate: time.Date(2026, 4, 4, 0, 0, 0, 0, loc)},
		{input: "今週日曜日", wantDate: time.Date(2026, 4, 5, 0, 0, 0, 0, loc)},
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
