package gotime

import (
	"fmt"
	"strings"
	"time"

	ianazone "github.com/agentable/go-time/internal/zone"
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
	r := buildDateTimeResult(input, t, offsetPart, cfg)
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

	return resolveLocalDateTime(input, cfg, year, mon, day, hour, min, sec, ns, truncated)
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

	z := cfg.zone
	if z.IsZero() {
		z = UTC
	}
	loc := z.Location()

	// Use ProjectLocalTime to detect DST spring-forward and fall-back.
	res := ianazone.ProjectLocalTime(loc, year, time.Month(mon), day, hour, min, sec)
	switch res.Status {
	case ianazone.DSTNonexistent:
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
	case ianazone.DSTAmbiguous:
		// Preserve nanosecond precision on both candidates.
		t1 := res.Times[0].Add(time.Duration(ns))
		t2 := res.Times[1].Add(time.Duration(ns))
		r1 := duplicateTimeCandidate(input, cfg, z, t1, truncated)
		r2 := duplicateTimeCandidate(input, cfg, z, t2, truncated)

		return ParseResult{
			Status:     StatusAmbiguous,
			Kind:       KindDateTime,
			Input:      input,
			Zone:       cfg.zone,
			Warnings:   localDateTimeWarnings(cfg, z, truncated),
			Candidates: []ParseResult{r1, r2},
		}, true
	case ianazone.DSTNormal:
		t := time.Date(year, time.Month(mon), day, hour, min, sec, ns, loc)
		dt := DateTime{t: t, zone: z}
		r := resolvedResult(input, KindDateTime, cfg)
		r.dateTime = dt
		r.Warnings = localDateTimeWarnings(cfg, z, truncated)
		return r, true
	}
	return ParseResult{}, false
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
		return resolveLocalDateTime(input, cfg, year, mon, day, hour, min, sec, nsec, truncated)
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
	r := buildDateTimeResult(input, t, offsetPart, cfg)
	if truncated {
		r.Warnings = append(r.Warnings, truncatedPrecisionWarning())
	}
	return r, true
}

func buildDateTimeResult(input string, t time.Time, offsetPart string, cfg *config) ParseResult {
	// RFC 3339 UTC -> Instant
	if strings.EqualFold(offsetPart, "Z") {
		i := InstantFromTime(t)
		r := resolvedResult(input, KindInstant, cfg)
		r.Zone = Zone{}
		r.instant = i
		r.HasZone = true
		return r
	}
	// Non-UTC offset -> DateTime with fixed-offset zone. WithZone only
	// supplies a zone for floating datetimes; explicit offsets win.
	_, offsetSec := t.Zone()
	offsetID := formatOffset(offsetSec)
	loc := time.FixedZone(offsetID, offsetSec)
	t = t.In(loc)
	z := Zone{id: offsetID, loc: loc}
	dt := DateTime{t: t, zone: z}
	r := resolvedResult(input, KindDateTime, cfg)
	r.Zone = Zone{}
	r.dateTime = dt
	r.HasZone = true
	return r
}
