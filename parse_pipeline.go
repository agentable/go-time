package gotime

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// RFC 3339 / ISO 8601 datetime with timezone offset
	reDateTime = regexp.MustCompile(
		`^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}(?::\d{2}(?:[.,]\d+)?)?)(Z|[+-]\d{2}:?\d{2}|[+-]\d{2})$`,
	)
	// ISO 8601 datetime without offset (local/naive datetime)
	reDateTimeNoOffset = regexp.MustCompile(
		`^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}(?::\d{2}(?:[.,]\d+)?)?)$`,
	)
	// Compact datetime: 20260327T130000 or 20260327T130000+0900
	reCompactDateTime = regexp.MustCompile(
		`^(\d{8})[T ](\d{2})(\d{2})?(\d{2})?(?:[.,](\d+))?(Z|[+-]\d{2}:?\d{2}|[+-]\d{2})?$`,
	)
	// ISO 8601 date: YYYY-MM-DD
	reDate = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	// Compact date: YYYYMMDD
	reCompactDate = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)
	// Year-month: YYYY-MM
	reYearMonth = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	// Ordinal date: YYYY-DDD or YYYYDDD
	reOrdinalDate = regexp.MustCompile(`^(\d{4})-?(\d{3})$`)
	// Week date: YYYY-Www-D or YYYY-Www
	reWeekDate = regexp.MustCompile(`^(\d{4})-W(\d{2})(?:-(\d))?$`)
	// ISO 8601 duration/period: Period date components may carry individual signs.
	// Group 1 is the optional leading sign; capture groups 2..8 are the components.
	reDuration = regexp.MustCompile(
		`^(-?)P(?:([+-]?\d+(?:[.,]\d+)?)Y)?(?:([+-]?\d+(?:[.,]\d+)?)M)?(?:([+-]?\d+(?:[.,]\d+)?)W)?(?:([+-]?\d+(?:[.,]\d+)?)D)?(?:T(?:(\d+(?:[.,]\d+)?)H)?(?:(\d+(?:[.,]\d+)?)M)?(?:(\d+(?:[.,]\d+)?)S)?)?$`,
	)
	// 24h time: HH:MM or HH:MM:SS
	reTime24 = regexp.MustCompile(`^(\d{1,2}):(\d{2})(?::(\d{2}))?$`)
	// 12h time: 3pm, 3:30pm, 3:30 PM
	reTime12 = regexp.MustCompile(`(?i)^(\d{1,2})(?::(\d{2}))?(?::(\d{2}))?\s*(am|pm)$`)
	// Ambiguous slash date: M/D/YYYY or D/M/YYYY
	reSlashDate = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})$`)
)

func parseWithConfig(input string, cfg *config) ParseResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return invalidResult(input, ErrEmptyInput, "input is empty",
			"Provide a date, time, or datetime string in ISO 8601 format")
	}

	if r, ok := parseDateTimeStage(input, cfg); ok {
		return r
	}
	if r, ok := parseIntervalDurationStage(input, cfg); ok {
		return r
	}
	if r, ok := parseDateStage(input, cfg); ok {
		return r
	}
	if r, ok := parseTimeStage(input, cfg); ok {
		return r
	}
	if r, ok := tryParseNatural(input, cfg); ok {
		return r
	}

	return invalidResult(input, ErrUnparseable,
		fmt.Sprintf("cannot parse %q as a date, time, or datetime", input),
		"Use ISO 8601 format (e.g. 2026-03-27, 2026-03-27T13:00:00+09:00, PT1H30M)")
}

func parseDateTimeStage(input string, cfg *config) (ParseResult, bool) {
	if r, ok := tryParseDateTime(input, cfg); ok {
		return r, true
	}
	if r, ok := tryParseDateTimeNoOffset(input, cfg); ok {
		return r, true
	}
	return tryParseCompactDateTime(input, cfg)
}

func parseIntervalDurationStage(input string, cfg *config) (ParseResult, bool) {
	if strings.Contains(input, "/") && !reSlashDate.MatchString(input) {
		return parseInterval(input, cfg), true
	}
	if strings.HasPrefix(input, "P") || strings.HasPrefix(input, "-P") {
		return parseDuration(input, cfg), true
	}
	return ParseResult{}, false
}

func parseDateStage(input string, cfg *config) (ParseResult, bool) {
	if r, ok := tryParseDate(input, cfg); ok {
		return r, true
	}
	m := reSlashDate.FindStringSubmatch(input)
	if m == nil {
		return ParseResult{}, false
	}
	return parseSlashDate(input, m, cfg), true
}

func parseTimeStage(input string, cfg *config) (ParseResult, bool) {
	if m := reTime24.FindStringSubmatch(input); m != nil {
		return parseTime24(input, m, cfg), true
	}
	if m := reTime12.FindStringSubmatch(input); m != nil {
		return parseTime12(input, m, cfg), true
	}
	return ParseResult{}, false
}
