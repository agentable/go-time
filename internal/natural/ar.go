package natural

import (
	"regexp"
	"strconv"
	"strings"
)

// Pre-compiled patterns for Arabic natural language expressions.
var (
	// reArFuture matches "بعد N unit" (in N units).
	// Group 1: number, Group 2: unit word.
	reArFuture = regexp.MustCompile(`^بعد\s+(\d+)\s+(\S+)$`)

	// reArPast matches "منذ N unit" (N units ago).
	// Group 1: number, Group 2: unit word.
	reArPast = regexp.MustCompile(`^منذ\s+(\d+)\s+(\S+)$`)

	// reArDual matches "بعد/منذ dualWord" where dualWord encodes "2 of unit".
	// Group 1: direction (بعد or منذ), Group 2: dual word.
	reArDual = regexp.MustCompile(`^(بعد|منذ)\s+(\S+)$`)
)

// arUnits maps Arabic unit words (singular and plural) to canonical names.
var arUnits = map[string]string{
	"ثانية":  "second",
	"ثواني":  "second",
	"ثوان":   "second",
	"دقيقة":  "minute",
	"دقائق":  "minute",
	"ساعة":   "hour",
	"ساعات":  "hour",
	"يوم":    "day",
	"أيام":   "day",
	"يوما":   "day",
	"أسبوع":  "week",
	"أسابيع": "week",
	"شهر":    "month",
	"أشهر":   "month",
	"شهور":   "month",
}

// arDual maps Arabic dual-form words to canonical unit names (count is always 2).
var arDual = map[string]string{
	"ساعتين":  "hour",
	"دقيقتين": "minute",
	"يومين":   "day",
	"أسبوعين": "week",
	"شهرين":   "month",
	"ثانيتين": "second",
}

type arParser struct{}

func init() {
	register(&arParser{})
}

func (p *arParser) canHandle(locale string) bool {
	return matchesLocalePrefix(locale, "ar")
}

func (p *arParser) parse(input string, ctx Context) (Result, bool) {
	input = strings.TrimSpace(input)
	loc := locForZone(ctx.ZoneID)
	base := midnightInLoc(ctx.RelativeTo, loc)

	switch input {
	case "اليوم":
		return dateResult(base, ctx.ZoneID), true
	case "غداً", "غدا":
		return dateResult(base.AddDate(0, 0, 1), ctx.ZoneID), true
	case "أمس":
		return dateResult(base.AddDate(0, 0, -1), ctx.ZoneID), true
	}

	if m := reArDual.FindStringSubmatch(input); m != nil {
		direction := m[1]
		word := m[2]
		if canonical, ok := arDual[word]; ok {
			n := int64(2)
			if direction == "منذ" {
				n = -n
			}
			return relativeUnitResult(canonical, n), true
		}
	}

	if m := reArFuture.FindStringSubmatch(input); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		if canonical, ok := arUnits[m[2]]; ok {
			return relativeUnitResult(canonical, n), true
		}
	}

	if m := reArPast.FindStringSubmatch(input); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		if canonical, ok := arUnits[m[2]]; ok {
			return relativeUnitResult(canonical, -n), true
		}
	}

	return Result{}, false
}
