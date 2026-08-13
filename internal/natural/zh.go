package natural

import (
	"regexp"
	"strconv"
	"time"
)

// Pre-compiled patterns for Chinese natural language expressions.
var (
	// reZhRelativeDate matches simple relative day names.
	reZhRelativeDate = regexp.MustCompile(`^(今天|明天|后天|後天|昨天|前天)$`)

	// reZhWeekRelative matches week-relative expressions like 下周五 / 上週三.
	reZhWeekRelative = regexp.MustCompile(`^(下|本|上|这|這)(周|週)(一|二|三|四|五|六|日|天)$`)

	// reZhMonthRelative matches month-relative expressions.
	reZhMonthRelative = regexp.MustCompile(`^(下个月|下個月|本月底|上个月|上個月)$`)

	// reZhDateTime matches date + optional time-word + hour + 点/點 + optional minutes.
	// Group 1: date phrase, Group 2: time word, Group 3: hour, Group 4: half/min suffix, Group 5: digit minutes
	reZhDateTime = regexp.MustCompile(
		`^(今天|明天|后天|後天|昨天|前天|(?:[下本上这這])(?:周|週)[一二三四五六日天])` +
			`(早上|上午|下午|晚上)?` +
			`(十[一二]?|[一二三四五六七八九]|\d{1,2})` +
			`[点點]` +
			`(半|(\d{1,2})分)?$`,
	)

	// reZhRelativeDur matches duration expressions like 两小时后 / 30分钟前.
	// Group 1: number (Arabic or Chinese), Group 2: unit, Group 3: direction (后/後=future, 前=past)
	reZhRelativeDur = regexp.MustCompile(
		`^(\d+|两|兩|[零〇一二两兩三四五六七八九十百千]+)(小时|小時|分钟|分鐘|天|周|週|个月|個月)(后|後|前)$`,
	)
)

type zhParser struct{}

func init() {
	register(&zhParser{})
}

func (p *zhParser) canHandle(locale string) bool {
	return matchesLocalePrefix(locale, "zh-Hans") || matchesLocalePrefix(locale, "zh-Hant")
}

func (p *zhParser) parse(input string, ctx Context) (Result, bool) {
	base := midnight(ctx.RelativeTo)

	if m := reZhRelativeDate.FindStringSubmatch(input); m != nil {
		return dateResult(zhRelativeDateOffset(base, m[1])), true
	}

	if m := reZhWeekRelative.FindStringSubmatch(input); m != nil {
		t := zhWeekDate(base, m[1], m[3])
		return dateResult(t), true
	}

	if m := reZhMonthRelative.FindStringSubmatch(input); m != nil {
		t := zhMonthDate(base, m[1])
		return dateResult(t), true
	}

	if m := reZhDateTime.FindStringSubmatch(input); m != nil {
		datePart := m[1]
		timeWord := m[2]
		hourStr := m[3]
		halfOrMin := m[4]
		digitMin := m[5]

		var dateBase time.Time
		if reZhRelativeDate.MatchString(datePart) {
			dateBase = zhRelativeDateOffset(base, datePart)
		} else if mm := reZhWeekRelative.FindStringSubmatch(datePart); mm != nil {
			dateBase = zhWeekDate(base, mm[1], mm[3])
		} else {
			return Result{}, false
		}

		hour := zhHourToInt(hourStr)
		hour = zhApplyTimeWord(hour, timeWord)
		min := zhMinutes(halfOrMin, digitMin)
		return datetimeAt(dateBase, hour, min), true
	}

	if m := reZhRelativeDur.FindStringSubmatch(input); m != nil {
		if r, ok := zhRelativeUnitResult(m[1], m[2], m[3]); ok {
			return r, true
		}
		return Result{}, false
	}

	return Result{}, false
}

// zhRelativeDateOffset maps a Chinese relative-day keyword to midnight time.
func zhRelativeDateOffset(base time.Time, keyword string) time.Time {
	switch keyword {
	case "今天":
		return base
	case "明天":
		return base.AddDate(0, 0, 1)
	case "昨天":
		return base.AddDate(0, 0, -1)
	case "后天", "後天":
		return base.AddDate(0, 0, 2)
	case "前天":
		return base.AddDate(0, 0, -2)
	default:
		return base
	}
}

// zhWeekDate computes the target weekday from week-modifier + weekday character.
func zhWeekDate(base time.Time, modifier, wdChar string) time.Time {
	wd := zhWeekdayChar(wdChar)
	switch modifier {
	case "下": // next week's weekday
		return nextWeekday(base, wd)
	case "本", "这", "這": // this week's weekday
		return thisWeekday(base, wd)
	case "上": // last week's weekday
		return lastWeekday(base, wd)
	default:
		return base
	}
}

// zhWeekdayChar maps Chinese weekday character to time.Weekday.
func zhWeekdayChar(c string) time.Weekday {
	switch c {
	case "一":
		return time.Monday
	case "二":
		return time.Tuesday
	case "三":
		return time.Wednesday
	case "四":
		return time.Thursday
	case "五":
		return time.Friday
	case "六":
		return time.Saturday
	case "日", "天":
		return time.Sunday
	default:
		return time.Monday
	}
}

// zhMonthDate computes the target date from a month-relative keyword.
func zhMonthDate(base time.Time, keyword string) time.Time {
	switch keyword {
	case "下个月", "下個月":
		// First day of next month
		return time.Date(base.Year(), base.Month()+1, 1, 0, 0, 0, 0, base.Location())
	case "本月底":
		// Last day of current month = first day of next month - 1 day
		return time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, base.Location())
	case "上个月", "上個月":
		// First day of previous month
		return time.Date(base.Year(), base.Month()-1, 1, 0, 0, 0, 0, base.Location())
	default:
		return base
	}
}

// zhHourToInt converts a Chinese or Arabic hour string to 24h integer (raw, before AM/PM).
func zhHourToInt(s string) int {
	switch s {
	case "十一":
		return 11
	case "十二":
		return 12
	default:
		n, _ := zhNumericValue(s)
		return int(n)
	}
}

// zhApplyTimeWord converts a raw 1-12 hour to 0-23 using the Chinese time-of-day word.
func zhApplyTimeWord(hour int, timeWord string) int {
	switch timeWord {
	case "早上", "上午":
		// Morning: 1-11 stays as-is; 12 → 0 (midnight edge, unlikely)
		if hour == 12 {
			return 0
		}
		return hour
	case "下午":
		// Afternoon: add 12 if hour < 12
		if hour < 12 {
			return hour + 12
		}
		return hour
	case "晚上":
		// Evening: add 12 if hour < 12; 12 stays 12
		if hour < 12 {
			return hour + 12
		}
		return hour
	default:
		// No time word — return as-is (treat as 24h already, or 12h ambiguous)
		return hour
	}
}

// zhMinutes extracts minutes from the half/min suffix.
func zhMinutes(halfOrMin, digitMin string) int {
	if halfOrMin == "半" {
		return 30
	}
	if digitMin != "" {
		n, _ := strconv.Atoi(digitMin)
		return n
	}
	return 0
}

// zhRelativeUnitResult converts a matched relative unit into a Duration or Period.
func zhRelativeUnitResult(numStr, unit, dir string) (Result, bool) {
	n, ok := zhNumericValue(numStr)
	if !ok {
		if isASCIIDigits(numStr) {
			return invalidResult(ErrorOverflow, "relative count overflows int64", "use a smaller count"), true
		}
		return Result{}, false
	}
	var canonical string
	switch unit {
	case "小时", "小時":
		canonical = "hour"
	case "分钟", "分鐘":
		canonical = "minute"
	case "天":
		canonical = "day"
	case "周", "週":
		canonical = "week"
	case "个月", "個月":
		canonical = "month"
	default:
		canonical = "second"
	}
	if dir == "前" {
		n = -n
	}
	return relativeUnitResult(canonical, n), true
}

func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// zhNumericValue converts a Chinese or Arabic number string to int64.
func zhNumericValue(s string) (int64, bool) {
	if s == "两" || s == "兩" {
		return 2, true
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, true
	}
	return zhChineseNumeral(s)
}

func zhChineseNumeral(s string) (int64, bool) {
	runes := []rune(s)
	digits := map[rune]int64{'一': 1, '二': 2, '两': 2, '兩': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	units := map[rune]int64{'十': 10, '百': 100, '千': 1000}
	var total int64
	previousUnit := int64(10000)
	hasPreviousUnit := false
	hasZeroPlaceholder := false
	for i := 0; i < len(runes); {
		if runes[i] == '零' || runes[i] == '〇' {
			if !hasPreviousUnit || hasZeroPlaceholder || i+1 == len(runes) {
				return 0, false
			}
			hasZeroPlaceholder = true
			i++
			continue
		}

		coefficient := int64(1)
		unit := int64(0)
		if runes[i] == '十' {
			if hasPreviousUnit {
				return 0, false
			}
			unit = 10
			i++
		} else {
			digit, ok := digits[runes[i]]
			if !ok {
				return 0, false
			}
			coefficient = digit
			twoVariant := runes[i] == '两' || runes[i] == '兩'
			i++
			if i == len(runes) {
				requiresZeroPlaceholder := previousUnit > 10
				if hasPreviousUnit && (twoVariant || requiresZeroPlaceholder != hasZeroPlaceholder) {
					return 0, false
				}
				return total + coefficient, true
			}
			unit = units[runes[i]]
			if unit == 0 || (twoVariant && unit < 100) {
				return 0, false
			}
			i++
		}

		if unit >= previousUnit {
			return 0, false
		}
		requiresZeroPlaceholder := previousUnit/unit > 10
		if hasPreviousUnit && requiresZeroPlaceholder != hasZeroPlaceholder {
			return 0, false
		}
		total += coefficient * unit
		previousUnit = unit
		hasPreviousUnit = true
		hasZeroPlaceholder = false
	}
	return total, total != 0
}
