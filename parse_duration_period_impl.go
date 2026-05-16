package gotime

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// parseDuration dispatches an ISO 8601 P-prefixed string to either Duration
// (time-only, e.g. PT1H30M) or Period (date-only, e.g. P1Y, P1M, P2W, P5D).
//
// Mixed inputs (date AND time components - e.g. P1Y2DT3H) are rejected with
// CodeInvalidFormat: the gotime type system splits exact-elapsed time
// (Duration) from calendar offsets (Period), and there is no single type
// that can carry both halves correctly. Callers should parse each half
// separately.
func parseDuration(input string, cfg *config) ParseResult {
	m := reDuration.FindStringSubmatch(input)
	if m == nil {
		return invalidResult(input, ErrInvalidFormat,
			fmt.Sprintf("invalid ISO 8601 duration/period: %q", input),
			"Use ISO 8601 form, e.g. PT1H30M (Duration) or P1Y3M (Period)")
	}

	// Capture groups after the sign: 2=Y 3=M 4=W 5=D 6=H 7=M 8=S
	hasDateComp := m[2] != "" || m[3] != "" || m[4] != "" || m[5] != ""
	hasTimeComp := m[6] != "" || m[7] != "" || m[8] != ""

	if !hasDateComp && !hasTimeComp {
		return invalidResult(input, ErrInvalidFormat,
			fmt.Sprintf("empty duration: %q", input),
			"Specify at least one component, e.g. PT0S (Duration) or P0D (Period)")
	}

	if hasDateComp && hasTimeComp {
		return invalidResult(input, ErrInvalidFormat,
			fmt.Sprintf("ISO 8601 input mixes date and time components: %q", input),
			"Calendar offsets (Y/M/W/D) belong to Period; clock spans (H/M/S) belong to Duration. Parse each half separately.")
	}

	if hasDateComp {
		return parseISOPeriodMatch(input, m, cfg)
	}
	return parseISODurationMatch(input, m, cfg)
}

func parseISOPeriodMatch(input string, m []string, cfg *config) ParseResult {
	neg := m[1] == "-"
	var years, months, weeks, days int64
	var ok bool
	if m[2] != "" {
		years, ok = parsePeriodComponent(m[2])
		if !ok {
			return invalidPeriodComponent(input, "year", m[2])
		}
	}
	if m[3] != "" {
		months, ok = parsePeriodComponent(m[3])
		if !ok {
			return invalidPeriodComponent(input, "month", m[3])
		}
	}
	if m[4] != "" {
		weeks, ok = parsePeriodComponent(m[4])
		if !ok {
			return invalidPeriodComponent(input, "week", m[4])
		}
	}
	if m[5] != "" {
		days, ok = parsePeriodComponent(m[5])
		if !ok {
			return invalidPeriodComponent(input, "day", m[5])
		}
	}

	totalDays, ok := checkedAddInt64(weeks*7, days)
	if !ok || years > maxInt32 || months > maxInt32 || totalDays > maxInt32 {
		return invalidResult(input, ErrOverflow,
			fmt.Sprintf("period component overflows int32: %q", input),
			"Use smaller year, month, week, or day components")
	}

	p := Period{
		Years:  int32(years),
		Months: int32(months),
		Days:   int32(totalDays), //nolint:gosec // checked above against maxInt32
	}
	if neg {
		p = p.Negate()
	}
	r := resolvedResult(input, KindPeriod, cfg)
	r.period = p
	return r
}

func parseISODurationMatch(input string, m []string, cfg *config) ParseResult {
	neg := m[1] == "-"
	var totalNs int64
	if m[6] != "" {
		ns, ok := parseDurationComponent(m[6], time.Hour)
		if !ok {
			return invalidDurationComponent(input, "hour", m[6])
		}
		var added bool
		totalNs, added = checkedAddInt64(totalNs, ns)
		if !added {
			return durationOverflow(input)
		}
	}
	if m[7] != "" {
		ns, ok := parseDurationComponent(m[7], time.Minute)
		if !ok {
			return invalidDurationComponent(input, "minute", m[7])
		}
		var added bool
		totalNs, added = checkedAddInt64(totalNs, ns)
		if !added {
			return durationOverflow(input)
		}
	}
	if m[8] != "" {
		ns, ok := parseDurationComponent(m[8], time.Second)
		if !ok {
			return invalidDurationComponent(input, "second", m[8])
		}
		var added bool
		totalNs, added = checkedAddInt64(totalNs, ns)
		if !added {
			return durationOverflow(input)
		}
	}
	if neg {
		totalNs = -totalNs
	}
	r := resolvedResult(input, KindDuration, cfg)
	r.duration = Duration(totalNs)
	return r
}

func parsePeriodComponent(raw string) (int64, bool) {
	if strings.ContainsAny(raw, ".,") {
		return 0, false
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, false
	}
	return n, true
}

func invalidPeriodComponent(input, component, raw string) ParseResult {
	if !strings.ContainsAny(raw, ".,") {
		return invalidResult(input, ErrOverflow,
			fmt.Sprintf("period %s component overflows int32: %q", component, raw),
			"Use a smaller whole-number period component")
	}
	return invalidResult(input, ErrInvalidPeriod,
		fmt.Sprintf("period %s component must be a whole number: %q", component, raw),
		"Use whole-number calendar period components such as P1Y, P2M, P3W, or P4D")
}

func parseDurationComponent(raw string, unit time.Duration) (int64, bool) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, false
	}
	ns := f * float64(unit)
	if ns > float64(maxInt64) || ns < float64(minInt64) {
		return 0, false
	}
	return int64(ns), true
}

func invalidDurationComponent(input, component, raw string) ParseResult {
	return invalidResult(input, ErrOverflow,
		fmt.Sprintf("duration %s component overflows nanoseconds: %q", component, raw),
		"Use a smaller duration component")
}

func durationOverflow(input string) ParseResult {
	return invalidResult(input, ErrOverflow,
		fmt.Sprintf("duration overflows nanoseconds: %q", input),
		"Use a smaller duration")
}

func checkedAddInt64(a, b int64) (int64, bool) {
	if (b > 0 && a > maxInt64-b) || (b < 0 && a < minInt64-b) {
		return 0, false
	}
	return a + b, true
}
