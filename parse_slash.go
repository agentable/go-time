package gotime

import (
	"fmt"
	"time"

	"golang.org/x/text/language"
)

type slashDateOrder uint8

const (
	slashMonthFirst slashDateOrder = iota + 1
	slashDayFirst
)

var slashDateOrders = map[string]slashDateOrder{
	"US": slashMonthFirst,
	"GB": slashDayFirst,
	"AU": slashDayFirst,
}

// parseSlashDate resolves "a/b/year" using a closed locale policy:
//
//   - A supported locale selects its documented slash order.
//   - An unsupported locale tries both; if exactly one parses as a valid
//     calendar date, return it; otherwise return Ambiguous candidates so
//     the caller can pick.
func parseSlashDate(input string, m []string, cfg *config) ParseResult {
	a, b, year := atoi(m[1]), atoi(m[2]), atoi(m[3])

	if order, ok := slashDateOrderForTag(cfg.lang); ok {
		return resolvedSlashDate(input, a, b, year, order == slashMonthFirst, cfg)
	}

	monthFirstValid := validateDateComponents(year, a, b) == ""
	dayFirstValid := validateDateComponents(year, b, a) == ""
	switch {
	case monthFirstValid && dayFirstValid:
		return ambiguousSlashDate(input, a, b, year, cfg)
	case monthFirstValid && !dayFirstValid:
		return resolvedSlashDate(input, a, b, year, true, cfg)
	case dayFirstValid && !monthFirstValid:
		return resolvedSlashDate(input, a, b, year, false, cfg)
	default:
		return invalidResult(
			input,
			ErrInvalidDate,
			fmt.Sprintf("slash date %q has no valid month/day interpretation", input),
			"provide a real calendar date or use ISO 8601 form YYYY-MM-DD",
		)
	}
}

func slashDateOrderForTag(tag language.Tag) (slashDateOrder, bool) {
	if tag.IsRoot() || len(tag.Variants()) != 0 {
		return 0, false
	}
	for _, extension := range tag.Extensions() {
		if extension.Type() != 'u' {
			return 0, false
		}
	}

	base, script, region := tag.Raw()
	if base.String() != "en" {
		return 0, false
	}
	if scriptCode := script.String(); scriptCode != "Zzzz" && scriptCode != "Latn" {
		return 0, false
	}
	order, ok := slashDateOrders[region.String()]
	return order, ok
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
		ambiguity:  ambiguityDateOrder,
	}
}
