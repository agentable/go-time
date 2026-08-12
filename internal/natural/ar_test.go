package natural

import (
	"testing"
	"time"
)

var arBase = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

func TestArabicRelativeDates(t *testing.T) {
	ctx := Context{Locale: "ar", RelativeTo: arBase}
	tests := []struct {
		input    string
		wantDays int
	}{
		{"اليوم", 0},
		{"غداً", 1},
		{"غدا", 1}, // without tashkeel
		{"أمس", -1},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDate {
			t.Errorf("Parse(%q) kind=%v want KindDate", tt.input, r.Kind)
			continue
		}
		wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
		if !equalCivil(r, wantDate) {
			t.Errorf("Parse(%q) time=%v want %v", tt.input, civilTime(r), wantDate)
		}
	}
}

func TestArabicNumericFutureDuration(t *testing.T) {
	ctx := Context{Locale: "ar", RelativeTo: arBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"بعد 3 ساعات", 3 * int64(time.Hour)},
		{"بعد 1 دقيقة", int64(time.Minute)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDuration {
			t.Errorf("Parse(%q) kind=%v want KindDuration", tt.input, r.Kind)
			continue
		}
		if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

func TestArabicNumericFuturePeriod(t *testing.T) {
	ctx := Context{Locale: "ar", RelativeTo: arBase}
	tests := []struct {
		input    string
		wantDays int32
	}{
		{"بعد 5 أيام", 5},
		{"بعد 2 أسابيع", 14},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindPeriod {
			t.Errorf("Parse(%q) kind=%v want KindPeriod", tt.input, r.Kind)
			continue
		}
		if r.PeriodDays != tt.wantDays {
			t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
		}
	}
}

func TestArabicDualDuration(t *testing.T) {
	ctx := Context{Locale: "ar", RelativeTo: arBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"بعد ساعتين", 2 * int64(time.Hour)},
		{"منذ ساعتين", -2 * int64(time.Hour)},
		{"بعد دقيقتين", 2 * int64(time.Minute)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDuration {
			t.Errorf("Parse(%q) kind=%v want KindDuration", tt.input, r.Kind)
			continue
		}
		if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

func TestArabicDualPeriod(t *testing.T) {
	ctx := Context{Locale: "ar", RelativeTo: arBase}
	tests := []struct {
		input      string
		wantDays   int32
		wantMonths int32
	}{
		{input: "بعد يومين", wantDays: 2},
		{input: "بعد أسبوعين", wantDays: 14},
		{input: "بعد شهرين", wantMonths: 2},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindPeriod {
			t.Errorf("Parse(%q) kind=%v want KindPeriod", tt.input, r.Kind)
			continue
		}
		if r.PeriodDays != tt.wantDays {
			t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
		}
		if r.PeriodMonths != tt.wantMonths {
			t.Errorf("Parse(%q) months=%d want %d", tt.input, r.PeriodMonths, tt.wantMonths)
		}
	}
}

func TestArabicPastDuration(t *testing.T) {
	ctx := Context{Locale: "ar", RelativeTo: arBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"منذ 1 ساعة", -int64(time.Hour)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDuration {
			t.Errorf("Parse(%q) kind=%v want KindDuration", tt.input, r.Kind)
			continue
		}
		if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

func TestArabicPastPeriod(t *testing.T) {
	ctx := Context{Locale: "ar", RelativeTo: arBase}
	tests := []struct {
		input    string
		wantDays int32
	}{
		{"منذ 3 أيام", -3},
		{"منذ 2 أسابيع", -14},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindPeriod {
			t.Errorf("Parse(%q) kind=%v want KindPeriod", tt.input, r.Kind)
			continue
		}
		if r.PeriodDays != tt.wantDays {
			t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
		}
	}
}

func TestArabicLocalePrefixMatching(t *testing.T) {
	tests := []struct {
		locale string
		input  string
		wantOK bool
	}{
		{"ar-SA", "اليوم", true},
		{"ar-EG", "غدا", true},
		{"ar-MA", "أمس", true},
		{"ar-SA", "ليس وقتا", false},
	}
	for _, tt := range tests {
		ctx := Context{Locale: tt.locale, RelativeTo: arBase}
		_, ok := Parse(tt.input, ctx)
		if ok != tt.wantOK {
			t.Errorf("Parse(%q, locale=%q) ok = %v, want %v", tt.input, tt.locale, ok, tt.wantOK)
		}
	}
}
