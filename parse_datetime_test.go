package gotime

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParse_RFC3339_UTC(t *testing.T) {
	r := Parse("2026-03-27T04:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindInstant {
		t.Fatalf("kind = %v, want KindInstant", r.Kind)
	}
	i, _ := r.Instant()
	if i.IsZero() {
		t.Fatal("Instant must not be zero")
	}
	got := i.Std()
	want := time.Date(2026, 3, 27, 4, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Instant = %v, want %v", got, want)
	}
}

func TestParse_DateTime_WithOffsetReturnsInstant(t *testing.T) {
	r := Parse("2026-03-27T13:00:00+09:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindInstant {
		t.Fatalf("kind = %v, want KindInstant", r.Kind)
	}
	if !r.HasZone {
		t.Fatal("HasZone = false, want true")
	}
	i, ok := r.Instant()
	if !ok {
		t.Fatal("Instant() ok=false, want true")
	}
	want := time.Date(2026, time.March, 27, 4, 0, 0, 0, time.UTC)
	if got := i.Std(); !got.Equal(want) {
		t.Errorf("Instant() = %v, want %v", got, want)
	}
}

func TestParse_DateTime_OffsetSubSecondReturnsInstant(t *testing.T) {
	r := Parse("2026-03-27T13:00:00.123456+09:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindInstant {
		t.Fatalf("kind = %v, want KindInstant", r.Kind)
	}
	i, _ := r.Instant()
	if got := i.Std().Nanosecond(); got != 123456000 {
		t.Errorf("nanoseconds = %d, want 123456000", got)
	}
}

func TestParse_DateTime_FractionalNanoseconds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  int
	}{
		{input: "2026-03-27T13:00:00.1", want: 100_000_000},
		{input: "2026-03-27T13:00:00.123456789", want: 123_456_789},
		{input: "20260327T130000.1234567899", want: 123_456_789},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
			}
			var got int
			switch r.Kind {
			case KindDateTime:
				dt, _ := r.DateTime()
				got = dt.Clock().Nanosecond()
			case KindLocalDateTime:
				ldt, _ := r.LocalDateTime()
				got = ldt.Time.Nanosecond()
			case KindInstant:
				i, _ := r.Instant()
				got = i.Std().Nanosecond()
			default:
				t.Fatalf("kind = %v, want KindDateTime, KindLocalDateTime, or KindInstant", r.Kind)
			}
			if got != tc.want {
				t.Fatalf("nanosecond = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestParse_DateTime_CompactOffsetReturnsInstant(t *testing.T) {
	r := Parse("20260327T130000+0900")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindInstant {
		t.Fatalf("kind = %v, want KindInstant", r.Kind)
	}
	i, _ := r.Instant()
	utc := i.Std()
	if utc.Hour() != 4 {
		t.Errorf("UTC hour = %d, want 4", utc.Hour())
	}
}

func TestParse_DateTime_CompactLocal(t *testing.T) {
	r := Parse("20260327T130000")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindLocalDateTime {
		t.Fatalf("kind = %v, want KindLocalDateTime", r.Kind)
	}
	if r.HasZone {
		t.Fatal("HasZone = true, want false")
	}
	ldt, _ := r.LocalDateTime()
	if ldt.Time.Hour() != 13 {
		t.Errorf("hour = %d, want 13", ldt.Time.Hour())
	}
}

func TestParse_DateTime_NoOffset_ReturnsLocalDateTime(t *testing.T) {
	r := Parse("2026-03-27T13:00:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindLocalDateTime {
		t.Fatalf("kind = %v, want KindLocalDateTime", r.Kind)
	}
	ldt, ok := r.LocalDateTime()
	if !ok {
		t.Fatal("LocalDateTime() ok=false, want true")
	}
	if got := ldt.String(); got != "2026-03-27T13:00:00" {
		t.Errorf("LocalDateTime() = %s, want 2026-03-27T13:00:00", got)
	}
	if _, ok := r.DateTime(); ok {
		t.Fatal("DateTime() ok=true, want false")
	}
}

func TestParse_DateTime_NoOffset_WithZoneOption(t *testing.T) {
	r := Parse("2026-03-27T13:00:00", WithZone(MustLoadZone("Asia/Tokyo")))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	dt, _ := r.DateTime()
	if dt.Zone().ID() != "Asia/Tokyo" {
		t.Errorf("zone = %q, want Asia/Tokyo", dt.Zone().ID())
	}
	want := time.Date(2026, time.March, 27, 4, 0, 0, 0, time.UTC)
	if got := dt.Instant().Std(); !got.Equal(want) {
		t.Errorf("instant = %v, want %v", got, want)
	}
	if !hasWarning(r.Warnings, WarnAssumedZone) {
		t.Fatalf("warnings = %v, want WarnAssumedZone", r.Warnings)
	}
}

func TestParse_DateTime_ExplicitZeroZoneMeansUTC(t *testing.T) {
	t.Parallel()

	r := Parse("2026-03-27T13:00:00", WithZone(Zone{}))
	if r.Status != StatusResolved || r.Kind != KindDateTime {
		t.Fatalf("Parse() status/kind = %q/%q, want resolved/datetime", r.Status, r.Kind)
	}
	dt, ok := r.DateTime()
	if !ok {
		t.Fatal("DateTime() ok=false, want true")
	}
	if !dt.Zone().Equal(UTC) {
		t.Fatalf("DateTime zone = %q, want UTC", dt.Zone().ID())
	}
	if !hasWarning(r.Warnings, WarnAssumedZone) {
		t.Fatalf("warnings = %v, want WarnAssumedZone", r.Warnings)
	}
}

func TestParse_DateTime_ExplicitOffsetIsInstantAndIgnoresZoneOption(t *testing.T) {
	t.Parallel()

	r := Parse("2026-03-27T13:00:00+09:00", WithZone(MustLoadZone("America/New_York")))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if !r.HasZone {
		t.Fatal("HasZone = false, want true")
	}
	if !r.Zone.IsZero() {
		t.Fatalf("result Zone = %q, want zero because input carried an offset", r.Zone.ID())
	}
	if hasWarning(r.Warnings, WarnAssumedZone) {
		t.Fatalf("warnings = %v, do not want WarnAssumedZone", r.Warnings)
	}
	i, ok := r.Instant()
	if !ok {
		t.Fatal("Instant() ok=false, want true")
	}
	want := time.Date(2026, time.March, 27, 4, 0, 0, 0, time.UTC)
	if got := i.Std(); !got.Equal(want) {
		t.Fatalf("instant = %v, want %v", got, want)
	}
}

func TestParse_DateTime_TruncatedPrecisionWarning(t *testing.T) {
	r := Parse("2026-03-27T13:00:00.1234567899+09:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if !hasWarning(r.Warnings, WarnTruncatedPrecision) {
		t.Fatalf("warnings = %v, want WarnTruncatedPrecision", r.Warnings)
	}
	i, _ := r.Instant()
	if got := i.Std().Nanosecond(); got != 123456789 {
		t.Errorf("nanosecond = %d, want 123456789", got)
	}
}

func TestParse_DateTime_DuplicateTimeCandidatesWarn(t *testing.T) {
	r := Parse("2026-11-01T01:30:00", WithZone(MustLoadZone("America/New_York")))
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v, want Ambiguous", r.Status)
	}
	if len(r.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(r.Candidates))
	}
	for _, c := range r.Candidates {
		if !hasWarning(c.Warnings, WarnDuplicateTime) {
			t.Fatalf("candidate warnings = %v, want WarnDuplicateTime", c.Warnings)
		}
		if hasWarning(c.Warnings, WarnInferredCalendar) {
			t.Fatalf("candidate warnings = %v, do not want WarnInferredCalendar", c.Warnings)
		}
	}
}

func TestParse_DateTime_DuplicateTime_NonHourDSTTransition(t *testing.T) {
	r := Parse("2026-04-05T01:45:00", WithZone(MustLoadZone("Australia/Lord_Howe")))
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v, want Ambiguous", r.Status)
	}
	if r.Kind != KindDateTime {
		t.Fatalf("kind = %v, want KindDateTime", r.Kind)
	}
	if len(r.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(r.Candidates))
	}
	for _, c := range r.Candidates {
		dt, ok := c.DateTime()
		if !ok {
			t.Fatalf("candidate kind = %v, want DateTime", c.Kind)
		}
		if dt.Clock().Hour() != 1 || dt.Clock().Minute() != 45 {
			t.Fatalf("candidate clock = %v, want 01:45", dt.Clock())
		}
		if !hasWarning(c.Warnings, WarnDuplicateTime) {
			t.Fatalf("candidate warnings = %v, want WarnDuplicateTime", c.Warnings)
		}
	}
}

func TestParse_DateTime_NonexistentLocalTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		zone  Zone
	}{
		{name: "new york spring forward", input: "2026-03-08T02:30:00", zone: MustLoadZone("America/New_York")},
		{name: "lord howe half hour gap", input: "2026-10-04T02:15:00", zone: MustLoadZone("Australia/Lord_Howe")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input, WithZone(tc.zone))
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error == nil {
				t.Fatal("Error is nil, want TimeError")
			}
			if r.Error.Code != CodeNonexistentTime {
				t.Fatalf("error code = %q, want %q", r.Error.Code, CodeNonexistentTime)
			}
			if r.Error.Hint == "" {
				t.Fatal("error Hint is empty")
			}
		})
	}
}

func TestParse_DateTime_InvalidComponents(t *testing.T) {
	tests := []struct {
		input    string
		wantCode ErrorCode
	}{
		{input: "2026-03-27T25:00:00Z", wantCode: CodeInvalidTime},
		{input: "2026-03-27T13:60:00Z", wantCode: CodeInvalidTime},
		{input: "20260230T130000", wantCode: CodeInvalidDate},
		{input: "20260327T250000", wantCode: CodeInvalidTime},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid for %q", r.Status, tt.input)
			}
			if r.Error == nil || r.Error.Code != tt.wantCode {
				t.Fatalf("error = %#v, want code %q", r.Error, tt.wantCode)
			}
		})
	}
}

func hasWarning(warnings []Warning, code WarningCode) bool {
	for _, w := range warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestParse_DateTime_InvalidOffset(t *testing.T) {
	tests := []string{
		"2026-03-27T13:00:00+99:99",
		"20260327T130000+9999",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error.Code != CodeInvalidZone {
				t.Errorf("error code = %q, want %q", r.Error.Code, CodeInvalidZone)
			}
		})
	}
}

func TestParseDateTimeYearDomain(t *testing.T) {
	t.Parallel()
	for _, year := range []int{0, 1, 9999} {
		for _, input := range []string{fmt.Sprintf("%04d-01-01T13:00:00", year), fmt.Sprintf("%04d0101T130000", year)} {
			local, err := ParseLocalDateTime(input)
			if err != nil || local.Date.Year() != year || local.Time.Hour() != 13 {
				t.Errorf("%s: %v %v", input, local, err)
			}
			zoned, err := ParseDateTime(input, WithZone(UTC))
			if err != nil || zoned.Date().Year() != year || zoned.Clock().Hour() != 13 {
				t.Errorf("%s with UTC: %v %v", input, zoned, err)
			}
			r := Parse(input + "Z")
			instant, ok := r.Instant()
			if !ok || instant.Std().Year() != year || instant.Std().Hour() != 13 {
				t.Errorf("%s offset: %#v", input, r)
			}
		}
	}
	for _, input := range []string{"0000-02-30T13:00:00", "00000230T130000"} {
		if _, err := ParseLocalDateTime(input); !errors.Is(err, ErrInvalidDate) {
			t.Errorf("%s: %v", input, err)
		}
	}
}

func TestParseDateTimeReducedPrecision(t *testing.T) {
	t.Parallel()
	for _, offset := range []string{"Z", "+09:00", "+0900"} {
		want, err := ParseInstant("2026-03-27T13:30:00" + offset)
		if err != nil {
			t.Fatal(err)
		}
		for _, input := range []string{"2026-03-27T13:30" + offset, "20260327T1330" + offset} {
			got, err := ParseInstant(input)
			if err != nil || !got.Equal(want) {
				t.Errorf("%s: %v %v, want %v", input, got, err, want)
			}
			iv, err := ParseInterval(input + "/PT1H")
			if err != nil || !iv.Start().Equal(want) {
				t.Errorf("interval %s: %v %v", input, iv, err)
			}
		}
	}
	for _, input := range []string{"20260327T13.5Z", "20260327T1330.5Z", "20260327T13,5", "20260327T1330,5", "2026-03-27T13:30:00+09", "20260327T133000+09"} {
		r := Parse(input)
		if r.Status != StatusInvalid || r.Error == nil || r.Error.Hint == "" {
			t.Errorf("%s: %s, want invalid with hint", input, r.Status)
		}
	}
	for _, tc := range []struct {
		fraction  string
		want      int
		truncated bool
	}{{".1", 100000000, false}, {".123456789", 123456789, false}, {",1234567891", 123456789, true}} {
		for _, base := range []string{"2026-03-27T13:30:45", "20260327T133045"} {
			r := Parse(base + tc.fraction + "Z")
			i, ok := r.Instant()
			if !ok || i.Std().Nanosecond() != tc.want || hasWarning(r.Warnings, WarnTruncatedPrecision) != tc.truncated {
				t.Errorf("%s%s: %v %v", base, tc.fraction, i, r.Error)
			}
		}
	}
}

func TestParseGapHintsUseKnownFacts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ zone, input string }{{"America/New_York", "2026-03-08T02:30:00"}, {"Australia/Lord_Howe", "2026-10-04T02:15:00"}, {"Pacific/Apia", "2011-12-30T12:00:00"}} {
		_, err := ParseDateTime(tc.input, WithZone(MustLoadZone(tc.zone)))
		var detail *TimeError
		if !errors.Is(err, ErrNonexistentTime) || !errors.As(err, &detail) {
			t.Fatalf("%s: %v", tc.zone, err)
		}
		if strings.Contains(detail.Hint, "Clocks skip") || strings.Contains(detail.Hint, "Try ") || !strings.Contains(strings.ToLower(detail.Hint), "offset") {
			t.Errorf("%s: unsupported advice %q", tc.zone, detail.Hint)
		}
	}
}

func TestParseDateTimeComponentErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		local, compact string
		want           error
	}{{"2026-13-01T12:00:00", "20261301T120000", ErrInvalidDate}, {"2026-02-30T12:00:00", "20260230T120000", ErrInvalidDate}, {"2026-01-01T25:00:00", "20260101T250000", ErrInvalidTime}, {"2026-01-01T12:60:00", "20260101T126000", ErrInvalidTime}, {"2026-01-01T12:00:60", "20260101T120060", ErrInvalidTime}} {
		for _, input := range []string{tc.local, tc.compact, tc.local + "Z", tc.compact + "Z"} {
			r := Parse(input)
			if !errors.Is(r.Error, tc.want) {
				t.Errorf("%s: %v want %v", input, r.Error, tc.want)
			}
			if strings.HasSuffix(input, "Z") {
				_, err := ParseInstant(input)
				if !errors.Is(err, tc.want) {
					t.Errorf("typed %s: %v", input, err)
				}
				_, err = ParseInterval(input + "/PT1H")
				var detail *TimeError
				if !errors.Is(err, tc.want) || !errors.As(err, &detail) || detail.Input != input+"/PT1H" || detail.Hint == "" {
					t.Errorf("interval %s: %v", input, err)
				}
			}
		}
	}
	_, err := ParseInstant("2026-02-30T12:00:00Z")
	var cause *time.ParseError
	if !errors.As(err, &cause) {
		t.Errorf("missing stdlib cause: %v", err)
	}
}
