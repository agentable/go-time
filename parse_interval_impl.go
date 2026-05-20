package gotime

import (
	"fmt"
	"strings"
)

func parseInterval(input string, cfg *config) ParseResult {
	left, right, ok := strings.Cut(input, "/")
	if !ok {
		return invalidResult(input, ErrInvalidFormat,
			"interval requires start/end separated by /",
			"Use ISO 8601 interval format: start/end, start/duration, or duration/end")
	}

	startIsDur := strings.HasPrefix(left, "P")
	endIsDur := strings.HasPrefix(right, "P")

	var startInstant, endInstant Instant

	switch {
	case !startIsDur && !endIsDur:
		sr, ok := parseIntervalPart(input, left, "start", WithZone(cfg.zone))
		if !ok {
			return sr
		}
		er, ok := parseIntervalPart(input, right, "end", WithZone(cfg.zone))
		if !ok {
			return er
		}
		startInstant, ok = toInstant(&sr)
		if !ok {
			return invalidIntervalType(input, "start", sr.Kind)
		}
		endInstant, ok = toInstant(&er)
		if !ok {
			return invalidIntervalType(input, "end", er.Kind)
		}

	case !startIsDur && endIsDur:
		sr, ok := parseIntervalPart(input, left, "start", WithZone(cfg.zone))
		if !ok {
			return sr
		}
		dr, ok := parseIntervalPart(input, right, "duration")
		if !ok {
			return dr
		}
		startInstant, ok = toInstant(&sr)
		if !ok {
			return invalidIntervalType(input, "start", sr.Kind)
		}
		endInstant = startInstant.Add(dr.duration)

	case startIsDur && !endIsDur:
		dr, ok := parseIntervalPart(input, left, "duration")
		if !ok {
			return dr
		}
		er, ok := parseIntervalPart(input, right, "end", WithZone(cfg.zone))
		if !ok {
			return er
		}
		endInstant, ok = toInstant(&er)
		if !ok {
			return invalidIntervalType(input, "end", er.Kind)
		}
		startInstant = endInstant.Add(-dr.duration)

	default:
		return invalidResult(input, ErrInvalidFormat,
			"interval cannot have two duration components",
			"Use ISO 8601 interval format: start/end, start/duration, or duration/end")
	}

	if endInstant.Before(startInstant) {
		return invalidResult(input, ErrIntervalReversed,
			"interval end is before start",
			"Ensure the interval end is at or after the start")
	}
	iv, err := NewInterval(startInstant, endInstant)
	if err != nil {
		return invalidResult(input, ErrIntervalReversed,
			"interval end is before start",
			"Ensure the interval end is at or after the start")
	}
	r := resolvedResult(input, KindInterval, cfg)
	r.interval = iv
	return r
}

func parseIntervalPart(input, part, label string, opts ...Option) (ParseResult, bool) {
	r := Parse(part, opts...)
	switch r.Status {
	case StatusResolved:
		if label == "duration" {
			if r.Kind == KindDuration {
				return r, true
			}
			return invalidResult(input, ErrIncompatibleTypes,
				fmt.Sprintf("interval %s must be a duration, got %s", label, r.Kind),
				"Use a PT-prefixed exact duration such as PT1H30M"), false
		}
		if r.Kind == KindInstant || r.Kind == KindDateTime {
			return r, true
		}
		return invalidIntervalType(input, label, r.Kind), false
	case StatusAmbiguous:
		r.Input = input
		return r, false
	case StatusInvalid:
		message := "parse failed"
		hint := "Use ISO 8601 interval format"
		if r.Error != nil {
			message = r.Error.Message
			hint = r.Error.Hint
		}
		return invalidResult(input, ErrInvalidFormat,
			fmt.Sprintf("invalid interval %s: %s", label, message),
			hint), false
	default:
		return invalidResult(input, ErrUnparseable,
			fmt.Sprintf("invalid interval %s", label),
			"Use ISO 8601 interval format"), false
	}
}

func invalidIntervalType(input, label string, kind Kind) ParseResult {
	return invalidResult(input, ErrIncompatibleTypes,
		fmt.Sprintf("interval %s must resolve to instant or datetime, got %s", label, kind),
		"Use explicit datetimes with timezone information for interval boundaries")
}

func toInstant(r *ParseResult) (Instant, bool) {
	switch r.Kind {
	case KindInstant:
		return r.instant, true
	case KindDateTime:
		return r.dateTime.Instant(), true
	default:
		return Instant{}, false
	}
}
