package gotime

// This file implements the typed parse helpers documented in
// SPECS/20-parsing.md and SPECS/70-api-surface.md. Each function calls
// Parse and then translates the ParseResult into a Go-idiomatic
// (value, error) pair:
//
//   - StatusResolved + matching Kind → (value, nil)
//   - StatusResolved + wrong Kind    → (zero, *TimeError wrapping ErrIncompatibleTypes)
//   - StatusAmbiguous                → (zero, *TimeError wrapping the matching ErrAmbiguous*)
//   - StatusInvalid                  → (zero, r.Error)
//
// These helpers are sugar over Parse — callers that need Candidates or
// Warnings should call Parse directly and inspect ParseResult.

// ParseInstant parses input and returns an Instant or an error.
func ParseInstant(input string, opts ...Option) (Instant, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindInstant); err != nil {
		return Instant{}, err
	}
	return r.instant, nil
}

// ParseDateTime parses input and returns a DateTime or an error.
func ParseDateTime(input string, opts ...Option) (DateTime, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindDateTime); err != nil {
		return DateTime{}, err
	}
	return r.dateTime, nil
}

// ParseLocalDateTime parses input and returns a LocalDateTime or an error.
func ParseLocalDateTime(input string, opts ...Option) (LocalDateTime, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindLocalDateTime); err != nil {
		return LocalDateTime{}, err
	}
	return r.localDateTime, nil
}

// ParseDate parses input and returns a Date or an error.
func ParseDate(input string, opts ...Option) (Date, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindDate); err != nil {
		return Date{}, err
	}
	return r.date, nil
}

// ParseTime parses input and returns a Time or an error.
func ParseTime(input string, opts ...Option) (Time, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindTime); err != nil {
		return Time{}, err
	}
	return r.timeVal, nil
}

// ParseDuration parses input and returns a Duration or an error.
// Rejects ISO 8601 inputs with calendar components (P1Y, P5D) — use
// ParsePeriod for those.
func ParseDuration(input string, opts ...Option) (Duration, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindDuration); err != nil {
		return 0, err
	}
	return r.duration, nil
}

// ParsePeriod parses input and returns a Period or an error.
// Rejects ISO 8601 inputs with clock components (PT1H) — use ParseDuration
// for those.
func ParsePeriod(input string, opts ...Option) (Period, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindPeriod); err != nil {
		return Period{}, err
	}
	return r.period, nil
}

// ParseInterval parses input and returns an Interval or an error.
func ParseInterval(input string, opts ...Option) (Interval, error) {
	r := Parse(input, opts...)
	if err := parseResultError(r, KindInterval); err != nil {
		return Interval{}, err
	}
	return r.interval, nil
}

// parseResultError maps a ParseResult to a typed (value, error) error half.
// Returns nil if r is Resolved with the requested Kind; otherwise returns a
// *TimeError appropriate to the failure mode.
func parseResultError(r ParseResult, want Kind) error {
	switch r.Status {
	case StatusResolved:
		if r.Kind == want {
			return nil
		}
		return newTimeError(
			ErrIncompatibleTypes,
			"parsed value has a different kind than the typed parser expected",
			r.Input,
			"input parsed as "+string(r.Kind)+", call "+parseFunctionForKind(r.Kind)+" instead",
		)
	case StatusAmbiguous:
		return newTimeError(
			ambiguousSentinelForKind(r.Kind),
			"input is ambiguous; inspect Parse(...).Candidates to choose",
			r.Input,
			"add WithInputLocale or call Parse to inspect candidates",
		)
	case StatusInvalid:
		if r.Error != nil {
			return r.Error
		}
		return newTimeError(ErrUnparseable, "parse failed", r.Input, "")
	default:
		return newTimeError(ErrUnparseable, "parse returned unknown status", r.Input, "")
	}
}

func ambiguousSentinelForKind(k Kind) error {
	switch k {
	case KindDate, KindDateTime, KindLocalDateTime, KindInstant, KindInterval:
		return ErrAmbiguousDate
	case KindTime:
		return ErrAmbiguousTime
	default:
		return ErrAmbiguousDate
	}
}

func parseFunctionForKind(k Kind) string {
	switch k {
	case KindInstant:
		return "ParseInstant"
	case KindDateTime:
		return "ParseDateTime"
	case KindLocalDateTime:
		return "ParseLocalDateTime"
	case KindDate:
		return "ParseDate"
	case KindTime:
		return "ParseTime"
	case KindDuration:
		return "ParseDuration"
	case KindPeriod:
		return "ParsePeriod"
	case KindInterval:
		return "ParseInterval"
	default:
		return "the matching typed parser"
	}
}
