// Package natural implements the internal natural-language parsing layer for gotime.
// It uses only standard-library time types so it can stay decoupled from the parent package.
package natural

import (
	"strings"
	"time"

	ianazone "github.com/agentable/go-time/internal/zone"
)

// Kind classifies the semantic output of natural language parsing.
type Kind uint8

const (
	// KindDate is a calendar date without a clock time.
	KindDate Kind = iota + 1
	// KindDateTime is a zoned local date-time.
	KindDateTime
	// KindDuration is an elapsed relative duration.
	KindDuration
	// KindPeriod is a calendar relative period.
	KindPeriod
	// KindAmbiguous represents multiple valid interpretations.
	KindAmbiguous
	// KindInvalid represents a parse failure.
	KindInvalid
)

// Context provides the parsing environment, using only stdlib types.
type Context struct {
	// Locale is the BCP 47 locale tag used to select a parser.
	Locale string
	// ZoneID is the IANA zone ID used for relative expressions.
	ZoneID string
	// RelativeTo is the reference time for relative expressions.
	RelativeTo time.Time
}

// Result is the intermediate parse output.
type Result struct {
	// Kind classifies the parsed value.
	Kind Kind
	// Time is the resolved time for KindDate and KindDateTime results.
	Time time.Time
	// DateOnly reports whether Time should be interpreted as a date without a clock time.
	DateOnly bool
	// ZoneID is the zone associated with the result when one applies.
	ZoneID string
	// NeedsReference reports whether the result was resolved from Context.RelativeTo.
	NeedsReference bool
	// DurNanos is the parsed duration when Kind is KindDuration.
	DurNanos int64
	// Period fields hold calendar offsets when Kind is KindPeriod.
	PeriodYears  int32
	PeriodMonths int32
	PeriodDays   int32
	// Candidates holds the alternative interpretations when Kind is KindAmbiguous.
	Candidates []Result
	// ErrCode is the stable error code when Kind is KindInvalid.
	ErrCode string
	// ErrMessage is the human-readable error summary when Kind is KindInvalid.
	ErrMessage string
	// ErrHint explains how to fix the invalid input when Kind is KindInvalid.
	ErrHint string
}

// parser is implemented by each locale-specific parser.
type parser interface {
	canHandle(locale string) bool
	parse(input string, ctx Context) (Result, bool)
}

// registered holds the locale parsers in dispatch order.
var registered []parser

// register adds a parser to the dispatch list (called from init()).
func register(p parser) {
	registered = append(registered, p)
}

// Parse dispatches to the locale-appropriate parser.
// It returns (result, true) when a registered parser recognizes the input.
func Parse(input string, ctx Context) (Result, bool) {
	input = strings.TrimSpace(input)
	if input == "" || ctx.Locale == "" {
		return Result{}, false
	}
	for _, p := range registered {
		if !p.canHandle(ctx.Locale) {
			continue
		}
		if r, ok := p.parse(input, ctx); ok {
			return r, true
		}
	}
	return Result{}, false
}

func matchesLocalePrefix(locale, prefix string) bool {
	if locale == prefix {
		return true
	}
	rest, ok := strings.CutPrefix(locale, prefix)
	return ok && strings.HasPrefix(rest, "-")
}

// locForZone returns a *time.Location for zoneID.
// Falls back to UTC if the zone cannot be loaded.
func locForZone(zoneID string) *time.Location {
	if _, loc, ok := ianazone.ResolveLocation(zoneID); ok {
		return loc
	}
	return time.UTC
}

// midnightInLoc returns t truncated to midnight in loc.
func midnightInLoc(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// nextWeekday returns the next occurrence of wd strictly after base.
// On a Friday, "next Friday" is +7 days — the same Friday is "this Friday".
func nextWeekday(base time.Time, wd time.Weekday) time.Time {
	diff := int(wd) - int(base.Weekday())
	if diff <= 0 {
		diff += 7
	}
	return base.AddDate(0, 0, diff)
}

// thisWeekday returns the occurrence of wd in the current week containing base.
// "current week" is defined as Mon–Sun starting from the Monday on or before base.
func thisWeekday(base time.Time, wd time.Weekday) time.Time {
	// Find Monday of current week
	daysFromMon := int(base.Weekday()) - int(time.Monday)
	if daysFromMon < 0 {
		daysFromMon += 7
	}
	monday := base.AddDate(0, 0, -daysFromMon)
	// Advance to target weekday
	daysToWd := int(wd) - int(time.Monday)
	if daysToWd < 0 {
		daysToWd += 7
	}
	return monday.AddDate(0, 0, daysToWd)
}

// lastWeekday returns the most recent past occurrence of wd before base.
func lastWeekday(base time.Time, wd time.Weekday) time.Time {
	diff := int(base.Weekday()) - int(wd)
	if diff <= 0 {
		diff += 7
	}
	return base.AddDate(0, 0, -diff)
}

// dateResult builds a KindDate Result at midnight in the given location.
func dateResult(t time.Time, zoneID string) Result {
	return Result{
		Kind:           KindDate,
		Time:           t,
		DateOnly:       true,
		ZoneID:         zoneID,
		NeedsReference: true,
	}
}

// datetimeResult builds a KindDateTime Result.
func datetimeResult(t time.Time, zoneID string) Result {
	return Result{
		Kind:           KindDateTime,
		Time:           t,
		ZoneID:         zoneID,
		NeedsReference: true,
	}
}

// durationResult builds a KindDuration Result.
func durationResult(nanos int64) Result {
	return Result{
		Kind:     KindDuration,
		DurNanos: nanos,
	}
}

// periodResult builds a KindPeriod Result.
func periodResult(years, months, days int32) Result {
	return Result{
		Kind:         KindPeriod,
		PeriodYears:  years,
		PeriodMonths: months,
		PeriodDays:   days,
	}
}

func invalidResult(code, message, hint string) Result {
	return Result{
		Kind:       KindInvalid,
		ErrCode:    code,
		ErrMessage: message,
		ErrHint:    hint,
	}
}

// datetimeAt builds a KindDateTime Result at hour:min on the given date.
func datetimeAt(dateBase time.Time, hour, min int, loc *time.Location, zoneID string) Result {
	t := time.Date(dateBase.Year(), dateBase.Month(), dateBase.Day(), hour, min, 0, 0, loc)
	return datetimeResult(t, zoneID)
}

// applyHourPeriod converts a 12-hour clock value to 0–23 using AM/PM markers.
// Pass "" for marker to return hour unchanged.
func applyHourPeriod(hour int, marker, amMark, pmMark string) int {
	switch marker {
	case amMark:
		if hour == 12 {
			return 0
		}
		return hour
	case pmMark:
		if hour < 12 {
			return hour + 12
		}
		return hour
	default:
		return hour
	}
}

// relativeUnitResult returns Duration for exact units and Period for calendar units.
const (
	minInt32 = -1 << 31
	maxInt32 = 1<<31 - 1
)

func relativeUnitResult(unit string, n int64) Result {
	switch unit {
	case "day":
		return checkedPeriodDays(n)
	case "week":
		days, ok := checkedMulInt64(n, 7)
		if !ok {
			return invalidResult("OVERFLOW", "calendar week count overflows int32 days", "use a smaller week count")
		}
		return checkedPeriodDays(days)
	case "month":
		return checkedPeriodMonths(n)
	case "year":
		return checkedPeriodYears(n)
	case "second", "minute", "hour":
		nanos, ok := checkedMulInt64(canonicalNanos(unit), n)
		if !ok {
			return invalidResult("OVERFLOW", "duration overflows nanoseconds", "use a smaller duration")
		}
		return durationResult(nanos)
	default:
		return invalidResult("INVALID_DURATION", "unknown relative unit", "use second, minute, hour, day, week, month, or year")
	}
}

// canonicalNanos returns the nanosecond value for a canonical exact unit name.
// Recognized units are "second", "minute", and "hour".
func canonicalNanos(unit string) int64 {
	switch unit {
	case "second":
		return int64(time.Second)
	case "minute":
		return int64(time.Minute)
	case "hour":
		return int64(time.Hour)
	default:
		return int64(time.Second)
	}
}

func checkedPeriodYears(n int64) Result {
	if n < minInt32 || n > maxInt32 {
		return invalidResult("OVERFLOW", "calendar year count overflows int32", "use a smaller year count")
	}
	return periodResult(int32(n), 0, 0)
}

func checkedPeriodMonths(n int64) Result {
	if n < minInt32 || n > maxInt32 {
		return invalidResult("OVERFLOW", "calendar month count overflows int32", "use a smaller month count")
	}
	return periodResult(0, int32(n), 0)
}

func checkedPeriodDays(n int64) Result {
	if n < minInt32 || n > maxInt32 {
		return invalidResult("OVERFLOW", "calendar day count overflows int32", "use a smaller day count")
	}
	return periodResult(0, 0, int32(n))
}

func checkedMulInt64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	result := a * b
	return result, result/b == a
}
