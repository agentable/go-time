package gotime

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

func parseInterval(input string, cfg *config) ParseResult {
	left, right, ok := strings.Cut(input, "/")
	if !ok {
		return invalidResult(input, ErrInvalidFormat,
			"interval requires start/end separated by /",
			"Use ISO 8601 interval format: start/end, start/duration, or duration/end")
	}

	startIsDur := strings.HasPrefix(left, "P") || strings.HasPrefix(left, "-P")
	endIsDur := strings.HasPrefix(right, "P") || strings.HasPrefix(right, "-P")

	if startIsDur && endIsDur {
		return invalidResult(input, ErrInvalidFormat, "interval cannot have two duration components",
			"Use ISO 8601 interval format: start/end, start/duration, or duration/end")
	}
	var sr, er ParseResult
	if startIsDur {
		sr, _ = parseIntervalDuration(input, left)
	} else {
		sr, _ = parseIntervalBoundary(input, left, "start", cfg)
	}
	if endIsDur {
		er, _ = parseIntervalDuration(input, right)
	} else {
		er, _ = parseIntervalBoundary(input, right, "end", cfg)
	}
	if sr.Status == StatusInvalid {
		return sr
	}
	if er.Status == StatusInvalid {
		return er
	}

	var candidates []ParseResult
	starts, ends := intervalPartCandidates(sr), intervalPartCandidates(er)
	for si := range starts {
		for ei := range ends {
			iv, err := intervalFromParts(starts[si], ends[ei])
			if errors.Is(err, ErrIntervalReversed) {
				continue
			}
			if err != nil {
				return ParseResult{Status: StatusInvalid, Input: input, Error: newTimeErrorWithCause(
					ErrOverflow, err, "interval arithmetic overflow", input, "use an endpoint and duration whose result can be represented")}
			}
			candidate := intervalParseResult(input, cfg, starts[si], ends[ei])
			candidate.interval = iv
			candidates = append(candidates, candidate)
		}
	}
	slices.SortFunc(candidates, func(a, b ParseResult) int {
		if order := a.interval.start.Compare(b.interval.start); order != 0 {
			return order
		}
		return a.interval.end.Compare(b.interval.end)
	})
	candidates = slices.CompactFunc(candidates, func(a, b ParseResult) bool {
		return a.interval.start.Equal(b.interval.start) && a.interval.end.Equal(b.interval.end)
	})
	switch len(candidates) {
	case 0:
		return invalidResult(input, ErrIntervalReversed, "interval end is before start", "Ensure the interval end is at or after the start")
	case 1:
		return candidates[0]
	default:
		r := intervalParseResult(input, cfg, sr, er)
		r.Status = StatusAmbiguous
		r.Candidates = candidates
		r.ambiguity = ambiguityDuplicateTime
		return r
	}
}

func intervalPartCandidates(r ParseResult) []ParseResult {
	if r.Status == StatusAmbiguous {
		return r.Candidates
	}
	return []ParseResult{r}
}

func intervalFromParts(start, end ParseResult) (Interval, error) {
	startInstant, _ := toInstant(&start)
	endInstant, _ := toInstant(&end)
	if start.Kind == KindDuration {
		return NewIntervalEndingAt(endInstant, start.duration)
	}
	if end.Kind == KindDuration {
		return NewIntervalStartingAt(startInstant, end.duration)
	}
	return NewInterval(startInstant, endInstant)
}

func parseIntervalBoundary(input, part, label string, cfg *config) (ParseResult, bool) {
	if r, ok := parseDateTimeStage(part, cfg); ok {
		return validateIntervalPart(input, label, r)
	}
	if r, ok := parseDateStage(part, cfg); ok {
		return validateIntervalPart(input, label, r)
	}
	if r, ok := parseTimeStage(part, cfg); ok {
		return validateIntervalPart(input, label, r)
	}
	return invalidResult(input, ErrInvalidFormat,
		fmt.Sprintf("invalid interval %s: parse failed", label),
		"Use ISO 8601 interval format"), false
}

func parseIntervalDuration(input, part string) (ParseResult, bool) {
	var cfg config
	return validateIntervalPart(input, "duration", parseDuration(part, &cfg))
}

func validateIntervalPart(input, label string, r ParseResult) (ParseResult, bool) {
	switch r.Status {
	case StatusResolved:
		if label == "duration" {
			if r.Kind == KindDuration {
				if r.duration.IsNegative() {
					return invalidResult(input, ErrInvalidDuration, "interval length must not be negative", "use a non-negative exact duration"), false
				}
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
		if r.Error != nil {
			err := *r.Error
			err.Message = fmt.Sprintf("invalid interval %s: %s", label, r.Error.Message)
			err.Input = input
			r.Input = input
			r.Error = &err
			return r, false
		}
		return invalidResult(input, ErrInvalidFormat,
			fmt.Sprintf("invalid interval %s: parse failed", label),
			"Use ISO 8601 interval format"), false
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

func intervalParseResult(input string, cfg *config, start, end ParseResult) ParseResult {
	r := resolvedResult(input, KindInterval, cfg)
	r.HasZone = start.HasZone || end.HasZone
	for _, part := range []*ParseResult{&start, &end} {
		for _, warning := range part.Warnings {
			if !slices.ContainsFunc(r.Warnings, func(existing Warning) bool {
				return existing.Code == warning.Code && existing.Message == warning.Message
			}) {
				r.Warnings = append(r.Warnings, warning)
			}
		}
	}
	return r
}
