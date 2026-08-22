package gotime

import (
	"errors"
	"strings"
	"testing"
	"time"

	"encoding/json/v2"
)

func TestTimeError_Error(t *testing.T) {
	e := &TimeError{
		Code:    CodeUnparseable,
		Message: "cannot parse input",
		Input:   "not a time",
		Hint:    "use ISO 8601 format",
	}
	if got, want := e.Error(), "UNPARSEABLE: unparseable"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestTimeError_ErrorKeepsStructuredDetailsOutOfText(t *testing.T) {
	t.Parallel()

	const sensitiveInput = "secret-token\n\x1b[31m"
	err := newTimeError(
		ErrUnparseable,
		"cannot parse "+sensitiveInput,
		sensitiveInput,
		"use an explicit format",
	)

	if got, want := err.Error(), "UNPARSEABLE: unparseable"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, ErrUnparseable) {
		t.Fatal("errors.Is(err, ErrUnparseable) = false, want true")
	}
	if err.Input != sensitiveInput || err.Message != "cannot parse "+sensitiveInput {
		t.Fatalf("structured details = (%q, %q), want original values", err.Input, err.Message)
	}
}

func TestTimeError_ErrorEdges(t *testing.T) {
	t.Parallel()

	var nilErr *TimeError
	if got := nilErr.Error(); got != "<nil>" {
		t.Fatalf("nil Error() = %q, want %q", got, "<nil>")
	}
	if got := nilErr.Unwrap(); got != nil {
		t.Fatalf("nil Unwrap() = %v, want nil", got)
	}

	tests := []struct {
		name string
		err  *TimeError
		want string
	}{
		{
			name: "code only",
			err:  &TimeError{Code: CodeInvalidFormat},
			want: "INVALID_FORMAT: invalid format",
		},
		{
			name: "message without input",
			err:  &TimeError{Code: CodeInvalidFormat, Message: "bad format"},
			want: "INVALID_FORMAT: invalid format",
		},
		{
			name: "unknown code",
			err:  &TimeError{Code: ErrorCode("secret-code"), Message: "secret-message"},
			want: "time error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTimeError_JSON(t *testing.T) {
	e := &TimeError{
		Code:    CodeAmbiguousDate,
		Message: "Ambiguous date",
		Input:   "04/05/2026",
		Hint:    "Use WithInputLocale to disambiguate",
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for key, want := range map[string]string{
		"code":    string(CodeAmbiguousDate),
		"message": "Ambiguous date",
		"input":   "04/05/2026",
		"hint":    "Use WithInputLocale to disambiguate",
	} {
		if got := m[key]; got != want {
			t.Errorf("JSON[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestTimeError_NilJSON(t *testing.T) {
	var e *TimeError
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal(nil): %v", err)
	}
	if string(data) != "null" {
		t.Errorf("json.Marshal(nil *TimeError) = %q, want \"null\"", data)
	}
}

func TestErrorCodes_Unique(t *testing.T) {
	codes := []ErrorCode{
		CodeEmptyInput,
		CodeInvalidFormat,
		CodeInvalidDate,
		CodeInvalidTime,
		CodeInvalidDuration,
		CodeInvalidPeriod,
		CodeInvalidZone,
		CodeAmbiguousDate,
		CodeNonexistentTime,
		CodeDuplicateTime,
		CodeIntervalReversed,
		CodeIntervalsDisjoint,
		CodeUnparseable,
		CodeOverflow,
		CodeIncompatibleTypes,
	}
	seen := make(map[ErrorCode]bool, len(codes))
	for _, c := range codes {
		if c == "" {
			t.Errorf("error code must not be empty")
		}
		if seen[c] {
			t.Errorf("duplicate error code: %q", c)
		}
		seen[c] = true
	}
}

func TestSentinelTextOmitsPackagePrefix(t *testing.T) {
	t.Parallel()

	for sentinel := range codeBySentinel {
		if strings.HasPrefix(sentinel.Error(), "gotime:") {
			t.Errorf("sentinel text %q has redundant package prefix", sentinel)
		}
	}
}

func TestTimeError_UnwrapsSentinel(t *testing.T) {
	// A richly-populated error should match its sentinel via errors.Is.
	err := newTimeError(ErrAmbiguousDate, "ambiguous", "04/05/2026", "use WithInputLocale")
	if !errors.Is(err, ErrAmbiguousDate) {
		t.Errorf("errors.Is(err, ErrAmbiguousDate) = false, want true")
	}
	// Non-matching sentinel must not match.
	if errors.Is(err, ErrInvalidZone) {
		t.Errorf("errors.Is(err, ErrInvalidZone) = true, want false")
	}
}

func TestTimeError_CodeDoesNotDriveMatching(t *testing.T) {
	err := &TimeError{Code: CodeAmbiguousDate}
	if errors.Is(err, ErrAmbiguousDate) {
		t.Errorf("errors.Is(err, ErrAmbiguousDate) = true, want false without Err sentinel")
	}
}

func TestTimeError_As(t *testing.T) {
	err := newTimeError(ErrInvalidZone, "unknown zone", "Mars/Olympus", "use IANA zone ids")
	var te *TimeError
	if !errors.As(err, &te) {
		t.Fatalf("errors.As(err, &te) = false, want true")
	}
	if te.Code != CodeInvalidZone {
		t.Errorf("te.Code = %q, want %q", te.Code, CodeInvalidZone)
	}
	if te.Input != "Mars/Olympus" {
		t.Errorf("te.Input = %q, want %q", te.Input, "Mars/Olympus")
	}
	if te.Hint == "" {
		t.Errorf("te.Hint must not be empty")
	}
}

func TestPublicSentinels_HaveBehaviorProducer(t *testing.T) {
	t.Parallel()

	newYork := MustLoadZone("America/New_York")
	mustInterval := func(start, end int64) Interval {
		t.Helper()
		iv, err := NewInterval(UnixSeconds(start), UnixSeconds(end))
		if err != nil {
			t.Fatalf("NewInterval(%d, %d): %v", start, end, err)
		}
		return iv
	}

	type producer struct {
		name string
		err  func() error
	}
	producers := map[error]producer{
		ErrEmptyInput: {
			name: "empty Parse input",
			err: func() error {
				return Parse("").Error
			},
		},
		ErrInvalidFormat: {
			name: "empty ISO duration",
			err: func() error {
				return Parse("P").Error
			},
		},
		ErrInvalidDate: {
			name: "invalid Date constructor",
			err: func() error {
				_, err := NewDate(2026, time.February, 30)
				return err
			},
		},
		ErrInvalidTime: {
			name: "invalid Time constructor",
			err: func() error {
				_, err := NewTime(24, 0, 0)
				return err
			},
		},
		ErrInvalidDuration: {
			name: "negative interval length",
			err: func() error {
				_, err := NewIntervalStartingAt(UnixSeconds(0), -Second)
				return err
			},
		},
		ErrInvalidPeriod: {
			name: "fractional calendar period",
			err: func() error {
				return Parse("P1.5Y").Error
			},
		},
		ErrInvalidZone: {
			name: "unknown zone",
			err: func() error {
				_, err := LoadZone("Mars/Olympus")
				return err
			},
		},
		ErrAmbiguousDate: {
			name: "ambiguous slash date",
			err: func() error {
				_, err := ParseDate("04/05/2026")
				return err
			},
		},
		ErrNonexistentTime: {
			name: "DST gap datetime",
			err: func() error {
				_, err := ParseDateTime("2026-03-08T02:30:00", WithZone(newYork))
				return err
			},
		},
		ErrDuplicateTime: {
			name: "DST duplicate datetime",
			err: func() error {
				_, err := ParseDateTime("2026-11-01T01:30:00", WithZone(newYork))
				return err
			},
		},
		ErrIntervalReversed: {
			name: "reversed interval constructor",
			err: func() error {
				_, err := NewInterval(UnixSeconds(2), UnixSeconds(1))
				return err
			},
		},
		ErrIntervalsDisjoint: {
			name: "disjoint interval union",
			err: func() error {
				left := mustInterval(0, 1)
				right := mustInterval(2, 3)
				_, err := left.Union(right)
				return err
			},
		},
		ErrUnparseable: {
			name: "unparseable input",
			err: func() error {
				return Parse("not time").Error
			},
		},
		ErrOverflow: {
			name: "overflowing duration",
			err: func() error {
				return Parse("PT999999999999999999999999H").Error
			},
		},
		ErrIncompatibleTypes: {
			name: "typed parser kind mismatch",
			err: func() error {
				_, err := ParseInstant("2026-03-27")
				return err
			},
		},
	}

	for sentinel, code := range codeBySentinel {
		tc, ok := producers[sentinel]
		if !ok {
			t.Fatalf("%v (%s) has no behavior producer in inventory", sentinel, code)
		}
		err := tc.err()
		if err == nil {
			t.Fatalf("%s produced nil error for %v", tc.name, sentinel)
		}
		if !errors.Is(err, sentinel) {
			t.Fatalf("%s error = %v, want errors.Is(..., %v)", tc.name, err, sentinel)
		}
		var te *TimeError
		if !errors.As(err, &te) {
			t.Fatalf("%s error = %T, want *TimeError in chain", tc.name, err)
		}
		if te.Code != code {
			t.Fatalf("%s code = %s, want %s", tc.name, te.Code, code)
		}
		if te.Hint == "" {
			t.Fatalf("%s hint is empty", tc.name)
		}
	}
}
