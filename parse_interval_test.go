package gotime

import (
	"encoding/json/v2"
	"errors"
	"testing"

	"golang.org/x/text/language"
)

func TestParseIntervalRejectsAmbiguousDates(t *testing.T) {
	t.Parallel()
	for _, locale := range []string{"", "en-CA", "en-US", "en-GB"} {
		for _, input := range []string{"PT1H/04/05/2026", "2026-01-01T00:00:00Z/04/05/2026"} {
			opts := []Option{WithInputLocale(language.Make(locale))}
			r := Parse(input, opts...)
			if _, ok := r.Interval(); ok || r.Status != StatusInvalid || !errors.Is(r.Error, ErrIncompatibleTypes) {
				t.Errorf("%s locale=%q: status=%s error=%v", input, locale, r.Status, r.Error)
			}
			iv, err := ParseInterval(input, opts...)
			var detail *TimeError
			if !iv.IsZero() || !errors.Is(err, ErrIncompatibleTypes) || !errors.As(err, &detail) || detail.Input != input || detail.Hint == "" {
				t.Errorf("%s locale=%q: interval=%v error=%v", input, locale, iv, err)
			}
		}
	}
	// A valid first candidate must not hide an incompatible later candidate.
	r := Parse("2026-11-01T01:30:00", WithZone(MustLoadZone("America/New_York")))
	r.Candidates = append(r.Candidates, Parse("2026-11-01"))
	if got, _ := validateIntervalPart("interval", "end", r); !errors.Is(got.Error, ErrIncompatibleTypes) {
		t.Errorf("mixed endpoint candidates accepted: %#v", got)
	}
}

func TestParse_Interval_InvalidEndpointPreservesSemanticError(t *testing.T) {
	t.Parallel()

	const valid = "2026-03-27T00:00:00"
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantCode ErrorCode
	}{
		{name: "invalid start date", input: "2026-02-30T00:00:00/" + valid, wantErr: ErrInvalidDate, wantCode: CodeInvalidDate},
		{name: "invalid end date", input: valid + "/2026-02-30T00:00:00", wantErr: ErrInvalidDate, wantCode: CodeInvalidDate},
		{name: "invalid start time", input: "2026-03-27T25:00:00/" + valid, wantErr: ErrInvalidTime, wantCode: CodeInvalidTime},
		{name: "invalid end time", input: valid + "/2026-03-28T25:00:00", wantErr: ErrInvalidTime, wantCode: CodeInvalidTime},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input, WithZone(UTC))
			if r.Status != StatusInvalid {
				t.Fatalf("Parse(%q).Status = %s, want %s", tc.input, r.Status, StatusInvalid)
			}
			if !errors.Is(r.Error, tc.wantErr) {
				t.Fatalf("Parse(%q).Error = %v, want %v", tc.input, r.Error, tc.wantErr)
			}
			if r.Error.Code != tc.wantCode {
				t.Errorf("Parse(%q).Error.Code = %q, want %q", tc.input, r.Error.Code, tc.wantCode)
			}
			if r.Error.Input != tc.input {
				t.Errorf("Parse(%q).Error.Input = %q, want complete interval", tc.input, r.Error.Input)
			}
			if r.Error.Hint == "" {
				t.Errorf("Parse(%q).Error.Hint is empty", tc.input)
			}
		})
	}
}

func TestParse_Interval_InvalidScenarios(t *testing.T) {
	t.Parallel()

	zone := MustLoadZone("America/New_York")
	tests := []struct {
		name     string
		input    string
		opts     []Option
		wantErr  error
		wantCode ErrorCode
	}{
		{
			name:     "nonexistent start",
			input:    "2026-03-08T02:30:00/2026-03-08T04:00:00",
			opts:     []Option{WithZone(zone)},
			wantErr:  ErrNonexistentTime,
			wantCode: CodeNonexistentTime,
		},
		{
			name:     "nonexistent end",
			input:    "2026-03-08T00:30:00/2026-03-08T02:30:00",
			opts:     []Option{WithZone(zone)},
			wantErr:  ErrNonexistentTime,
			wantCode: CodeNonexistentTime,
		},
		{name: "malformed start", input: "not-a-date/2026-03-27T00:00:00Z", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "malformed end", input: "2026-03-27T00:00:00Z/not-a-date", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "period is not exact duration", input: "2026-03-27T00:00:00Z/P1D", wantErr: ErrIncompatibleTypes, wantCode: CodeIncompatibleTypes},
		{name: "two durations", input: "PT1H/PT2H", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "duration overflow", input: "2026-03-27T00:00:00Z/PT999999999999999999999H", wantErr: ErrOverflow, wantCode: CodeOverflow},
		{name: "malformed start before duration", input: "not-a-date/PT1H", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
		{name: "overflow duration before end", input: "PT999999999999999999999H/2026-03-27T00:00:00Z", wantErr: ErrOverflow, wantCode: CodeOverflow},
		{name: "malformed end after duration", input: "PT1H/not-a-date", wantErr: ErrInvalidFormat, wantCode: CodeInvalidFormat},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input, tc.opts...)
			if r.Status != StatusInvalid || r.Error == nil {
				t.Fatalf("Parse(%q) = %#v, want invalid result", tc.input, r)
			}
			if !errors.Is(r.Error, tc.wantErr) {
				t.Fatalf("Parse(%q).Error = %v, want %v", tc.input, r.Error, tc.wantErr)
			}
			if r.Error.Code != tc.wantCode {
				t.Errorf("Parse(%q).Error.Code = %q, want %q", tc.input, r.Error.Code, tc.wantCode)
			}
			if r.Error.Input != tc.input || r.Error.Hint == "" {
				t.Errorf("Parse(%q) error input/hint = %q/%q", tc.input, r.Error.Input, r.Error.Hint)
			}
		})
	}
}

func TestParse_Interval_DateToDate(t *testing.T) {
	r := Parse("2026-03-27/2026-03-28")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeIncompatibleTypes {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeIncompatibleTypes)
	}
}

func TestParse_Interval_DatetimeToDatetime(t *testing.T) {
	r := Parse("2026-03-27T00:00:00Z/2026-03-28T00:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindInterval {
		t.Fatalf("kind = %v, want KindInterval", r.Kind)
	}
}

func TestParse_Interval_StartDuration(t *testing.T) {
	r := Parse("2026-03-27T00:00:00Z/PT12H")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindInterval {
		t.Fatalf("kind = %v, want KindInterval", r.Kind)
	}
	iv, _ := r.Interval()
	length, err := iv.Length()
	if err != nil {
		t.Fatalf("Length() error = %v", err)
	}
	if length.InHours() != 12 {
		t.Errorf("length = %v hours, want 12", length.InHours())
	}
}

func TestParse_Interval_DurationEnd(t *testing.T) {
	r := Parse("PT12H/2026-03-28T00:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	iv, _ := r.Interval()
	length, err := iv.Length()
	if err != nil {
		t.Fatalf("Length() error = %v", err)
	}
	if length.InHours() != 12 {
		t.Errorf("length = %v hours, want 12", length.InHours())
	}
}

func TestParse_Interval_EndBeforeStart(t *testing.T) {
	const input = "2026-03-28T00:00:00Z/2026-03-27T00:00:00Z"
	r := Parse(input)
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error == nil {
		t.Fatal("Error = nil, want ErrIntervalReversed details")
	}
	if !errors.Is(r.Error, ErrIntervalReversed) {
		t.Errorf("Error = %v, want ErrIntervalReversed", r.Error)
	}
	if r.Error.Code != CodeIntervalReversed {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeIntervalReversed)
	}
	if r.Error.Message != "interval end is before start" {
		t.Errorf("Error.Message = %q, want %q", r.Error.Message, "interval end is before start")
	}
	if r.Error.Input != input {
		t.Errorf("Error.Input = %q, want %q", r.Error.Input, input)
	}
	if r.Error.Hint != "Ensure the interval end is at or after the start" {
		t.Errorf("Error.Hint = %q, want %q", r.Error.Hint, "Ensure the interval end is at or after the start")
	}
}

func TestParse_Interval_AmbiguousEndpoint(t *testing.T) {
	r := Parse(
		"2026-11-01T01:30:00/2026-11-01T03:00:00",
		WithZone(MustLoadZone("America/New_York")),
	)
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v (err=%v), want Ambiguous", r.Status, r.Error)
	}
}

func TestParse_Interval_RejectsNaturalLanguageEndpoint(t *testing.T) {
	r := Parse(
		"tomorrow/2026-03-28T00:00:00Z",
		WithInputLocale(language.English),
		WithReference(fixedNow),
	)
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
}

func TestParseIntervalValidatesBothSides(t *testing.T) {
	t.Parallel()
	const fold = "2026-11-01T01:30:00"
	zone := MustLoadZone("America/New_York")
	for _, tc := range []struct {
		part string
		want error
	}{
		{"garbage", ErrInvalidFormat}, {"2026-02-30T00:00:00", ErrInvalidDate},
		{"2026-03-08T02:30:00", ErrNonexistentTime}, {"2026-03-27", ErrIncompatibleTypes},
		{"PTbad", ErrInvalidFormat}, {"PT999999999999999999999H", ErrOverflow},
	} {
		for _, input := range []string{fold + "/" + tc.part, tc.part + "/" + fold} {
			r := Parse(input, WithZone(zone))
			if r.Status != StatusInvalid || !errors.Is(r.Error, tc.want) {
				t.Errorf("%s: %s %v, want %v", input, r.Status, r.Error, tc.want)
				continue
			}
			_, err := ParseInterval(input, WithZone(zone))
			var detail *TimeError
			if !errors.Is(err, tc.want) || !errors.As(err, &detail) || detail.Input != input || detail.Hint == "" {
				t.Errorf("typed %s: %#v", input, err)
			}
		}
	}
	input := "2026-02-30T00:00:00/garbage"
	if _, err := ParseInterval(input, WithZone(zone)); !errors.Is(err, ErrInvalidDate) {
		t.Errorf("first error = %v", err)
	}
}

func TestParseIntervalCompleteCandidates(t *testing.T) {
	t.Parallel()
	zone := MustLoadZone("America/New_York")
	for _, tc := range []struct {
		left, right string
		want        [][2]string
	}{
		{"01:30:00", "03:00:00", [][2]string{{"05:30:00", "08:00:00"}, {"06:30:00", "08:00:00"}}},
		{"01:30:00", "01:15:00", [][2]string{{"05:30:00", "06:15:00"}}},
		{"01:15:00", "01:30:00", [][2]string{{"05:15:00", "05:30:00"}, {"05:15:00", "06:30:00"}, {"06:15:00", "06:30:00"}}},
		{"01:30:00", "01:30:00", [][2]string{{"05:30:00", "05:30:00"}, {"05:30:00", "06:30:00"}, {"06:30:00", "06:30:00"}}},
		{"01:30:00", "PT1H", [][2]string{{"05:30:00", "06:30:00"}, {"06:30:00", "07:30:00"}}},
		{"PT1H", "01:30:00", [][2]string{{"04:30:00", "05:30:00"}, {"05:30:00", "06:30:00"}}},
		{"01:30:00", "PT0S", [][2]string{{"05:30:00", "05:30:00"}, {"06:30:00", "06:30:00"}}},
		{"03:00:00", "01:30:00", nil},
	} {
		left, right := tc.left, tc.right
		if left[0] != 'P' {
			left = "2026-11-01T" + left
		}
		if right[0] != 'P' {
			right = "2026-11-01T" + right
		}
		input := left + "/" + right
		t.Run(input, func(t *testing.T) {
			r := Parse(input, WithZone(zone))
			if len(tc.want) == 0 {
				if r.Status != StatusInvalid || !errors.Is(r.Error, ErrIntervalReversed) {
					t.Fatalf("reversed result = %#v", r)
				}
				return
			}
			wantStatus := StatusResolved
			if len(tc.want) > 1 {
				wantStatus = StatusAmbiguous
			}
			if r.Status != wantStatus || r.Kind != KindInterval {
				t.Fatalf("result = %s/%s, want %s/interval", r.Status, r.Kind, wantStatus)
			}
			candidates := r.Candidates
			if r.Status == StatusResolved {
				candidates = []ParseResult{r}
			}
			if len(candidates) != len(tc.want) {
				t.Fatalf("candidates = %d, want %d", len(candidates), len(tc.want))
			}
			for n, c := range candidates {
				iv, ok := c.Interval()
				if !ok || c.Input != input {
					t.Fatalf("candidate %d is not complete interval: %#v", n, c)
				}
				if iv.Start().String() != "2026-11-01T"+tc.want[n][0]+"Z" || iv.End().String() != "2026-11-01T"+tc.want[n][1]+"Z" {
					t.Errorf("candidate %d = %v, want %v", n, iv, tc.want[n])
				}
			}
			if _, err := json.Marshal(r); err != nil {
				t.Errorf("diagnostic JSON: %v", err)
			}
			_, err := ParseInterval(input, WithZone(zone))
			if len(tc.want) > 1 {
				if !errors.Is(err, ErrDuplicateTime) {
					t.Errorf("typed ambiguity = %v", err)
				}
			} else if err != nil {
				t.Error(err)
			}
		})
	}
	for _, input := range []string{"2026-11-01T01:30:00/-PT1H", "-PT1H/2026-11-01T01:30:00"} {
		if _, err := ParseInterval(input, WithZone(zone)); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("negative interval length %s: %v", input, err)
		}
	}
}

func TestParseIntervalMetadata(t *testing.T) {
	t.Parallel()
	zone := MustLoadZone("America/New_York")
	for _, tc := range []struct {
		input           string
		zone, precision bool
	}{
		{"2026-01-01T00:00:00Z/2026-01-01T01:00:00Z", true, false},
		{"2026-01-01T00:00:00Z/2026-01-01T01:00:00", true, false},
		{"2026-01-01T00:00:00/2026-01-01T01:00:00", false, false},
		{"2026-01-01T00:00:00.1234567891Z/PT1H", true, true},
		{"PT1H/2026-01-01T01:00:00.1234567891Z", true, true},
		{"2026-11-01T01:30:00.1234567891/2026-11-01T08:00:00Z", true, true},
		{"2026-11-01T01:30:00.1234567891/2026-11-01T03:00:00", false, true},
		{"2026-11-01T01:15:00.1234567891/2026-11-01T01:30:00.1234567891", false, true},
	} {
		input := tc.input
		r := Parse(input, WithZone(zone))
		if r.Status == StatusInvalid {
			t.Fatalf("%s: %v", input, r.Error)
		}
		results := append([]ParseResult{r}, r.Candidates...)
		for _, result := range results {
			if result.Input != input || result.HasZone != tc.zone || hasWarning(result.Warnings, WarnTruncatedPrecision) != tc.precision {
				t.Errorf("%s: metadata zone=%v warnings=%v", input, result.HasZone, result.Warnings)
			}
			count := 0
			for _, w := range result.Warnings {
				if w.Code == WarnTruncatedPrecision {
					count++
				}
			}
			if count > 1 {
				t.Errorf("duplicate precision warning: %v", result.Warnings)
			}
		}
	}
	r := Parse("2026-11-01T01:30:00/2026-11-01T03:00:00", WithZone(zone))
	for _, candidate := range r.Candidates {
		iv, ok := candidate.Interval()
		if !ok {
			t.Fatal("not interval")
		}
		want := iv.Start().Std().In(zone.Location()).Format("MST (-07:00)")
		found := false
		for _, w := range candidate.Warnings {
			if w.Code == WarnDuplicateTime {
				found = true
				if w.Message != want {
					t.Errorf("fold warning=%q, want %q", w.Message, want)
				}
			}
		}
		if !found {
			t.Errorf("missing fold warning for %v", iv)
		}
	}
}
