package gotime

import (
	"fmt"
	"strings"
	"time"
)

func tryParseDate(input string, cfg *config) (ParseResult, bool) {
	if m := reDate.FindStringSubmatch(input); m != nil {
		year, mon, day := atoi(m[1]), atoi(m[2]), atoi(m[3])
		if msg := validateDateComponents(year, mon, day); msg != "" {
			return invalidResult(input, ErrInvalidDate, msg,
				"Provide a valid calendar date, e.g. 2026-03-27"), true
		}
		return resolvedDateResult(input, dateFromComponents(year, time.Month(mon), day), cfg), true
	}
	if m := reCompactDate.FindStringSubmatch(input); m != nil {
		year, mon, day := atoi(m[1]), atoi(m[2]), atoi(m[3])
		if msg := validateDateComponents(year, mon, day); msg != "" {
			return invalidResult(input, ErrInvalidDate, msg,
				"Provide a valid calendar date, e.g. 20260327"), true
		}
		return resolvedDateResult(input, dateFromComponents(year, time.Month(mon), day), cfg), true
	}
	if m := reYearMonth.FindStringSubmatch(input); m != nil {
		year, mon := atoi(m[1]), atoi(m[2])
		if mon < 1 || mon > 12 {
			return invalidResult(input, ErrInvalidDate,
				fmt.Sprintf("invalid month %d", mon),
				"Month must be between 1 and 12"), true
		}
		return resolvedDateResult(input, dateFromComponents(year, time.Month(mon), 1), cfg), true
	}
	if m := reOrdinalDate.FindStringSubmatch(input); m != nil {
		year, doy := atoi(m[1]), atoi(m[2])
		if doy < 1 || doy > 366 {
			return invalidResult(input, ErrInvalidDate,
				fmt.Sprintf("invalid ordinal day %d", doy),
				"Ordinal day must be between 1 and 366"), true
		}
		t := time.Date(year, time.January, doy, 0, 0, 0, 0, time.UTC)
		if t.Year() != year {
			return invalidResult(input, ErrInvalidDate,
				fmt.Sprintf("ordinal day %d does not exist in year %d", doy, year),
				"Use 366 only for leap years; otherwise use an ordinal day between 1 and 365"), true
		}
		return resolvedDateResult(input, DateFromTime(t), cfg), true
	}
	if m := reWeekDate.FindStringSubmatch(input); m != nil {
		year, week := atoi(m[1]), atoi(m[2])
		dayOfWeek := 1
		if m[3] != "" {
			dayOfWeek = atoi(m[3])
		}
		if week < 1 || week > 53 {
			return invalidResult(input, ErrInvalidDate,
				fmt.Sprintf("invalid ISO week %d", week),
				"ISO week must be between 1 and 53"), true
		}
		if dayOfWeek < 1 || dayOfWeek > 7 {
			return invalidResult(input, ErrInvalidDate,
				fmt.Sprintf("invalid day of week %d", dayOfWeek),
				"Day of week must be between 1 (Monday) and 7 (Sunday)"), true
		}
		return resolvedDateResult(input, DateFromTime(isoWeekDate(year, week, dayOfWeek)), cfg), true
	}
	return ParseResult{}, false
}

func resolvedDateResult(input string, d Date, cfg *config) ParseResult {
	r := resolvedResult(input, KindDate, cfg)
	r.date = d
	return r
}

func isoWeekDate(year, week, weekday int) time.Time {
	// Jan 4 is always in week 1 of the ISO year
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	// Monday of week 1
	mon1 := jan4.AddDate(0, 0, -int(jan4.Weekday()-time.Monday+7)%7)
	// Advance to requested week and weekday
	return mon1.AddDate(0, 0, (week-1)*7+(weekday-1))
}

func parseTime24(input string, m []string, cfg *config) ParseResult {
	h, min := atoi(m[1]), atoi(m[2])
	sec := 0
	if m[3] != "" {
		sec = atoi(m[3])
	}
	if h > 23 || min > 59 || sec > 59 {
		return invalidResult(input, ErrInvalidTime,
			fmt.Sprintf("invalid time %s", input),
			"Hours must be 0-23, minutes and seconds 0-59")
	}
	r := resolvedResult(input, KindTime, cfg)
	r.timeVal = timeFromComponents(h, min, sec, 0)
	return r
}

func parseTime12(input string, m []string, cfg *config) ParseResult {
	h, ampm := atoi(m[1]), strings.ToLower(m[4])
	min := 0
	if m[2] != "" {
		min = atoi(m[2])
	}
	sec := 0
	if m[3] != "" {
		sec = atoi(m[3])
	}
	if h < 1 || h > 12 || min > 59 || sec > 59 {
		return invalidResult(input, ErrInvalidTime,
			fmt.Sprintf("invalid 12-hour time %s", input),
			"12-hour clock hours must be 1-12")
	}
	if ampm == "am" && h == 12 {
		h = 0
	}
	if ampm == "pm" && h != 12 {
		h += 12
	}
	r := resolvedResult(input, KindTime, cfg)
	r.timeVal = timeFromComponents(h, min, sec, 0)
	return r
}
