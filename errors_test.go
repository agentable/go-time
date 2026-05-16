package gotime

import (
	"errors"
	"testing"

	"github.com/go-json-experiment/json"
)

func TestTimeError_Error(t *testing.T) {
	e := &TimeError{
		Code:    CodeUnparseable,
		Message: "cannot parse input",
		Input:   "not a time",
		Hint:    "use ISO 8601 format",
	}
	got := e.Error()
	if got == "" {
		t.Fatal("Error() must not return empty string")
	}
	for _, want := range []string{string(CodeUnparseable), "cannot parse input", "not a time"} {
		if !contains(got, want) {
			t.Errorf("Error() = %q, want it to contain %q", got, want)
		}
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
		CodeAmbiguousTime,
		CodeAmbiguousZone,
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

func TestSentinels_NonEmpty(t *testing.T) {
	sentinels := map[string]error{
		"ErrEmptyInput":        ErrEmptyInput,
		"ErrInvalidFormat":     ErrInvalidFormat,
		"ErrInvalidDate":       ErrInvalidDate,
		"ErrInvalidTime":       ErrInvalidTime,
		"ErrInvalidDuration":   ErrInvalidDuration,
		"ErrInvalidPeriod":     ErrInvalidPeriod,
		"ErrInvalidZone":       ErrInvalidZone,
		"ErrAmbiguousDate":     ErrAmbiguousDate,
		"ErrAmbiguousTime":     ErrAmbiguousTime,
		"ErrAmbiguousZone":     ErrAmbiguousZone,
		"ErrNonexistentTime":   ErrNonexistentTime,
		"ErrDuplicateTime":     ErrDuplicateTime,
		"ErrIntervalReversed":  ErrIntervalReversed,
		"ErrIntervalsDisjoint": ErrIntervalsDisjoint,
		"ErrUnparseable":       ErrUnparseable,
		"ErrOverflow":          ErrOverflow,
		"ErrIncompatibleTypes": ErrIncompatibleTypes,
	}
	for name, s := range sentinels {
		if s == nil {
			t.Errorf("%s is nil", name)
			continue
		}
		if s.Error() == "" {
			t.Errorf("%s has empty message", name)
		}
		if codeForSentinel(s) == "" {
			t.Errorf("%s has no code mapping", name)
		}
	}
}

// contains reports whether s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
