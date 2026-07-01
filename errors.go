package gotime

import (
	"errors"
	"fmt"
)

// ErrorCode is the stable machine-readable JSON/wire category.
type ErrorCode string

// Error code constants — string-typed, prefixed Code*. These are the
// stable codes that appear in JSON, logs, and tooling. Pair each with
// a sentinel error below for use in errors.Is matching.
const (
	// CodeEmptyInput reports that the input string was empty.
	CodeEmptyInput ErrorCode = "EMPTY_INPUT"
	// CodeInvalidFormat reports a syntactically invalid input shape.
	CodeInvalidFormat ErrorCode = "INVALID_FORMAT"
	// CodeInvalidDate reports an invalid calendar date.
	CodeInvalidDate ErrorCode = "INVALID_DATE"
	// CodeInvalidTime reports an invalid clock time.
	CodeInvalidTime ErrorCode = "INVALID_TIME"
	// CodeInvalidDuration reports an invalid duration string.
	CodeInvalidDuration ErrorCode = "INVALID_DURATION"
	// CodeInvalidPeriod reports an invalid period string.
	CodeInvalidPeriod ErrorCode = "INVALID_PERIOD"
	// CodeInvalidZone reports an invalid or unresolvable timezone identifier.
	CodeInvalidZone ErrorCode = "INVALID_ZONE"
	// CodeAmbiguousDate reports multiple plausible date interpretations.
	CodeAmbiguousDate ErrorCode = "AMBIGUOUS_DATE"
	// CodeNonexistentTime reports a local time that falls in a DST gap.
	CodeNonexistentTime ErrorCode = "NONEXISTENT_LOCAL_TIME"
	// CodeDuplicateTime reports a local time that occurs twice during DST fall-back.
	CodeDuplicateTime ErrorCode = "DUPLICATE_LOCAL_TIME"
	// CodeIntervalReversed reports an Interval where end is before start.
	CodeIntervalReversed ErrorCode = "INTERVAL_END_BEFORE_START"
	// CodeIntervalsDisjoint reports an Interval Union where the inputs do not touch.
	CodeIntervalsDisjoint ErrorCode = "INTERVALS_DISJOINT"
	// CodeUnparseable reports input that no parser could interpret.
	CodeUnparseable ErrorCode = "UNPARSEABLE"
	// CodeOverflow reports arithmetic overflow.
	CodeOverflow ErrorCode = "OVERFLOW"
	// CodeIncompatibleTypes reports an operation on incompatible time types.
	CodeIncompatibleTypes ErrorCode = "INCOMPATIBLE_TYPES"
)

// TimeError is the structured error type for every failure in this package.
// It composes with the standard library: errors.Is matches the Err sentinel,
// while errors.As extracts Input, Hint, Message, and Code for inspection.
type TimeError struct {
	// Code is the stable machine-readable error code.
	Code ErrorCode `json:"code"`
	// Message is the human-readable error summary.
	Message string `json:"message,omitzero"`
	// Input is the original input that triggered the error.
	Input string `json:"input,omitzero"`
	// Hint explains how to correct the input.
	Hint string `json:"hint,omitzero"`
	// Err is the sentinel identity for errors.Is.
	Err error `json:"-"`
}

func newTimeError(sentinel error, message, input, hint string) *TimeError {
	return &TimeError{
		Code:    codeForSentinel(sentinel),
		Message: message,
		Input:   input,
		Hint:    hint,
		Err:     sentinel,
	}
}

// Error implements the error interface.
func (e *TimeError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Input == "" {
		if e.Message == "" {
			return string(e.Code)
		}
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s (input: %q)", e.Code, e.Message, e.Input)
}

// Unwrap returns the sentinel identity for errors.Is.
func (e *TimeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Sentinel errors — one per ErrorCode. Use these with errors.Is for
// control-flow matching. Use errors.As(&te) to extract Input/Hint/Message
// from a returned *TimeError.
var (
	ErrEmptyInput        = errors.New("gotime: empty input")
	ErrInvalidFormat     = errors.New("gotime: invalid format")
	ErrInvalidDate       = errors.New("gotime: invalid date")
	ErrInvalidTime       = errors.New("gotime: invalid time")
	ErrInvalidDuration   = errors.New("gotime: invalid duration")
	ErrInvalidPeriod     = errors.New("gotime: invalid period")
	ErrInvalidZone       = errors.New("gotime: invalid zone")
	ErrAmbiguousDate     = errors.New("gotime: ambiguous date")
	ErrNonexistentTime   = errors.New("gotime: nonexistent local time")
	ErrDuplicateTime     = errors.New("gotime: duplicate local time")
	ErrIntervalReversed  = errors.New("gotime: interval end before start")
	ErrIntervalsDisjoint = errors.New("gotime: intervals disjoint")
	ErrUnparseable       = errors.New("gotime: unparseable")
	ErrOverflow          = errors.New("gotime: overflow")
	ErrIncompatibleTypes = errors.New("gotime: incompatible types")
)

// codeBySentinel is the single source of truth pairing sentinel errors with
// their wire-stable codes. sentinelByCode is computed as its inverse so adding
// a new error only requires one entry here.
var codeBySentinel = map[error]ErrorCode{
	ErrEmptyInput:        CodeEmptyInput,
	ErrInvalidFormat:     CodeInvalidFormat,
	ErrInvalidDate:       CodeInvalidDate,
	ErrInvalidTime:       CodeInvalidTime,
	ErrInvalidDuration:   CodeInvalidDuration,
	ErrInvalidPeriod:     CodeInvalidPeriod,
	ErrInvalidZone:       CodeInvalidZone,
	ErrAmbiguousDate:     CodeAmbiguousDate,
	ErrNonexistentTime:   CodeNonexistentTime,
	ErrDuplicateTime:     CodeDuplicateTime,
	ErrIntervalReversed:  CodeIntervalReversed,
	ErrIntervalsDisjoint: CodeIntervalsDisjoint,
	ErrUnparseable:       CodeUnparseable,
	ErrOverflow:          CodeOverflow,
	ErrIncompatibleTypes: CodeIncompatibleTypes,
}

var sentinelByCode = func() map[ErrorCode]error {
	m := make(map[ErrorCode]error, len(codeBySentinel))
	for s, c := range codeBySentinel {
		m[c] = s
	}
	return m
}()

func codeForSentinel(err error) ErrorCode {
	for s, c := range codeBySentinel {
		if errors.Is(err, s) {
			return c
		}
	}
	return ""
}

func sentinelForCode(code ErrorCode) error {
	return sentinelByCode[code]
}
