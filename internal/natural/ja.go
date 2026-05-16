package natural

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pre-compiled patterns for Japanese natural language expressions.
var (
	// reJaRelativeDate matches simple relative day names.
	reJaRelativeDate = regexp.MustCompile(`^(今日|明日|明後日|昨日|一昨日)$`)

	// reJaWeekRelative matches week-relative expressions like 来週金曜日 / 先週水曜.
	// Group 1: week modifier, Group 2: weekday (with optional 日 suffix)
	reJaWeekRelative = regexp.MustCompile(`^(来週|今週|先週)(月曜日?|火曜日?|水曜日?|木曜日?|金曜日?|土曜日?|日曜日?)$`)

	// reJaDateTime matches date phrase + optional の + 午前/午後 + hour + 時 + optional minutes.
	// Group 1: date phrase, Group 2: AM/PM (午前/午後), Group 3: hour, Group 4: minute digits
	reJaDateTime = regexp.MustCompile(
		`^(今日|明日|明後日|昨日|一昨日|(?:来週|今週|先週)(?:月曜日?|火曜日?|水曜日?|木曜日?|金曜日?|土曜日?|日曜日?))` +
			`の?` +
			`(午前|午後)?` +
			`(\d{1,2})時` +
			`(\d{1,2}分)?$`,
	)

	// reJaRelativeDur matches duration expressions like 2時間後 / 30分前.
	// Group 1: number, Group 2: unit, Group 3: direction (後=future, 前=past)
	reJaRelativeDur = regexp.MustCompile(`^(\d+)(秒|分|時間|日|週|ヶ月|年)(後|前)$`)
)

// jaUnits maps Japanese unit words to canonical English names for canonicalNanos.
var jaUnits = map[string]string{
	"秒":  "second",
	"分":  "minute",
	"時間": "hour",
	"日":  "day",
	"週":  "week",
	"ヶ月": "month",
	"年":  "year",
}

type jaParser struct{}

func init() {
	register(&jaParser{})
}

func (p *jaParser) canHandle(locale string) bool {
	return locale == "ja"
}

func (p *jaParser) parse(input string, ctx Context) (Result, bool) {
	loc := locForZone(ctx.ZoneID)
	base := midnightInLoc(ctx.RelativeTo, loc)

	if m := reJaRelativeDate.FindStringSubmatch(input); m != nil {
		return dateResult(jaRelativeDateOffset(base, m[1]), ctx.ZoneID), true
	}

	if m := reJaWeekRelative.FindStringSubmatch(input); m != nil {
		t := jaWeekDate(base, m[1], m[2])
		return dateResult(t, ctx.ZoneID), true
	}

	if m := reJaDateTime.FindStringSubmatch(input); m != nil {
		datePart := m[1]
		ampm := m[2]
		hourStr := m[3]
		minPart := m[4]

		var dateBase time.Time
		if reJaRelativeDate.MatchString(datePart) {
			dateBase = jaRelativeDateOffset(base, datePart)
		} else if mm := reJaWeekRelative.FindStringSubmatch(datePart); mm != nil {
			dateBase = jaWeekDate(base, mm[1], mm[2])
		} else {
			return Result{}, false
		}

		hour, _ := strconv.Atoi(hourStr)
		hour = applyHourPeriod(hour, ampm, "午前", "午後")
		min := 0
		if minPart != "" {
			// strip 分 suffix
			min, _ = strconv.Atoi(minPart[:len(minPart)-3]) // 3 bytes for "分" (UTF-8)
		}
		return datetimeAt(dateBase, hour, min, loc, ctx.ZoneID), true
	}

	if m := reJaRelativeDur.FindStringSubmatch(input); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		if canonical, ok := jaUnits[m[2]]; ok {
			if m[3] == "前" {
				n = -n
			}
			return relativeUnitResult(canonical, n), true
		}
	}

	return Result{}, false
}

func jaRelativeDateOffset(base time.Time, keyword string) time.Time {
	switch keyword {
	case "今日":
		return base
	case "明日":
		return base.AddDate(0, 0, 1)
	case "昨日":
		return base.AddDate(0, 0, -1)
	case "明後日":
		return base.AddDate(0, 0, 2)
	case "一昨日":
		return base.AddDate(0, 0, -2)
	default:
		return base
	}
}

func jaWeekDate(base time.Time, modifier, wdStr string) time.Time {
	wd := jaWeekdayStr(wdStr)
	switch modifier {
	case "来週":
		return nextWeekday(base, wd)
	case "今週":
		return thisWeekday(base, wd)
	case "先週":
		return lastWeekday(base, wd)
	default:
		return base
	}
}

func jaWeekdayStr(s string) time.Weekday {
	switch {
	case strings.HasPrefix(s, "月曜"):
		return time.Monday
	case strings.HasPrefix(s, "火曜"):
		return time.Tuesday
	case strings.HasPrefix(s, "水曜"):
		return time.Wednesday
	case strings.HasPrefix(s, "木曜"):
		return time.Thursday
	case strings.HasPrefix(s, "金曜"):
		return time.Friday
	case strings.HasPrefix(s, "土曜"):
		return time.Saturday
	case strings.HasPrefix(s, "日曜"):
		return time.Sunday
	default:
		return time.Monday
	}
}
