package natural

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pre-compiled patterns for Korean natural language expressions.
var (
	// reKoRelativeDate matches simple relative day names.
	reKoRelativeDate = regexp.MustCompile(`^(오늘|내일|모레|어제|그제)$`)

	// reKoWeekRelative matches week-relative expressions like 다음 주 월요일.
	// \s* allows with or without spaces around 주.
	reKoWeekRelative = regexp.MustCompile(`^(다음\s*주|이번\s*주|지난\s*주)\s*(월요일?|화요일?|수요일?|목요일?|금요일?|토요일?|일요일?)$`)

	// reKoDateTime matches date phrase + optional space + 오전/오후 + hour + 시 + optional minutes.
	// Group 1: date phrase, Group 2: AM/PM, Group 3: hour, Group 4: minute digits (without 분)
	reKoDateTime = regexp.MustCompile(
		`^(오늘|내일|모레|어제|그제|(?:다음\s*주|이번\s*주|지난\s*주)\s*(?:월요일?|화요일?|수요일?|목요일?|금요일?|토요일?|일요일?))` +
			`\s*(오전|오후)?\s*(\d{1,2})시(?:\s*(\d{1,2})분)?$`,
	)

	// reKoRelativeDur matches duration expressions like 2시간 후 / 30분 전.
	// Group 1: number, Group 2: unit, Group 3: direction (후=future, 전=past)
	reKoRelativeDur = regexp.MustCompile(`^(\d+)\s*(초|분|시간|일|주|달|년)\s*(후|전)$`)
)

// koUnits maps Korean unit words to canonical English names for canonicalNanos.
var koUnits = map[string]string{
	"초":  "second",
	"분":  "minute",
	"시간": "hour",
	"일":  "day",
	"주":  "week",
	"달":  "month",
	"년":  "year",
}

type koParser struct{}

func init() {
	register(&koParser{})
}

func (p *koParser) canHandle(locale string) bool {
	return matchesLocalePrefix(locale, "ko")
}

func (p *koParser) parse(input string, ctx Context) (Result, bool) {
	loc := locForZone(ctx.ZoneID)
	base := midnightInLoc(ctx.RelativeTo, loc)

	if m := reKoRelativeDate.FindStringSubmatch(input); m != nil {
		return dateResult(koRelativeDateOffset(base, m[1]), ctx.ZoneID), true
	}

	if m := reKoWeekRelative.FindStringSubmatch(input); m != nil {
		t := koWeekDate(base, m[1], m[2])
		return dateResult(t, ctx.ZoneID), true
	}

	if m := reKoDateTime.FindStringSubmatch(input); m != nil {
		datePart := m[1]
		ampm := m[2]
		hourStr := m[3]
		minStr := m[4]

		var dateBase time.Time
		if reKoRelativeDate.MatchString(datePart) {
			dateBase = koRelativeDateOffset(base, datePart)
		} else if mm := reKoWeekRelative.FindStringSubmatch(datePart); mm != nil {
			dateBase = koWeekDate(base, mm[1], mm[2])
		} else {
			return Result{}, false
		}

		hour, _ := strconv.Atoi(hourStr)
		hour = applyHourPeriod(hour, ampm, "오전", "오후")
		min := 0
		if minStr != "" {
			min, _ = strconv.Atoi(minStr)
		}
		return datetimeAt(dateBase, hour, min, loc, ctx.ZoneID), true
	}

	if m := reKoRelativeDur.FindStringSubmatch(input); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		if canonical, ok := koUnits[m[2]]; ok {
			if m[3] == "전" {
				n = -n
			}
			return relativeUnitResult(canonical, n), true
		}
	}

	return Result{}, false
}

func koRelativeDateOffset(base time.Time, keyword string) time.Time {
	switch keyword {
	case "오늘":
		return base
	case "내일":
		return base.AddDate(0, 0, 1)
	case "어제":
		return base.AddDate(0, 0, -1)
	case "모레":
		return base.AddDate(0, 0, 2)
	case "그제":
		return base.AddDate(0, 0, -2)
	default:
		return base
	}
}

func koWeekDate(base time.Time, modifier, wdStr string) time.Time {
	wd := koWeekdayStr(wdStr)
	modNorm := strings.ReplaceAll(modifier, " ", "")
	switch modNorm {
	case "다음주":
		return nextWeekday(base, wd)
	case "이번주":
		return thisWeekday(base, wd)
	case "지난주":
		return lastWeekday(base, wd)
	default:
		return base
	}
}

func koWeekdayStr(s string) time.Weekday {
	switch {
	case strings.HasPrefix(s, "월요"):
		return time.Monday
	case strings.HasPrefix(s, "화요"):
		return time.Tuesday
	case strings.HasPrefix(s, "수요"):
		return time.Wednesday
	case strings.HasPrefix(s, "목요"):
		return time.Thursday
	case strings.HasPrefix(s, "금요"):
		return time.Friday
	case strings.HasPrefix(s, "토요"):
		return time.Saturday
	case strings.HasPrefix(s, "일요"):
		return time.Sunday
	default:
		return time.Monday
	}
}
