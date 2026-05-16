package gotime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var errInvalidOffset = errors.New("invalid timezone offset")

const (
	maxInt32 = int64(1<<31 - 1)
	maxInt64 = int64(1<<63 - 1)
	minInt64 = -1 << 63
)

func resolvedResult(input string, kind Kind, cfg *config) ParseResult {
	return ParseResult{
		Status:    StatusResolved,
		Kind:      kind,
		Input:     input,
		Zone:      cfg.zone,
		Reference: cfg.relativeTo,
	}
}

func invalidResult(input string, sentinel error, message, hint string) ParseResult {
	return ParseResult{
		Status: StatusInvalid,
		Input:  input,
		Error:  newTimeError(sentinel, message, input, hint),
	}
}

func normalizeOffset(offset string) string {
	if offset == "Z" || offset == "z" {
		return "Z"
	}
	// Remove colon if missing: +0900 -> +09:00
	if len(offset) == 5 {
		return offset[:3] + ":" + offset[3:]
	}
	return offset
}

func parseOffsetLocation(offset string) (*time.Location, error) {
	norm := normalizeOffset(offset)
	// Parse +/-HH:MM
	if len(norm) < 6 {
		return nil, errInvalidOffset
	}
	sign := 1
	switch norm[0] {
	case '-':
		sign = -1
	case '+':
	default:
		return nil, errInvalidOffset
	}
	h, err1 := strconv.Atoi(norm[1:3])
	m, err2 := strconv.Atoi(norm[4:6])
	if err1 != nil || err2 != nil || h > 23 || m > 59 {
		return nil, errInvalidOffset
	}
	sec := sign * (h*3600 + m*60)
	return time.FixedZone(norm, sec), nil
}

func parseDateComponents(dateStr string) (year, mon, day int) {
	parts := strings.Split(dateStr, "-")
	if len(parts) != 3 {
		return 0, 0, 0
	}
	return atoi(parts[0]), atoi(parts[1]), atoi(parts[2])
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func intOrZero(s string) int {
	if s == "" {
		return 0
	}
	return atoi(s)
}

func localDateTimeWarnings(cfg *config, z Zone, truncated bool) []Warning {
	var warnings []Warning
	if !cfg.zone.IsZero() {
		warnings = append(warnings, assumedZoneWarning(z))
	}
	if truncated {
		warnings = append(warnings, truncatedPrecisionWarning())
	}
	return warnings
}

func assumedZoneWarning(z Zone) Warning {
	return Warning{
		Code:    WarnAssumedZone,
		Message: fmt.Sprintf("zone %s applied to floating input", z.ID()),
		Hint:    "include an explicit timezone or offset in the input to silence this warning",
	}
}

func truncatedPrecisionWarning() Warning {
	return Warning{
		Code:    WarnTruncatedPrecision,
		Message: "fractional seconds exceeded nanosecond precision and were truncated",
		Hint:    "use at most 9 fractional second digits",
	}
}

func duplicateTimeWarning(t time.Time) Warning {
	abbr, offset := t.Zone()
	return Warning{
		Code:    WarnDuplicateTime,
		Message: fmt.Sprintf("%s (%s)", abbr, formatOffset(offset)),
		Hint:    "Choose this candidate when its offset matches the intended occurrence.",
	}
}

func hasTruncatedFraction(s string) bool {
	idx := strings.IndexAny(s, ".,")
	return idx >= 0 && len(s[idx+1:]) > 9
}

func parseFracNano(frac string) (int, bool) {
	truncated := len(frac) > 9
	// frac is the digits after the decimal point, up to 9 digits
	if truncated {
		frac = frac[:9]
	}
	for len(frac) < 9 {
		frac += "0"
	}
	ns, _ := strconv.Atoi(frac)
	return ns, truncated
}
