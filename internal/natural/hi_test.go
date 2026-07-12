package natural

import (
	"testing"
	"time"
)

var hiBase = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

func TestHindiRelativeDates(t *testing.T) {
	ctx := Context{Locale: "hi", RelativeTo: hiBase}
	tests := []struct {
		input    string
		wantDays int
	}{
		{"आज", 0},
		{"कल", 1}, // future bias
		{"परसों", 2},
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

func TestHindiFutureDuration(t *testing.T) {
	ctx := Context{Locale: "hi", RelativeTo: hiBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"2 घंटे में", 2 * int64(time.Hour)},
		{"2 घंटे बाद", 2 * int64(time.Hour)},
		{"1 मिनट में", int64(time.Minute)},
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

func TestHindiFuturePeriod(t *testing.T) {
	ctx := Context{Locale: "hi", RelativeTo: hiBase}
	tests := []struct {
		input    string
		wantDays int32
	}{
		{"3 दिन बाद", 3},
		{"1 सप्ताह में", 7},
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

func TestHindiPastDuration(t *testing.T) {
	ctx := Context{Locale: "hi", RelativeTo: hiBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"1 घंटे पहले", -int64(time.Hour)},
		{"2 मिनट पहले", -2 * int64(time.Minute)},
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

func TestHindiPastPeriod(t *testing.T) {
	ctx := Context{Locale: "hi", RelativeTo: hiBase}
	r, ok := Parse("3 दिन पहले", ctx)
	if !ok {
		t.Fatal("Parse returned ok=false")
	}
	if r.Kind != KindPeriod {
		t.Fatalf("Kind = %v, want KindPeriod", r.Kind)
	}
	if r.PeriodDays != -3 {
		t.Errorf("PeriodDays = %d, want -3", r.PeriodDays)
	}
}

func TestHindiLocalePrefixMatching(t *testing.T) {
	ctx := Context{Locale: "hi-IN", RelativeTo: hiBase}
	_, ok := Parse("आज", ctx)
	if !ok {
		t.Error("Parse(आज, locale=hi-IN) not ok — locale prefix matching failed")
	}
}
