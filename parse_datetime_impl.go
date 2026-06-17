package gotime

import (
	"fmt"
	"strings"
	"time"
)

func tryParseDateTime(input string, cfg *config) (ParseResult, bool) {
	m := reDateTime.FindStringSubmatch(input)
	if m == nil {
		return ParseResult{}, false
	}
	datePart, timePart, offsetPart := m[1], m[2], m[3]
	truncated := hasTruncatedFraction(timePart)
	if !strings.EqualFold(offsetPart, "Z") {
		if _, err := parseOffsetLocation(offsetPart); err != nil {
			return invalidResult(input, ErrInvalidZone,
				fmt.Sprintf("invalid timezone offset %q", offsetPart),
				"Use an offset between -23:59 and +23:59, e.g. +09:00"), true
		}
	}
	fullStr := datePart + "T" + timePart + normalizeOffset(offsetPart)
	t, err := time.Parse(time.RFC3339Nano, fullStr)
	if err != nil {
		// Try without fractional seconds
		t, err = time.Parse(time.RFC3339, fullStr)
		if err != nil {
			return invalidResult(input, ErrInvalidFormat,
				fmt.Sprintf("invalid datetime %q", input),
				"Use RFC 3339 format, e.g. 2026-03-27T13:00:00+09:00"), true
		}
	}
	r := buildDateTimeResult(input, t, cfg)
	if truncated {
		r.Warnings = append(r.Warnings, truncatedPrecisionWarning())
	}
	return r, true
}

func tryParseDateTimeNoOffset(input string, cfg *config) (ParseResult, bool) {
	m := reDateTimeNoOffset.FindStringSubmatch(input)
	if m == nil {
		return ParseResult{}, false
	}
	// Parse time components
	timePart := m[2]
	var hour, min, sec, ns int
	truncated := false
	tp := strings.Split(timePart, ":")
	if len(tp) >= 1 {
		hour = atoi(tp[0])
	}
	if len(tp) >= 2 {
		min = atoi(tp[1])
	}
	if len(tp) >= 3 {
		secStr := tp[2]
		if idx := strings.IndexAny(secStr, ".,"); idx >= 0 {
			ns, truncated = parseFracNano(secStr[idx+1:])
			secStr = secStr[:idx]
		}
		sec = atoi(secStr)
	}
	year, mon, day := parseDateComponents(m[1])
	if year == 0 {
		return ParseResult{}, false
	}

	return localDateTimeResult(input, cfg, year, mon, day, hour, min, sec, ns, truncated)
}

func localDateTimeResult(input string, cfg *config, year, mon, day, hour, min, sec, ns int, truncated bool) (ParseResult, bool) {
	if msg := validateDateComponents(year, mon, day); msg != "" {
		return invalidResult(input, ErrInvalidDate, msg,
			"Provide a valid local calendar date"), true
	}
	if msg := validateTimeComponents(hour, min, sec, ns); msg != "" {
		return invalidResult(input, ErrInvalidTime, msg,
			"Provide a valid local clock time"), true
	}

	if !cfg.zone.IsZero() {
		return resolveLocalDateTime(input, cfg, year, mon, day, hour, min, sec, ns, truncated)
	}

	r := resolvedResult(input, KindLocalDateTime, cfg)
	r.localDateTime = NewLocalDateTime(
		dateFromComponents(year, time.Month(mon), day),
		timeFromComponents(hour, min, sec, ns),
	)
	if truncated {
		r.Warnings = append(r.Warnings, truncatedPrecisionWarning())
	}
	return r, true
}

func resolveLocalDateTime(input string, cfg *config, year, mon, day, hour, min, sec, ns int, truncated bool) (ParseResult, bool) {
	if msg := validateDateComponents(year, mon, day); msg != "" {
		return invalidResult(input, ErrInvalidDate, msg,
			"Provide a valid local calendar date"), true
	}
	if msg := validateTimeComponents(hour, min, sec, ns); msg != "" {
		return invalidResult(input, ErrInvalidTime, msg,
			"Provide a valid local clock time"), true
	}

	z := normalizeZone(cfg.zone)
	loc := z.Location()
	ldt := NewLocalDateTime(
		dateFromComponents(year, time.Month(mon), day),
		timeFromComponents(hour, min, sec, ns),
	)
	resolution := ldt.Resolve(z)

	switch resolution.Status {
	case LocalNonexistent:
		// Compute the post-normalization time for the hint message.
		tNorm := time.Date(year, time.Month(mon), day, hour, min, sec, 0, loc)
		return ParseResult{
			Status: StatusInvalid,
			Input:  input,
			Error: newTimeError(
				ErrNonexistentTime,
				fmt.Sprintf("local time %02d:%02d does not exist on %04d-%02d-%02d in %s (DST spring-forward gap)",
					hour, min, year, mon, day, z.ID()),
				input,
				fmt.Sprintf("Clocks skip %02d:00-%02d:00 in %s on this date. Try %02d:%02d or %02d:%02d instead.",
					hour, tNorm.Hour(), z.ID(), hour-1, min, tNorm.Hour(), tNorm.Minute()),
			),
		}, true
	case LocalAmbiguous:
		candidates := make([]ParseResult, 0, len(resolution.Candidates))
		for _, candidate := range resolution.Candidates {
			candidates = append(candidates, duplicateTimeCandidate(input, cfg, z, candidate.Std(), truncated))
		}

		return ParseResult{
			Status:     StatusAmbiguous,
			Kind:       KindDateTime,
			Input:      input,
			Zone:       cfg.zone,
			Warnings:   localDateTimeWarnings(cfg, z, truncated),
			Candidates: candidates,
		}, true
	case LocalResolved:
		dt := resolution.Candidates[0]
		r := resolvedResult(input, KindDateTime, cfg)
		r.dateTime = dt
		r.Warnings = localDateTimeWarnings(cfg, z, truncated)
		return r, true
	case LocalInvalid:
		return invalidResult(input, ErrInvalidTime,
			fmt.Sprintf("invalid local datetime %q", input),
			"Provide a valid local calendar date and clock time"), true
	default:
		return invalidResult(input, ErrInvalidTime,
			fmt.Sprintf("cannot resolve local datetime %q", input),
			"Provide a valid local calendar date, clock time, and zone"), true
	}
}

func duplicateTimeCandidate(input string, cfg *config, z Zone, t time.Time, truncated bool) ParseResult {
	r := resolvedResult(input, KindDateTime, cfg)
	r.dateTime = DateTime{t: t, zone: z}
	r.Warnings = append(localDateTimeWarnings(cfg, z, truncated), duplicateTimeWarning(t))
	return r
}

func tryParseCompactDateTime(input string, cfg *config) (ParseResult, bool) {
	m := reCompactDateTime.FindStringSubmatch(input)
	if m == nil {
		return ParseResult{}, false
	}
	// Require at least 8 digits + T + 2 digit hour to distinguish from other patterns
	if len(m[1]) != 8 {
		return ParseResult{}, false
	}
	datePart := m[1]
	dateStr := datePart[:4] + "-" + datePart[4:6] + "-" + datePart[6:8]
	hour := intOrZero(m[2])
	min := intOrZero(m[3])
	sec := intOrZero(m[4])
	// Parse fractional seconds
	var nsec int
	truncated := false
	if m[5] != "" {
		nsec, truncated = parseFracNano(m[5])
	}
	year, mon, day := parseDateComponents(dateStr)
	if year == 0 {
		return ParseResult{}, false
	}
	if msg := validateDateComponents(year, mon, day); msg != "" {
		return invalidResult(input, ErrInvalidDate, msg,
			"Provide a valid calendar date, e.g. 20260327T130000"), true
	}
	if msg := validateTimeComponents(hour, min, sec, nsec); msg != "" {
		return invalidResult(input, ErrInvalidTime, msg,
			"Provide a valid clock time, e.g. 20260327T130000"), true
	}
	offsetPart := m[6]
	if offsetPart == "" {
		return localDateTimeResult(input, cfg, year, mon, day, hour, min, sec, nsec, truncated)
	}

	var loc *time.Location
	if offsetPart == "Z" {
		loc = time.UTC
	} else {
		var parseErr error
		loc, parseErr = parseOffsetLocation(offsetPart)
		if parseErr != nil {
			return invalidResult(input, ErrInvalidZone,
				fmt.Sprintf("invalid timezone offset %q", offsetPart),
				"Use an offset between -23:59 and +23:59, e.g. +0900"), true
		}
	}
	t := time.Date(year, time.Month(mon), day, hour, min, sec, nsec, loc)
	r := buildDateTimeResult(input, t, cfg)
	if truncated {
		r.Warnings = append(r.Warnings, truncatedPrecisionWarning())
	}
	return r, true
}

func buildDateTimeResult(input string, t time.Time, cfg *config) ParseResult {
	r := resolvedResult(input, KindInstant, cfg)
	r.Zone = Zone{}
	r.instant = InstantFromTime(t)
	r.HasZone = true
	return r
}
