package natural

import (
	"testing"
	"time"
)

// fixedCtx returns a Context with a fixed reference time (2026-03-30 Monday, UTC).
// RelativeTo is in the given zone; pass "" for UTC.
func zhCtx(locale, zoneID string) Context {
	loc := locForZone(zoneID)
	ref := time.Date(2026, 3, 30, 12, 0, 0, 0, loc)
	return Context{
		Locale:     locale,
		ZoneID:     zoneID,
		RelativeTo: ref,
	}
}

func TestZh_RelativeDate(t *testing.T) {
	ctx := zhCtx("zh-Hans", "Asia/Shanghai")
	loc := locForZone("Asia/Shanghai")
	ref := time.Date(2026, 3, 30, 12, 0, 0, 0, loc)
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, loc)

	tests := []struct {
		input string
		want  time.Time
	}{
		{"今天", today},
		{"明天", today.AddDate(0, 0, 1)},
		{"昨天", today.AddDate(0, 0, -1)},
		{"后天", today.AddDate(0, 0, 2)},
		{"後天", today.AddDate(0, 0, 2)},
		{"前天", today.AddDate(0, 0, -2)},
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

func TestZh_WeekRelative(t *testing.T) {
	// RelativeTo = 2026-03-30 (Monday)
	ctx := zhCtx("zh-Hans", "Asia/Shanghai")
	loc := locForZone("Asia/Shanghai")

	tests := []struct {
		input    string
		wantDate time.Time
	}{
		// 下周五 with RelativeTo=Monday 2026-03-30 → 2026-04-03 (Friday)
		{"下周五", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		// 本周五 → 2026-04-03 (this week's Friday)
		{"本周五", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		// 上周三 → 2026-03-25 (last Wednesday)
		{"上周三", time.Date(2026, 3, 25, 0, 0, 0, 0, loc)},
		// Traditional variants
		{"下週五", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		{"本週五", time.Date(2026, 4, 3, 0, 0, 0, 0, loc)},
		{"上週三", time.Date(2026, 3, 25, 0, 0, 0, 0, loc)},
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

func TestZh_MonthRelative(t *testing.T) {
	ctx := zhCtx("zh-Hans", "Asia/Shanghai")
	loc := locForZone("Asia/Shanghai")

	tests := []struct {
		input    string
		wantDate time.Time
	}{
		{"下个月", time.Date(2026, 4, 1, 0, 0, 0, 0, loc)},
		{"下個月", time.Date(2026, 4, 1, 0, 0, 0, 0, loc)},
		{"本月底", time.Date(2026, 3, 31, 0, 0, 0, 0, loc)},
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

func TestZh_DateTime(t *testing.T) {
	ctx := zhCtx("zh-Hans", "Asia/Shanghai")
	loc := locForZone("Asia/Shanghai")
	today := time.Date(2026, 3, 30, 0, 0, 0, 0, loc)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		input    string
		wantTime time.Time
	}{
		// 今天下午三点 → today 15:00
		{"今天下午三点", time.Date(today.Year(), today.Month(), today.Day(), 15, 0, 0, 0, loc)},
		// 明天早上九点半 → tomorrow 09:30
		{"明天早上九点半", time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 30, 0, 0, loc)},
		// 今天上午十点 → today 10:00
		{"今天上午十点", time.Date(today.Year(), today.Month(), today.Day(), 10, 0, 0, 0, loc)},
		// 今天晚上八点 → today 20:00
		{"今天晚上八点", time.Date(today.Year(), today.Month(), today.Day(), 20, 0, 0, 0, loc)},
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

func TestZh_Duration(t *testing.T) {
	ctx := zhCtx("zh-Hans", "")
	hour := int64(time.Hour)
	minute := int64(time.Minute)

	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"两小时后", 2 * hour},
		{"2小时后", 2 * hour},
		{"30分钟前", -30 * minute},
		// Traditional
		{"兩小時後", 2 * hour},
		{"30分鐘前", -30 * minute},
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

func TestZh_Period(t *testing.T) {
	ctx := zhCtx("zh-Hans", "")
	r, ok := Parse("3天后", ctx)
	if !ok {
		t.Fatal("Parse returned ok=false")
	}
	if r.Kind != KindPeriod {
		t.Fatalf("Kind = %v, want KindPeriod", r.Kind)
	}
	if r.PeriodDays != 3 {
		t.Errorf("PeriodDays = %d, want 3", r.PeriodDays)
	}
}

func TestZh_NonMatch(t *testing.T) {
	ctx := zhCtx("zh-Hans", "")
	tests := []string{
		"hello",
		"tomorrow",
		"2026-03-30",
		"",
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

func TestZh_NonZhLocale(t *testing.T) {
	ctx := Context{Locale: "en", ZoneID: "", RelativeTo: time.Now()}
	_, ok := Parse("今天", ctx)
	if ok {
		t.Error("zh parser should not handle 'en' locale")
	}
}
