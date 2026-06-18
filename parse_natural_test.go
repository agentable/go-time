package gotime

import (
	"errors"
	"sync"
	"testing"
	"time"

	"golang.org/x/text/language"
)

// fixedNow is a stable reference time for natural language tests: 2026-03-30 (Monday) 12:00 UTC
var fixedNow = InstantFromTime(time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC))

func TestParse_Natural_Zh(t *testing.T) {
	tests := []struct {
		input    string
		locale   string
		zone     string
		wantKind Kind
		wantDate [3]int // year, month, day (0 = don't check)
		wantHour int    // -1 = don't check
	}{
		{"今天", "zh-Hans", "Asia/Shanghai", KindDate, [3]int{2026, 3, 30}, -1},
		{"明天", "zh-Hans", "Asia/Shanghai", KindDate, [3]int{2026, 3, 31}, -1},
		{"昨天", "zh-Hans", "Asia/Shanghai", KindDate, [3]int{2026, 3, 29}, -1},
		{"今天下午三点", "zh-Hans", "Asia/Shanghai", KindDateTime, [3]int{2026, 3, 30}, 15},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input,
				WithInputLocale(language.MustParse(tt.locale)),
				WithZone(MustLoadZone(tt.zone)),
				WithReference(fixedNow),
			)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
			}
			if r.Kind != tt.wantKind {
				t.Fatalf("Kind = %v, want %v", r.Kind, tt.wantKind)
			}
			if tt.wantKind == KindDate {
				d, _ := r.Date()
				if tt.wantDate[0] != 0 {
					if d.Year() != tt.wantDate[0] || int(d.Month()) != tt.wantDate[1] || d.Day() != tt.wantDate[2] {
						t.Errorf("date = %v, want %04d-%02d-%02d", d, tt.wantDate[0], tt.wantDate[1], tt.wantDate[2])
					}
				}
			}
			if tt.wantKind == KindDateTime && tt.wantHour >= 0 {
				dt, _ := r.DateTime()
				if dt.Clock().Hour() != tt.wantHour {
					t.Errorf("hour = %d, want %d", dt.Clock().Hour(), tt.wantHour)
				}
			}
		})
	}
}

func TestParse_Natural_En(t *testing.T) {
	tests := []struct {
		input    string
		wantKind Kind
		wantDate [3]int
		wantHour int
	}{
		{"tomorrow", KindDate, [3]int{2026, 3, 31}, -1},
		{"today", KindDate, [3]int{2026, 3, 30}, -1},
		{"tomorrow at 3pm", KindDateTime, [3]int{2026, 3, 31}, 15},
		{"in 2 hours", KindDuration, [3]int{}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input,
				WithInputLocale(language.English),
				WithZone(MustLoadZone("America/New_York")),
				WithReference(fixedNow),
			)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
			}
			if r.Kind != tt.wantKind {
				t.Fatalf("Kind = %v, want %v", r.Kind, tt.wantKind)
			}
			if tt.wantKind == KindDuration {
				d, _ := r.Duration()
				want := 2 * time.Hour
				if d.Std() != want {
					t.Errorf("duration = %v, want %v", d.Std(), want)
				}
			}
		})
	}
}

func TestParse_Natural_RelativeDateRequiresReference(t *testing.T) {
	t.Parallel()

	r := Parse("tomorrow",
		WithInputLocale(language.English),
		WithZone(MustLoadZone("America/New_York")),
	)
	if r.Status != StatusInvalid {
		t.Fatalf("Parse(tomorrow).Status = %s, want %s", r.Status, StatusInvalid)
	}
	if !errors.Is(r.Error, ErrInvalidFormat) {
		t.Fatalf("Parse(tomorrow).Error = %v, want ErrInvalidFormat", r.Error)
	}
}

func TestParse_Natural_DurationDoesNotRequireReference(t *testing.T) {
	t.Parallel()

	r := Parse("in 2 hours", WithInputLocale(language.English))
	if r.Status != StatusResolved {
		t.Fatalf("Parse(in 2 hours).Status = %s (err=%v), want %s", r.Status, r.Error, StatusResolved)
	}
	if r.Kind != KindDuration {
		t.Fatalf("Parse(in 2 hours).Kind = %s, want %s", r.Kind, KindDuration)
	}
	d, ok := r.Duration()
	if !ok {
		t.Fatal("Parse(in 2 hours).Duration() ok=false, want true")
	}
	if d != 2*Hour {
		t.Fatalf("Parse(in 2 hours).Duration() = %s, want 2h0m0s", d)
	}
}

func TestParse_Natural_DateTimeWithoutZoneReturnsLocalDateTime(t *testing.T) {
	r := Parse("tomorrow at 3pm",
		WithInputLocale(language.English),
		WithReference(fixedNow),
	)
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindLocalDateTime {
		t.Fatalf("Kind = %v, want KindLocalDateTime", r.Kind)
	}
	ldt, ok := r.LocalDateTime()
	if !ok {
		t.Fatal("LocalDateTime() ok=false, want true")
	}
	if got := ldt.String(); got != "2026-03-31T15:00:00" {
		t.Errorf("LocalDateTime() = %s, want 2026-03-31T15:00:00", got)
	}
}

func TestParse_Natural_Ja(t *testing.T) {
	r := Parse("明日",
		WithInputLocale(language.Japanese),
		WithZone(MustLoadZone("Asia/Tokyo")),
		WithReference(fixedNow),
	)
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDate {
		t.Fatalf("Kind = %v, want KindDate", r.Kind)
	}
	d, _ := r.Date()
	if d.Year() != 2026 || d.Month() != time.March || d.Day() != 31 {
		t.Errorf("date = %v, want 2026-03-31", d)
	}
}

func TestParse_Natural_Ko(t *testing.T) {
	r := Parse("오늘",
		WithInputLocale(language.Korean),
		WithReference(fixedNow),
	)
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDate {
		t.Fatalf("Kind = %v, want KindDate", r.Kind)
	}
}

func TestParse_Natural_ISO_Priority(t *testing.T) {
	// ISO 8601 must still take priority over natural language
	r := Parse("2026-03-27", WithInputLocale(language.MustParse("zh-Hans")))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDate {
		t.Fatalf("Kind = %v, want KindDate", r.Kind)
	}
	d, _ := r.Date()
	if d.Year() != 2026 || d.Month() != time.March || d.Day() != 27 {
		t.Errorf("date = %v, want 2026-03-27", d)
	}
}

func TestParse_Natural_NoLocale(t *testing.T) {
	// Without locale, natural parsing should be skipped
	r := Parse("今天") // no locale
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid (no locale → no natural parse)", r.Status)
	}
}

func TestParse_Natural_Unrecognised(t *testing.T) {
	r := Parse("hello world", WithInputLocale(language.English))
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error == nil {
		t.Fatal("Error must not be nil when status is Invalid")
	}
	if r.Error.Code != CodeUnparseable {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeUnparseable)
	}
}

func TestParse_Natural_OverflowPreservesSentinel(t *testing.T) {
	t.Parallel()

	r := Parse("in 9223372036854775808 months", WithInputLocale(language.English))
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error == nil {
		t.Fatal("Error must not be nil when status is Invalid")
	}
	if !errors.Is(r.Error, ErrOverflow) {
		t.Fatalf("error = %v, want ErrOverflow", r.Error)
	}
	if r.Error.Code != CodeOverflow {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeOverflow)
	}
	if r.Error.Hint == "" {
		t.Error("error Hint must not be empty")
	}
}

func TestParse_Natural_Duration_En(t *testing.T) {
	r := Parse("in 2 hours",
		WithInputLocale(language.English),
		WithReference(fixedNow),
	)
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDuration {
		t.Fatalf("Kind = %v, want KindDuration", r.Kind)
	}
	d, _ := r.Duration()
	if d.Std() != 2*time.Hour {
		t.Errorf("duration = %v, want 2h", d.Std())
	}
}

func TestParse_Natural_CalendarUnits_En(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  Period
	}{
		{input: "in 2 days", want: Period{Days: 2}},
		{input: "in 1 week", want: Period{Days: 7}},
		{input: "3 days ago", want: Period{Days: -3}},
		{input: "2 weeks ago", want: Period{Days: -14}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input,
				WithInputLocale(language.English),
				WithReference(fixedNow),
			)
			if r.Status != StatusResolved {
				t.Fatalf("Parse(%q).Status = %v (err=%v), want Resolved", tc.input, r.Status, r.Error)
			}
			if r.Kind != KindPeriod {
				t.Fatalf("Parse(%q).Kind = %v, want KindPeriod", tc.input, r.Kind)
			}
			got, ok := r.Period()
			if !ok {
				t.Fatalf("Parse(%q).Period() ok = false, want true", tc.input)
			}
			if got != tc.want {
				t.Errorf("Parse(%q).Period() = %+v, want %+v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParse_Natural_Race(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			Parse("明天", WithInputLocale(language.MustParse("zh-Hans")), WithReference(fixedNow))
		})
	}
	wg.Wait()
}
