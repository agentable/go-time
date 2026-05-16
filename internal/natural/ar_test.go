package natural

import (
	"testing"
	"time"
)

var arBase = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

func TestArabicRelativeDates(t *testing.T) {
	ctx := Context{Locale: "ar", ZoneID: "UTC", RelativeTo: arBase}
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
		if !r.Time.Equal(wantDate) {
			t.Errorf("Parse(%q) time=%v want %v", tt.input, r.Time, wantDate)
		}
	}
}

func TestArabicNumericFutureDuration(t *testing.T) {
	ctx := Context{Locale: "ar", ZoneID: "UTC", RelativeTo: arBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"بعد 3 ساعات", 3 * int64(time.Hour)},
		{"بعد 1 دقيقة", int64(time.Minute)},
		{"بعد 5 أيام", 5 * 24 * int64(time.Hour)},
		{"بعد 2 أسابيع", 2 * 7 * 24 * int64(time.Hour)},
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

func TestArabicDualDuration(t *testing.T) {
	ctx := Context{Locale: "ar", ZoneID: "UTC", RelativeTo: arBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"بعد ساعتين", 2 * int64(time.Hour)},
		{"بعد يومين", 2 * 24 * int64(time.Hour)},
		{"منذ ساعتين", -2 * int64(time.Hour)},
		{"بعد دقيقتين", 2 * int64(time.Minute)},
		{"بعد أسبوعين", 2 * 7 * 24 * int64(time.Hour)},
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
	ctx := Context{Locale: "ar", ZoneID: "UTC", RelativeTo: arBase}
	r, ok := Parse("بعد شهرين", ctx)
	if !ok {
		t.Fatal("Parse returned ok=false")
	}
	if r.Kind != KindPeriod {
		t.Fatalf("Kind = %v, want KindPeriod", r.Kind)
	}
	if r.PeriodMonths != 2 {
		t.Errorf("PeriodMonths = %d, want 2", r.PeriodMonths)
	}
}

func TestArabicPastDuration(t *testing.T) {
	ctx := Context{Locale: "ar", ZoneID: "UTC", RelativeTo: arBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"منذ 3 أيام", -3 * 24 * int64(time.Hour)},
		{"منذ 1 ساعة", -int64(time.Hour)},
		{"منذ 2 أسابيع", -2 * 7 * 24 * int64(time.Hour)},
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

func TestArabicLocalePrefixMatching(t *testing.T) {
	tests := []struct {
		locale string
		input  string
	}{
		{"ar-SA", "اليوم"},
		{"ar-EG", "غدا"},
		{"ar-MA", "أمس"},
	}
	for _, tt := range tests {
		ctx := Context{Locale: tt.locale, ZoneID: "UTC", RelativeTo: arBase}
		_, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q, locale=%q) not ok — locale prefix matching failed", tt.input, tt.locale)
		}
	}
}
