package gotime

import "github.com/go-json-experiment/json"

// Status is the outcome of a Parse call.
type Status string

const (
	// StatusResolved means Parse found exactly one interpretation.
	StatusResolved Status = "resolved"
	// StatusAmbiguous means Parse found multiple plausible interpretations.
	StatusAmbiguous Status = "ambiguous"
	// StatusInvalid means Parse could not parse the input.
	StatusInvalid Status = "invalid"
)

// Kind identifies the type of the parsed value.
type Kind string

const (
	// KindInstant identifies an absolute UTC timestamp.
	KindInstant Kind = "instant"
	// KindDateTime identifies a zoned local date-time.
	KindDateTime Kind = "datetime"
	// KindLocalDateTime identifies a local date-time without zone or offset.
	KindLocalDateTime Kind = "local_datetime"
	// KindDate identifies a calendar date.
	KindDate Kind = "date"
	// KindTime identifies a clock time.
	KindTime Kind = "time"
	// KindDuration identifies an exact-time duration.
	KindDuration Kind = "duration"
	// KindPeriod identifies a calendar period (years/months/days).
	KindPeriod Kind = "period"
	// KindInterval identifies a half-open time interval.
	KindInterval Kind = "interval"
)

// WarningCode classifies a parse warning. Warnings are non-fatal lossy
// assumptions; they never change Status.
type WarningCode string

const (
	// WarnAssumedZone reports that a default Zone was applied because the
	// input did not carry an explicit zone or offset.
	WarnAssumedZone WarningCode = "assumed_zone"
	// WarnTruncatedPrecision reports that input precision exceeded what the
	// target type can represent and was truncated.
	WarnTruncatedPrecision WarningCode = "truncated_precision"
	// WarnInferredCalendar reports that a date/calendar interpretation was
	// inferred from locale or candidate ordering.
	WarnInferredCalendar WarningCode = "inferred_calendar"
	// WarnDuplicateTime reports that a DST fall-back local time candidate is
	// one of multiple valid instants.
	WarnDuplicateTime WarningCode = "duplicate_time"
)

// Warning is a non-fatal advisory about a lossy assumption made during parsing.
type Warning struct {
	// Code classifies the warning.
	Code WarningCode `json:"code"`
	// Message is a short human-readable description.
	Message string `json:"message"`
	// Hint suggests how to silence the warning.
	Hint string `json:"hint,omitempty"`
}

// ParseResult holds the outcome of Parse.
// Check Status before calling the typed accessors (DateTime, Date, …).
type ParseResult struct {
	// Status reports whether parsing resolved, remained ambiguous, or failed.
	Status Status
	// Kind identifies the semantic type of the parsed value when Status is Resolved or Ambiguous.
	Kind Kind
	// Input is the original input string.
	Input string
	// Zone is the zone applied when the input did not carry its own zone or offset.
	Zone Zone
	// Reference is the reference instant used for relative expressions.
	Reference Instant
	// HasZone reports whether the input explicitly included timezone or offset information.
	HasZone bool
	// Warnings describes lossy assumptions made while parsing.
	Warnings []Warning
	// Candidates holds the alternative interpretations when Status is Ambiguous.
	// Each candidate is itself a StatusResolved ParseResult.
	Candidates []ParseResult
	// Error describes the failure when Status is Invalid.
	Error *TimeError

	// unexported value storage — populated when Status == StatusResolved
	instant       Instant
	dateTime      DateTime
	localDateTime LocalDateTime
	date          Date
	timeVal       Time
	duration      Duration
	period        Period
	interval      Interval
}

// Instant returns the parsed Instant. ok is false unless Status is Resolved and Kind == KindInstant.
func (r ParseResult) Instant() (Instant, bool) {
	if r.Status != StatusResolved || r.Kind != KindInstant {
		return Instant{}, false
	}
	return r.instant, true
}

// DateTime returns the parsed DateTime. ok is false unless Status is Resolved and Kind == KindDateTime.
func (r ParseResult) DateTime() (DateTime, bool) {
	if r.Status != StatusResolved || r.Kind != KindDateTime {
		return DateTime{}, false
	}
	return r.dateTime, true
}

// LocalDateTime returns the parsed LocalDateTime. ok is false unless Status is Resolved and Kind == KindLocalDateTime.
func (r ParseResult) LocalDateTime() (LocalDateTime, bool) {
	if r.Status != StatusResolved || r.Kind != KindLocalDateTime {
		return LocalDateTime{}, false
	}
	return r.localDateTime, true
}

// Date returns the parsed Date. ok is false unless Status is Resolved and Kind == KindDate.
func (r ParseResult) Date() (Date, bool) {
	if r.Status != StatusResolved || r.Kind != KindDate {
		return Date{}, false
	}
	return r.date, true
}

// Time returns the parsed Time. ok is false unless Status is Resolved and Kind == KindTime.
func (r ParseResult) Time() (Time, bool) {
	if r.Status != StatusResolved || r.Kind != KindTime {
		return Time{}, false
	}
	return r.timeVal, true
}

// Duration returns the parsed Duration. ok is false unless Status is Resolved and Kind == KindDuration.
func (r ParseResult) Duration() (Duration, bool) {
	if r.Status != StatusResolved || r.Kind != KindDuration {
		return 0, false
	}
	return r.duration, true
}

// Period returns the parsed Period. ok is false unless Status is Resolved and Kind == KindPeriod.
func (r ParseResult) Period() (Period, bool) {
	if r.Status != StatusResolved || r.Kind != KindPeriod {
		return Period{}, false
	}
	return r.period, true
}

// Interval returns the parsed Interval. ok is false unless Status is Resolved and Kind == KindInterval.
func (r ParseResult) Interval() (Interval, bool) {
	if r.Status != StatusResolved || r.Kind != KindInterval {
		return Interval{}, false
	}
	return r.interval, true
}

// Parse is the inspection / dispatch entry point. It accepts any supported
// input (ISO 8601, RFC 3339, natural language, ranges) and returns a
// [ParseResult] describing the outcome. It never returns a Go error —
// semantic states (ambiguous / invalid) live on [ParseResult.Status].
//
// When you already know which concrete type you expect, prefer the typed
// helpers ([ParseInstant], [ParseDateTime], [ParseLocalDateTime], [ParseDate],
// [ParseTime], [ParseDuration], [ParsePeriod], [ParseInterval]) — they return
// (T, error) directly and skip the Status / Kind dispatch entirely.
//
// Reach for Parse when you need any of:
//
//   - Polymorphic dispatch on Kind via Status, Kind, and comma-ok accessors.
//   - Access to [ParseResult.Candidates] for ambiguous inputs.
//   - Access to [ParseResult.Warnings], [ParseResult.HasZone], or
//     [ParseResult.Reference] metadata.
func Parse(input string, opts ...Option) ParseResult {
	cfg := applyOptions(opts)
	return parseWithConfig(input, &cfg)
}

// MarshalJSON serializes r to the stable JSON schema defined in SPECS/20-parsing.md.
func (r ParseResult) MarshalJSON() ([]byte, error) {
	wire := parseResultWire{
		Kind:     "parse_result",
		Status:   r.Status,
		Input:    r.Input,
		Warnings: r.Warnings,
	}
	switch r.Status {
	case StatusResolved:
		wire.ValueKind = r.Kind
		wire.Value = parseResultValue(r)
		if !r.Zone.IsZero() {
			wire.Zone = r.Zone.ID()
		}
	case StatusAmbiguous:
		wire.ValueKind = r.Kind
		wire.Candidates = r.Candidates
	case StatusInvalid:
		wire.Error = r.Error
	}
	return json.Marshal(wire)
}

type parseResultWire struct {
	Kind       string        `json:"kind"`
	Status     Status        `json:"status"`
	Input      string        `json:"input,omitzero"`
	Warnings   []Warning     `json:"warnings,omitzero"`
	ValueKind  Kind          `json:"value_kind,omitzero"`
	Value      any           `json:"value,omitzero"`
	Zone       string        `json:"zone,omitzero"`
	Candidates []ParseResult `json:"candidates,omitzero"`
	Error      *TimeError    `json:"error,omitzero"`
}

func parseResultValue(r ParseResult) any {
	if r.Status != StatusResolved {
		return nil
	}
	switch r.Kind {
	case KindInstant:
		return r.instant
	case KindDateTime:
		return r.dateTime
	case KindLocalDateTime:
		return r.localDateTime
	case KindDate:
		return r.date
	case KindTime:
		return r.timeVal
	case KindDuration:
		return r.duration
	case KindPeriod:
		return r.period
	case KindInterval:
		return r.interval
	default:
		return nil
	}
}
