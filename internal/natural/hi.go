package natural

import (
	"regexp"
	"strconv"
	"strings"
)

// Pre-compiled patterns for Hindi natural language expressions.
var (
	// reHiFuture matches "N unit में" or "N unit बाद" (in N units).
	// Group 1: number, Group 2: unit word.
	reHiFuture = regexp.MustCompile(`^(\d+)\s+(\S+)\s+(में|बाद)$`)

	// reHiPast matches "N unit पहले" (N units ago).
	// Group 1: number, Group 2: unit word.
	reHiPast = regexp.MustCompile(`^(\d+)\s+(\S+)\s+पहले$`)
)

// hiUnits maps Hindi unit words to canonical names.
var hiUnits = map[string]string{
	"सेकंड":  "second",
	"मिनट":   "minute",
	"घंटे":   "hour",
	"घंटा":   "hour",
	"दिन":    "day",
	"सप्ताह": "week",
	"हफ्ते":  "week",
	"महीने":  "month",
	"महीना":  "month",
}

type hiParser struct{}

func init() {
	register(&hiParser{})
}

func (p *hiParser) canHandle(locale string) bool {
	return matchesLocalePrefix(locale, "hi")
}

func (p *hiParser) parse(input string, ctx Context) (Result, bool) {
	input = strings.TrimSpace(input)
	loc := locForZone(ctx.ZoneID)
	base := midnightInLoc(ctx.RelativeTo, loc)

	switch input {
	case "आज":
		return dateResult(base, ctx.ZoneID), true
	case "कल":
		return dateResult(base.AddDate(0, 0, 1), ctx.ZoneID), true
	case "परसों":
		return dateResult(base.AddDate(0, 0, 2), ctx.ZoneID), true
	}

	if m := reHiFuture.FindStringSubmatch(input); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		if canonical, ok := hiUnits[m[2]]; ok {
			return relativeUnitResult(canonical, n), true
		}
	}

	if m := reHiPast.FindStringSubmatch(input); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		if canonical, ok := hiUnits[m[2]]; ok {
			return relativeUnitResult(canonical, -n), true
		}
	}

	return Result{}, false
}
