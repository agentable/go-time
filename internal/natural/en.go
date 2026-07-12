package natural

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pre-compiled patterns for English natural language expressions.
var (
	// reEnRelativeDate matches simple relative day names (case-insensitive via (?i)).
	reEnRelativeDate = regexp.MustCompile(`(?i)^(today|tomorrow|yesterday|day after tomorrow|day before yesterday)$`)

	// reEnWeekRelative matches "next/this/last <weekday>" expressions.
	reEnWeekRelative = regexp.MustCompile(`(?i)^(next|this|last)\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)$`)

	// reEnDateTime matches "<date_expr> at <hour>[:<min>] <am|pm>".
	// Group 1: date phrase, Group 2: hour, Group 3: optional minutes, Group 4: am/pm
	reEnDateTime = regexp.MustCompile(
		`(?i)^(today|tomorrow|yesterday|day after tomorrow|day before yesterday|` +
			`(?:next|this|last)\s+(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday))` +
			`\s+at\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)$`,
	)

	// reEnRelativeDurFuture matches "in N unit(s)".
	reEnRelativeDurFuture = regexp.MustCompile(`(?i)^in\s+(\d+)\s+(second|minute|hour|day|week|month)s?$`)

	// reEnRelativeDurPast matches "N unit(s) ago".
	reEnRelativeDurPast = regexp.MustCompile(`(?i)^(\d+)\s+(second|minute|hour|day|week|month)s?\s+ago$`)
)

type enParser struct{}

func init() {
	register(&enParser{})
}

func (p *enParser) canHandle(locale string) bool {
	return matchesLocalePrefix(locale, "en")
}

func (p *enParser) parse(input string, ctx Context) (Result, bool) {
	lower := strings.ToLower(strings.TrimSpace(input))
	base := midnight(ctx.RelativeTo)

	if m := reEnRelativeDate.FindStringSubmatch(lower); m != nil {
		return dateResult(enRelativeDateOffset(base, m[1])), true
	}

	if m := reEnWeekRelative.FindStringSubmatch(lower); m != nil {
		t := enWeekDate(base, m[1], m[2])
		return dateResult(t), true
	}

	if m := reEnDateTime.FindStringSubmatch(lower); m != nil {
		datePart := m[1]
		hourStr := m[2]
		minStr := m[3]
		ampm := strings.ToLower(m[4])

		var dateBase time.Time
		if reEnRelativeDate.MatchString(datePart) {
			dateBase = enRelativeDateOffset(base, datePart)
		} else if mm := reEnWeekRelative.FindStringSubmatch(datePart); mm != nil {
			dateBase = enWeekDate(base, mm[1], mm[2])
		} else {
			return Result{}, false
		}

		hour, _ := strconv.Atoi(hourStr)
		min := 0
		if minStr != "" {
			min, _ = strconv.Atoi(minStr)
		}
		hour = applyHourPeriod(hour, ampm, "am", "pm")
		return datetimeAt(dateBase, hour, min), true
	}

	if m := reEnRelativeDurFuture.FindStringSubmatch(lower); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		return relativeUnitResult(m[2], n), true
	}

	if m := reEnRelativeDurPast.FindStringSubmatch(lower); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		return relativeUnitResult(m[2], -n), true
	}

	return Result{}, false
}

// enRelativeDateOffset maps an English relative-day keyword to midnight time.
func enRelativeDateOffset(base time.Time, keyword string) time.Time {
	switch keyword {
	case "today":
		return base
	case "tomorrow":
		return base.AddDate(0, 0, 1)
	case "yesterday":
		return base.AddDate(0, 0, -1)
	case "day after tomorrow":
		return base.AddDate(0, 0, 2)
	case "day before yesterday":
		return base.AddDate(0, 0, -2)
	default:
		return base
	}
}

// enWeekDate computes the target weekday from modifier + weekday name.
func enWeekDate(base time.Time, modifier, weekdayName string) time.Time {
	wd := enWeekdayName(weekdayName)
	switch modifier {
	case "next":
		return nextWeekday(base, wd)
	case "this":
		return thisWeekday(base, wd)
	case "last":
		return lastWeekday(base, wd)
	default:
		return base
	}
}

// enWeekdayName maps an English weekday name (lower) to time.Weekday.
func enWeekdayName(name string) time.Weekday {
	switch name {
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	case "sunday":
		return time.Sunday
	default:
		return time.Monday
	}
}
