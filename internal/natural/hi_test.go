package natural

import (
	"testing"
	"time"
)

var hiBase = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

func TestHindiRelativeDates(t *testing.T) {
	ctx := Context{Locale: "hi", ZoneID: "UTC", RelativeTo: hiBase}
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
		if !r.Time.Equal(wantDate) {
			t.Errorf("Parse(%q) time=%v want %v", tt.input, r.Time, wantDate)
		}
	}
}

func TestHindiFutureDuration(t *testing.T) {
	ctx := Context{Locale: "hi", ZoneID: "UTC", RelativeTo: hiBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"2 घंटे में", 2 * int64(time.Hour)},
		{"2 घंटे बाद", 2 * int64(time.Hour)},
		{"1 मिनट में", int64(time.Minute)},
		{"3 दिन बाद", 3 * 24 * int64(time.Hour)},
		{"1 सप्ताह में", 7 * 24 * int64(time.Hour)},
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

func TestHindiPastDuration(t *testing.T) {
	ctx := Context{Locale: "hi", ZoneID: "UTC", RelativeTo: hiBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"3 दिन पहले", -3 * 24 * int64(time.Hour)},
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

func TestHindiLocalePrefixMatching(t *testing.T) {
	ctx := Context{Locale: "hi-IN", ZoneID: "UTC", RelativeTo: hiBase}
	_, ok := Parse("आज", ctx)
	if !ok {
		t.Error("Parse(आज, locale=hi-IN) not ok — locale prefix matching failed")
	}
}
