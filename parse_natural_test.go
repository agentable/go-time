package gotime

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/agentable/go-time/internal/natural"

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

func TestParse_Natural_ExplicitZeroReferenceIsPresent(t *testing.T) {
	t.Parallel()

	r := Parse("tomorrow",
		WithInputLocale(language.English),
		WithZone(Zone{}),
		WithReference(Instant{}),
	)
	if r.Status != StatusResolved || r.Kind != KindDate {
		t.Fatalf("Parse(tomorrow) status/kind = %q/%q (error=%v), want resolved/date", r.Status, r.Kind, r.Error)
	}
	d, ok := r.Date()
	if !ok {
		t.Fatal("Date() ok=false, want true")
	}
	want := mustDate(1, time.January, 2)
	if !d.Equal(want) {
		t.Fatalf("Parse(tomorrow) date = %v, want %v", d, want)
	}
}

func TestParse_Natural_ReferenceUsesExplicitZoneDate(t *testing.T) {
	t.Parallel()

	zone := MustLoadZone("America/New_York")
	reference := InstantFromTime(time.Date(2026, time.March, 31, 0, 30, 0, 0, time.UTC))
	r := Parse("tomorrow",
		WithInputLocale(language.English),
		WithZone(zone),
		WithReference(reference),
	)
	if r.Status != StatusResolved {
		t.Fatalf("Parse(tomorrow).Status = %s (error=%v), want %s", r.Status, r.Error, StatusResolved)
	}
	d, ok := r.Date()
	if !ok {
		t.Fatal("Parse(tomorrow).Date() ok = false, want true")
	}
	want := mustDate(2026, time.March, 31)
	if !d.Equal(want) {
		t.Fatalf("Parse(tomorrow).Date() = %s, want %s", d, want)
	}
}

func TestParse_Natural_RelativeDateRejectsCivilDomainOverflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		reference time.Time
	}{
		{name: "before year zero", input: "yesterday", reference: time.Date(0, time.January, 1, 12, 0, 0, 0, time.UTC)},
		{name: "after year 9999", input: "tomorrow", reference: time.Date(9999, time.December, 31, 12, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := Parse(
				tc.input,
				WithInputLocale(language.English),
				WithReference(InstantFromTime(tc.reference)),
				WithZone(UTC),
			)
			if r.Status != StatusInvalid {
				t.Fatalf("Parse(%q) status = %s, want %s; date=%v", tc.input, r.Status, StatusInvalid, r.date)
			}
			if !errors.Is(r.Error, ErrInvalidDate) {
				t.Fatalf("Parse(%q) error = %v, want ErrInvalidDate", tc.input, r.Error)
			}
		})
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

func TestParse_Natural_ReferenceDependentInputRequiresZone(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"tomorrow", "tomorrow at 3pm"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			r := Parse(input,
				WithInputLocale(language.English),
				WithReference(fixedNow),
			)
			if r.Status != StatusInvalid {
				t.Fatalf("Parse(%q).Status = %s, want %s", input, r.Status, StatusInvalid)
			}
			if !errors.Is(r.Error, ErrInvalidZone) {
				t.Fatalf("Parse(%q).Error = %v, want ErrInvalidZone", input, r.Error)
			}
			if r.Error.Hint == "" {
				t.Fatalf("Parse(%q).Error.Hint is empty", input)
			}
			if _, ok := r.Date(); ok {
				t.Fatalf("Parse(%q).Date() ok = true, want false", input)
			}
			if _, ok := r.DateTime(); ok {
				t.Fatalf("Parse(%q).DateTime() ok = true, want false", input)
			}
			if _, ok := r.LocalDateTime(); ok {
				t.Fatalf("Parse(%q).LocalDateTime() ok = true, want false", input)
			}
		})
	}
}

func TestParse_Natural_DateTimeUsesLocalResolution(t *testing.T) {
	t.Parallel()

	zone := MustLoadZone("America/New_York")
	tests := []struct {
		name           string
		input          string
		reference      time.Time
		wantStatus     Status
		wantErr        error
		wantCandidates int
	}{
		{
			name:       "spring gap",
			input:      "tomorrow at 2:30am",
			reference:  time.Date(2026, time.March, 7, 17, 0, 0, 0, time.UTC),
			wantStatus: StatusInvalid,
			wantErr:    ErrNonexistentTime,
		},
		{
			name:           "fall overlap",
			input:          "tomorrow at 1:30am",
			reference:      time.Date(2026, time.October, 31, 16, 0, 0, 0, time.UTC),
			wantStatus:     StatusAmbiguous,
			wantErr:        ErrDuplicateTime,
			wantCandidates: 2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{
				WithInputLocale(language.English),
				WithZone(zone),
				WithReference(InstantFromTime(tc.reference)),
			}
			r := Parse(tc.input, opts...)
			if r.Status != tc.wantStatus {
				t.Fatalf("Parse(%q).Status = %q, want %q", tc.input, r.Status, tc.wantStatus)
			}
			if len(r.Candidates) != tc.wantCandidates {
				t.Fatalf("len(Parse(%q).Candidates) = %d, want %d", tc.input, len(r.Candidates), tc.wantCandidates)
			}
			for i, candidate := range r.Candidates {
				if _, ok := candidate.DateTime(); !ok {
					t.Fatalf("Parse(%q).Candidates[%d].DateTime() ok = false, want true", tc.input, i)
				}
			}
			if len(r.Candidates) == 2 {
				first, _ := r.Candidates[0].DateTime()
				second, _ := r.Candidates[1].DateTime()
				if !first.Before(second) {
					t.Fatalf("Parse(%q) candidates are not chronological: %v, %v", tc.input, first, second)
				}
			}

			_, err := ParseDateTime(tc.input, opts...)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseDateTime(%q) error = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestParse_Natural_DateTimeMatchesExplicitResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		zoneID        string
		referenceDate Date
		input         string
		targetDate    Date
		targetTime    Time
	}{
		{
			name:          "Lord Howe half-hour gap",
			zoneID:        "Australia/Lord_Howe",
			referenceDate: mustDate(2026, time.October, 3),
			input:         "tomorrow at 2:15am",
			targetDate:    mustDate(2026, time.October, 4),
			targetTime:    mustTime(2, 15, 0),
		},
		{
			name:          "Lord Howe half-hour overlap",
			zoneID:        "Australia/Lord_Howe",
			referenceDate: mustDate(2026, time.April, 4),
			input:         "tomorrow at 1:45am",
			targetDate:    mustDate(2026, time.April, 5),
			targetTime:    mustTime(1, 45, 0),
		},
		{
			name:          "Apia skipped date",
			zoneID:        "Pacific/Apia",
			referenceDate: mustDate(2011, time.December, 29),
			input:         "tomorrow at 12pm",
			targetDate:    mustDate(2011, time.December, 30),
			targetTime:    mustTime(12, 0, 0),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			zone := MustLoadZone(tc.zoneID)
			reference := InstantFromTime(time.Date(
				tc.referenceDate.Year(), tc.referenceDate.Month(), tc.referenceDate.Day(),
				12, 0, 0, 0, zone.Location(),
			))
			want := NewLocalDateTime(tc.targetDate, tc.targetTime).Resolve(zone)
			got := Parse(tc.input,
				WithInputLocale(language.English),
				WithZone(zone),
				WithReference(reference),
			)

			switch want.Status {
			case LocalNonexistent:
				if got.Status != StatusInvalid || !errors.Is(got.Error, ErrNonexistentTime) {
					t.Fatalf("Parse(%q) status/error = %q/%v, want invalid/ErrNonexistentTime", tc.input, got.Status, got.Error)
				}
			case LocalAmbiguous:
				if got.Status != StatusAmbiguous {
					t.Fatalf("Parse(%q).Status = %q, want ambiguous", tc.input, got.Status)
				}
				if len(got.Candidates) != len(want.Candidates) {
					t.Fatalf("len(Parse(%q).Candidates) = %d, want %d", tc.input, len(got.Candidates), len(want.Candidates))
				}
				for i, candidate := range got.Candidates {
					dt, ok := candidate.DateTime()
					if !ok {
						t.Fatalf("Parse(%q).Candidates[%d].DateTime() ok = false", tc.input, i)
					}
					if !dt.Instant().Equal(want.Candidates[i].Instant()) {
						t.Errorf("Parse(%q).Candidates[%d] instant = %s, want %s", tc.input, i, dt.Instant(), want.Candidates[i].Instant())
					}
				}
			default:
				t.Fatalf("explicit resolution status = %q, want nonexistent or ambiguous test fixture", want.Status)
			}
		})
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
		WithZone(MustLoadZone("Asia/Seoul")),
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

func TestParse_Natural_SupportedLanguagesUseExplicitContext(t *testing.T) {
	t.Parallel()

	zone := MustLoadZone("UTC")
	tests := []struct {
		name     string
		locale   language.Tag
		date     string
		duration string
		period   string
	}{
		{name: "Arabic", locale: language.Arabic, date: "اليوم", duration: "بعد 2 ساعات", period: "بعد 2 أيام"},
		{name: "English", locale: language.English, date: "today", duration: "in 2 hours", period: "in 2 days"},
		{name: "Hindi", locale: language.Hindi, date: "आज", duration: "2 घंटे में", period: "3 दिन बाद"},
		{name: "Japanese", locale: language.Japanese, date: "今日", duration: "2時間後", period: "3日後"},
		{name: "Korean", locale: language.Korean, date: "오늘", duration: "2시간 후", period: "3일 후"},
		{name: "French", locale: language.French, date: "aujourd'hui", duration: "dans 2 heures", period: "dans 3 jours"},
		{name: "German", locale: language.German, date: "heute", duration: "in 2 Stunden", period: "vor 3 Tagen"},
		{name: "Spanish", locale: language.Spanish, date: "hoy", duration: "en 2 horas", period: "hace 3 días"},
		{name: "Portuguese", locale: language.Portuguese, date: "hoje", duration: "em 2 horas", period: "há 3 dias"},
		{name: "Russian", locale: language.Russian, date: "сегодня", duration: "через 2 часа", period: "3 дня назад"},
		{name: "Simplified Chinese", locale: language.SimplifiedChinese, date: "今天", duration: "两小时后", period: "3天后"},
		{name: "Traditional Chinese", locale: language.TraditionalChinese, date: "今天", duration: "兩小時後", period: "3天後"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if r := Parse(tc.date); r.Status != StatusInvalid {
				t.Fatalf("Parse(%q) without locale status = %s, want %s", tc.date, r.Status, StatusInvalid)
			}
			if r := Parse(tc.date, WithInputLocale(tc.locale), WithZone(zone)); r.Status != StatusInvalid || !errors.Is(r.Error, ErrInvalidFormat) {
				t.Fatalf("Parse(%q) without reference = %#v, want ErrInvalidFormat", tc.date, r)
			}
			if r := Parse(tc.date, WithInputLocale(tc.locale), WithReference(fixedNow)); r.Status != StatusInvalid || !errors.Is(r.Error, ErrInvalidZone) {
				t.Fatalf("Parse(%q) without zone = %#v, want ErrInvalidZone", tc.date, r)
			}
			if r := Parse(tc.date, WithInputLocale(tc.locale), WithReference(fixedNow), WithZone(zone)); r.Status != StatusResolved || r.Kind != KindDate {
				t.Fatalf("Parse(%q) with explicit context = %#v, want resolved date", tc.date, r)
			}

			if r := Parse(tc.duration); r.Status != StatusInvalid {
				t.Fatalf("Parse(%q) without locale status = %s, want %s", tc.duration, r.Status, StatusInvalid)
			}
			if r := Parse(tc.duration, WithInputLocale(tc.locale)); r.Status != StatusResolved || r.Kind != KindDuration {
				t.Fatalf("Parse(%q) with locale = %#v, want reference-independent duration", tc.duration, r)
			}
			if r := Parse(tc.period); r.Status != StatusInvalid {
				t.Fatalf("Parse(%q) without locale status = %s, want %s", tc.period, r.Status, StatusInvalid)
			}
			if r := Parse(tc.period, WithInputLocale(tc.locale)); r.Status != StatusResolved || r.Kind != KindPeriod {
				t.Fatalf("Parse(%q) with locale = %#v, want reference-independent period", tc.period, r)
			}
		})
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

	for _, input := range []string{
		"in 3000000000 months",
		"in 3000000000 hours",
	} {
		r := Parse(input, WithInputLocale(language.English))
		if r.Status != StatusInvalid {
			t.Fatalf("Parse(%q).Status = %v, want Invalid", input, r.Status)
		}
		if r.Error == nil {
			t.Fatalf("Parse(%q).Error = nil, want error", input)
		}
		if !errors.Is(r.Error, ErrOverflow) {
			t.Fatalf("Parse(%q).Error = %v, want ErrOverflow", input, r.Error)
		}
		if r.Error.Code != CodeOverflow {
			t.Errorf("Parse(%q).Error.Code = %q, want %q", input, r.Error.Code, CodeOverflow)
		}
		if r.Error.Hint == "" {
			t.Errorf("Parse(%q).Error.Hint is empty", input)
		}
	}
}

func TestParse_Natural_ChineseNumericAndOverflow(t *testing.T) {
	t.Parallel()

	resolved := Parse("一百零一分钟后", WithInputLocale(language.SimplifiedChinese))
	if resolved.Status != StatusResolved || resolved.Kind != KindDuration {
		t.Fatalf("Parse(一百零一分钟后) = %#v, want resolved duration", resolved)
	}
	duration, ok := resolved.Duration()
	if !ok || duration != 101*Minute {
		t.Fatalf("Parse(一百零一分钟后).Duration() = %v/%v, want 101 minutes", duration, ok)
	}

	const input = "9223372036854775808分钟后"
	r := Parse(input, WithInputLocale(language.SimplifiedChinese))
	if r.Status != StatusInvalid || !errors.Is(r.Error, ErrOverflow) {
		t.Fatalf("Parse(%q) = %#v, want ErrOverflow", input, r)
	}
	if r.Error.Code != CodeOverflow || r.Error.Hint == "" {
		t.Fatalf("Parse(%q).Error = %#v, want CodeOverflow with hint", input, r.Error)
	}
}

func TestNaturalResultToParseResultRejectsUnknownErrorKind(t *testing.T) {
	t.Parallel()

	r := natural.Result{
		Kind:       natural.KindInvalid,
		ErrorKind:  natural.ErrorKind(255),
		ErrMessage: "unknown internal failure",
		ErrHint:    "do not expose this hint",
	}
	got := naturalResultToParseResult("input", &r, &config{})
	if got.Status != StatusInvalid || got.Error == nil {
		t.Fatalf("result = %#v, want invalid result with error", got)
	}
	if !errors.Is(got.Error, ErrUnparseable) || got.Error.Code != CodeUnparseable {
		t.Fatalf("error = %#v, want ErrUnparseable and CodeUnparseable", got.Error)
	}
	if got.Error.Hint == "" {
		t.Fatal("error Hint is empty")
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
	zone := MustLoadZone("Asia/Shanghai")
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			Parse("明天", WithInputLocale(language.MustParse("zh-Hans")), WithZone(zone), WithReference(fixedNow))
		})
	}
	wg.Wait()
}
