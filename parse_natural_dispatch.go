package gotime

import (
	"fmt"
	"time"

	"github.com/agentable/go-time/internal/natural"
)

// tryParseNatural attempts natural language parsing after ISO 8601 fails.
func tryParseNatural(input string, cfg *config) (ParseResult, bool) {
	if cfg.lang.IsRoot() {
		return ParseResult{}, false
	}

	var reference time.Time
	if cfg.referenceSet && cfg.zoneSet {
		projected := cfg.relativeTo.Std().In(cfg.zone.Location())
		reference = time.Date(
			projected.Year(), projected.Month(), projected.Day(),
			projected.Hour(), projected.Minute(), projected.Second(), projected.Nanosecond(),
			time.UTC,
		)
	}
	ctx := natural.Context{
		Locale:     cfg.lang.String(),
		RelativeTo: reference,
	}

	r, ok := natural.Parse(input, ctx)
	if !ok {
		return ParseResult{}, false
	}
	if r.NeedsReference && !cfg.referenceSet {
		return invalidResult(input, ErrInvalidFormat,
			"natural language expression requires a reference instant",
			"pass WithReference(gotime.Now()) at the product boundary or provide a fixed reference for deterministic parsing"), true
	}
	if r.NeedsReference && !cfg.zoneSet {
		return invalidResult(input, ErrInvalidZone,
			"natural language expression requires a zone for its calendar reference",
			"pass WithZone with the IANA zone used to interpret the reference instant"), true
	}
	return naturalResultToParseResult(input, &r, cfg), true
}

// naturalResultToParseResult maps a natural.Result to a ParseResult.
func naturalResultToParseResult(input string, r *natural.Result, cfg *config) ParseResult {
	switch r.Kind {
	case natural.KindDate:
		if msg := validateDateComponents(r.Year, int(r.Month), r.Day); msg != "" {
			return invalidResult(
				input,
				ErrInvalidDate,
				fmt.Sprintf("natural date is outside the supported civil domain: %s", msg),
				"use a reference and expression whose result is between 0000-01-01 and 9999-12-31",
			)
		}
		pr := resolvedResult(input, KindDate, cfg)
		pr.date = dateFromComponents(r.Year, r.Month, r.Day)
		return pr

	case natural.KindDateTime:
		pr, _ := localDateTimeResult(
			input,
			cfg,
			r.Year,
			int(r.Month),
			r.Day,
			r.Hour,
			r.Minute,
			r.Second,
			r.Nanosecond,
			false,
		)
		return pr

	case natural.KindDuration:
		pr := resolvedResult(input, KindDuration, cfg)
		pr.duration = Duration(r.DurNanos)
		return pr

	case natural.KindPeriod:
		pr := resolvedResult(input, KindPeriod, cfg)
		pr.period = Period{
			Years:  r.PeriodYears,
			Months: r.PeriodMonths,
			Days:   r.PeriodDays,
		}
		return pr

	case natural.KindAmbiguous:
		// Natural parsers don't produce ambiguous results in Tier 1.
		return invalidResult(input, ErrUnparseable,
			fmt.Sprintf("ambiguous natural language expression: %q", input),
			"Provide a more specific time expression or add a locale hint")

	case natural.KindInvalid:
		return invalidResult(input, sentinelForCode(ErrorCode(r.ErrCode)), r.ErrMessage, r.ErrHint)

	default:
		return invalidResult(input, ErrUnparseable,
			fmt.Sprintf("natural language parser returned unexpected kind: %v", r.Kind),
			"Report this as a bug")
	}
}
