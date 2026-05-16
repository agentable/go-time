package gotime

import "time"

// monthFirstLocales lists locales whose conventional date ordering is
// month/day/year. Every other non-empty locale is treated as day-first.
var monthFirstLocales = map[string]bool{
	"en-US": true, "en-CA": true, "en-AU": true,
}

// parseSlashDate resolves "a/b/year" using only the input locale:
//
//   - Locale is month-first → month-first interpretation wins.
//   - Locale is non-empty but not month-first → day-first wins.
//   - Locale is empty/unknown → try both; if exactly one parses as a valid
//     calendar date, return it; otherwise return Ambiguous candidates so
//     the caller can pick.
func parseSlashDate(input string, m []string, cfg *config) ParseResult {
	a, b, year := atoi(m[1]), atoi(m[2]), atoi(m[3])

	if !cfg.lang.IsRoot() {
		return resolvedSlashDate(input, a, b, year, monthFirstLocales[cfg.lang.String()], cfg)
	}

	monthFirstValid := validateDateComponents(year, a, b) == ""
	dayFirstValid := validateDateComponents(year, b, a) == ""
	switch {
	case monthFirstValid && !dayFirstValid:
		return resolvedSlashDate(input, a, b, year, true, cfg)
	case dayFirstValid && !monthFirstValid:
		return resolvedSlashDate(input, a, b, year, false, cfg)
	default:
		return ambiguousSlashDate(input, a, b, year, cfg)
	}
}

func resolvedSlashDate(input string, a, b, year int, monthFirst bool, cfg *config) ParseResult {
	var month, day int
	if monthFirst {
		month, day = a, b
	} else {
		day, month = a, b
	}
	if msg := validateDateComponents(year, month, day); msg != "" {
		return invalidResult(input, ErrInvalidDate, msg,
			"Provide a valid calendar date")
	}
	d := dateFromComponents(year, time.Month(month), day)
	r := resolvedResult(input, KindDate, cfg)
	r.date = d
	return r
}

func ambiguousSlashDate(input string, a, b, year int, cfg *config) ParseResult {
	c1 := resolvedSlashDate(input, a, b, year, true, cfg)
	c1.Warnings = append(c1.Warnings, Warning{Code: WarnInferredCalendar, Message: "month-first interpretation (e.g. en-US)"})
	c2 := resolvedSlashDate(input, a, b, year, false, cfg)
	c2.Warnings = append(c2.Warnings, Warning{Code: WarnInferredCalendar, Message: "day-first interpretation (e.g. en-GB)"})
	return ParseResult{
		Status:     StatusAmbiguous,
		Kind:       KindDate,
		Input:      input,
		Zone:       cfg.zone,
		Candidates: []ParseResult{c1, c2},
	}
}
