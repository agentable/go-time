package natural

import (
	"testing"
	"time"
)

// zhCtx returns a Context with a fixed civil reference (2026-03-30 Monday).
func zhCtx(locale string) Context {
	return Context{
		Locale:     locale,
		RelativeTo: time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
	}
}

func TestZh_RelativeDate(t *testing.T) {
	ctx := zhCtx("zh-Hans")
	loc := mustLoadLocation(t, "Asia/Shanghai")
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
			if !equalCivil(r, tt.want) {
				t.Errorf("Time = %v, want %v", civilTime(r), tt.want)
			}
		})
	}
}

func TestZh_WeekRelative(t *testing.T) {
	// RelativeTo = 2026-03-30 (Monday)
	ctx := zhCtx("zh-Hans")
	loc := mustLoadLocation(t, "Asia/Shanghai")

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
		{"這週一", time.Date(2026, 3, 30, 0, 0, 0, 0, loc)},
		{"本周二", time.Date(2026, 3, 31, 0, 0, 0, 0, loc)},
		{"本周四", time.Date(2026, 4, 2, 0, 0, 0, 0, loc)},
		{"本周六", time.Date(2026, 4, 4, 0, 0, 0, 0, loc)},
		{"本周日", time.Date(2026, 4, 5, 0, 0, 0, 0, loc)},
		{"本週天", time.Date(2026, 4, 5, 0, 0, 0, 0, loc)},
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

func TestZh_MonthRelative(t *testing.T) {
	ctx := zhCtx("zh-Hans")
	loc := mustLoadLocation(t, "Asia/Shanghai")

	tests := []struct {
		input    string
		wantDate time.Time
	}{
		{"下个月", time.Date(2026, 4, 1, 0, 0, 0, 0, loc)},
		{"下個月", time.Date(2026, 4, 1, 0, 0, 0, 0, loc)},
		{"本月底", time.Date(2026, 3, 31, 0, 0, 0, 0, loc)},
		{"上个月", time.Date(2026, 2, 1, 0, 0, 0, 0, loc)},
		{"上個月", time.Date(2026, 2, 1, 0, 0, 0, 0, loc)},
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

func TestZh_DateTime(t *testing.T) {
	ctx := zhCtx("zh-Hans")
	loc := mustLoadLocation(t, "Asia/Shanghai")
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
		{"今天上午十二点", time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)},
		{"今天下午十二点", time.Date(today.Year(), today.Month(), today.Day(), 12, 0, 0, 0, loc)},
		{"今天下午十一点", time.Date(today.Year(), today.Month(), today.Day(), 23, 0, 0, 0, loc)},
		{"今天13点45分", time.Date(today.Year(), today.Month(), today.Day(), 13, 45, 0, 0, loc)},
		{"今天九点半", time.Date(today.Year(), today.Month(), today.Day(), 9, 30, 0, 0, loc)},
		{"下周五下午三点", time.Date(2026, 4, 3, 15, 0, 0, 0, loc)},
		{"今天晚上十二点", time.Date(today.Year(), today.Month(), today.Day(), 12, 0, 0, 0, loc)},
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

func TestZh_Duration(t *testing.T) {
	ctx := zhCtx("zh-Hans")
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
		{"十一分钟后", 11 * minute},
		{"十二分钟后", 12 * minute},
		{"二十分钟后", 20 * minute},
		{"二十一分钟后", 21 * minute},
		{"四分钟后", 4 * minute},
		{"五分钟后", 5 * minute},
		{"六分钟后", 6 * minute},
		{"七分钟后", 7 * minute},
		{"一百分钟后", 100 * minute},
		{"一百零一分钟后", 101 * minute},
		{"两百零一分钟后", 201 * minute},
		{"兩百零一分鐘後", 201 * minute},
		{"一百二十一分钟后", 121 * minute},
		{"一千零一十分钟后", 1010 * minute},
		{"一千一百一十一分钟后", 1111 * minute},
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

func TestZh_InvalidNumericComposition(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"十十分钟后",
		"一二分钟后",
		"百百天后",
		"一百一分鐘後",
		"一千一分鐘後",
		"零一分钟后",
		"两十分钟后",
		"二十一百分钟后",
		"一千一十分钟后",
		"一百零两分钟后",
		"一百零兩分鐘後",
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if r, ok := Parse(input, zhCtx("zh-Hans")); ok {
				t.Fatalf("Parse(%q) = %#v, true; want no match", input, r)
			}
		})
	}
}

func TestZh_Overflow(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"9223372036854775808分钟后",
		"2562048小时后",
		"2147483648天后",
		"306783379周后",
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			r, ok := Parse(input, zhCtx("zh-Hans"))
			if !ok {
				t.Fatalf("Parse(%q) returned ok=false, want recognized overflow", input)
			}
			if r.Kind != KindInvalid || r.ErrorKind != ErrorOverflow {
				t.Fatalf("Parse(%q) = %#v, want ErrorOverflow", input, r)
			}
			if r.ErrHint == "" {
				t.Fatalf("Parse(%q).ErrHint is empty", input)
			}
		})
	}
}

func TestZh_Period(t *testing.T) {
	ctx := zhCtx("zh-Hans")
	tests := []struct {
		input      string
		wantDays   int32
		wantMonths int32
	}{
		{input: "3天后", wantDays: 3},
		{input: "3个月后", wantMonths: 3},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			r, ok := Parse(tc.input, ctx)
			if !ok {
				t.Fatal("Parse returned ok=false")
			}
			if r.Kind != KindPeriod {
				t.Fatalf("Kind = %v, want KindPeriod", r.Kind)
			}
			if r.PeriodDays != tc.wantDays || r.PeriodMonths != tc.wantMonths {
				t.Errorf("period days/months = %d/%d, want %d/%d", r.PeriodDays, r.PeriodMonths, tc.wantDays, tc.wantMonths)
			}
		})
	}
}

func TestZh_NonMatch(t *testing.T) {
	ctx := zhCtx("zh-Hans")
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
	ctx := Context{Locale: "en", RelativeTo: time.Now()}
	_, ok := Parse("今天", ctx)
	if ok {
		t.Error("zh parser should not handle 'en' locale")
	}
}
