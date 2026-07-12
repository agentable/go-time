package gotime

import (
	"fmt"
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
	for _, component := range []struct {
		raw    string
		name   string
		target *int64
	}{
		{m[2], "year", &years},
		{m[3], "month", &months},
		{m[4], "week", &weeks},
		{m[5], "day", &days},
	} {
		if component.raw == "" {
			continue
		}
		n, ok := parsePeriodWireMagnitude(component.raw)
		if !ok {
			return invalidPeriodComponent(input, component.name, component.raw)
		}
		*component.target = n
	}

	totalDays := weeks*7 + days
	if neg {
		years, months, totalDays = -years, -months, -totalDays
	}
	p, ok := periodFromInt64(years, months, totalDays)
	if !ok {
		return invalidResult(input, ErrOverflow,
			fmt.Sprintf("period component overflows int32: %q", input),
			"Use smaller year, month, week, or day components")
	}
	r := resolvedResult(input, KindPeriod, cfg)
	r.period = p
	return r
}

func parseISODurationMatch(input string, m []string, cfg *config) ParseResult {
	neg := m[1] == "-"
	var totalNs int64
	for _, component := range []struct {
		raw  string
		name string
		unit time.Duration
	}{
		{m[6], "hour", time.Hour},
		{m[7], "minute", time.Minute},
		{m[8], "second", time.Second},
	} {
		if component.raw == "" {
			continue
		}
		ns, ok := parseDurationComponent(component.raw, component.unit)
		if !ok {
			return invalidDurationComponent(input, component.name, component.raw)
		}
		total, ok := checkedAddInt64(totalNs, ns)
		if !ok {
			return durationOverflow(input)
		}
		totalNs = total
	}
	if neg {
		totalNs = -totalNs
	}
	r := resolvedResult(input, KindDuration, cfg)
	r.duration = Duration(totalNs)
	return r
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
	wholeText, fracText, hasFrac := cutDecimal(raw)
	whole, err := strconv.ParseInt(wholeText, 10, 64)
	if err != nil {
		return 0, false
	}
	unitNs := int64(unit)
	if whole > maxInt64/unitNs {
		return 0, false
	}
	total := whole * unitNs
	if !hasFrac {
		return total, true
	}
	if fracText == "" || len(fracText) > 9 {
		return 0, false
	}
	frac, err := strconv.ParseInt(fracText, 10, 64)
	if err != nil {
		return 0, false
	}
	scale := pow10(len(fracText))
	fractionNs := frac * (unitNs / scale)
	return checkedAddInt64(total, fractionNs)
}

func cutDecimal(raw string) (whole, frac string, ok bool) {
	if whole, frac, ok = strings.Cut(raw, "."); ok {
		return whole, frac, true
	}
	whole, frac, ok = strings.Cut(raw, ",")
	return whole, frac, ok
}

func pow10(n int) int64 {
	p := int64(1)
	for range n {
		p *= 10
	}
	return p
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
