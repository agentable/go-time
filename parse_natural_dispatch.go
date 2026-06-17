package gotime

import (
	"fmt"
	"time"

	"github.com/agentable/go-time/internal/natural"
	ianazone "github.com/agentable/go-time/internal/zone"
)

// tryParseNatural attempts natural language parsing after ISO 8601 fails.
func tryParseNatural(input string, cfg *config) (ParseResult, bool) {
	if cfg.lang.IsRoot() {
		return ParseResult{}, false
	}

	var refTime time.Time
	if !cfg.relativeTo.IsZero() {
		refTime = cfg.relativeTo.Std()
	} else {
		refTime = time.Now()
	}

	ctx := natural.Context{
		Locale:     cfg.lang.String(),
		ZoneID:     cfg.zone.ID(),
		RelativeTo: refTime,
	}

	r, ok := natural.Parse(input, ctx)
	if !ok {
		return ParseResult{}, false
	}
	return naturalResultToParseResult(input, &r, cfg), true
}

// naturalResultToParseResult maps a natural.Result to a ParseResult.
func naturalResultToParseResult(input string, r *natural.Result, cfg *config) ParseResult {
	switch r.Kind {
	case natural.KindDate:
		d := DateFromTime(r.Time.In(configuredZoneIDOrUTC(r.ZoneID).Location()))
		pr := resolvedResult(input, KindDate, cfg)
		pr.date = d
		return pr

	case natural.KindDateTime:
		z := configuredZoneIDOrUTC(r.ZoneID)
		loc := z.Location()
		if cfg.zone.IsZero() {
			t := r.Time.In(loc)
			pr := resolvedResult(input, KindLocalDateTime, cfg)
			pr.localDateTime = NewLocalDateTime(DateFromTime(t), TimeFromTime(t))
			return pr
		}
		dt := DateTimeFromTime(r.Time.In(loc), z)
		pr := resolvedResult(input, KindDateTime, cfg)
		pr.dateTime = dt
		if !cfg.zone.IsZero() {
			pr.Warnings = append(pr.Warnings, assumedZoneWarning(z))
		}
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

func configuredZoneIDOrUTC(zoneID string) Zone {
	if zoneID == "" {
		return UTC
	}
	if canonical, loc, ok := ianazone.ResolveLocation(zoneID); ok {
		return Zone{id: canonical, loc: loc}
	}
	return UTC
}
