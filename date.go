package gotime

import (
	"cmp"
	"fmt"
	"time"

	"github.com/go-json-experiment/json"
)

// Date is a calendar date without time or timezone information.
type Date struct {
	year  int
	month time.Month
	day   int
}

// NewDate creates a Date from year, month, and day components.
func NewDate(year int, month time.Month, day int) (Date, error) {
	if msg := validateDateComponents(year, int(month), day); msg != "" {
		return Date{}, newTimeError(
			ErrInvalidDate,
			msg,
			fmt.Sprintf("year=%d month=%d day=%d", year, month, day),
			"provide a real calendar date, e.g. 2026-03-27",
		)
	}
	return dateFromComponents(year, month, day), nil
}

func dateFromComponents(year int, month time.Month, day int) Date {
	return Date{year: year, month: month, day: day}
}

// DateFromTime extracts the date from a time.Time using its location.
func DateFromTime(t time.Time) Date {
	return dateFromComponents(t.Year(), t.Month(), t.Day())
}

// Year returns the year component.
func (d Date) Year() int { return d.year }

// Month returns the month component.
func (d Date) Month() time.Month { return d.month }

// Day returns the day component.
func (d Date) Day() int { return d.day }

// IsZero reports whether d is the zero value.
func (d Date) IsZero() bool { return d.year == 0 && d.month == 0 && d.day == 0 }

// Equal reports whether d and other represent the same calendar date.
func (d Date) Equal(other Date) bool {
	return d.year == other.year && d.month == other.month && d.day == other.day
}

// Before reports whether d is before other.
func (d Date) Before(other Date) bool { return d.Compare(other) < 0 }

// After reports whether d is after other.
func (d Date) After(other Date) bool { return d.Compare(other) > 0 }

// Add returns a new Date advanced by p (calendar arithmetic).
// Month/year arithmetic applies end-of-month clamping. To move back,
// pass a negated Period: d.Add(gotime.Days(3).Negate()).
func (d Date) Add(p Period) Date {
	return DateFromTime(clampAddDate(d.toTime(), int(p.Years), int(p.Months), int(p.Days)))
}

// DaysUntil returns the signed number of calendar days from d to other.
func (d Date) DaysUntil(other Date) int {
	return other.dayNumber() - d.dayNumber()
}

// PeriodUntil returns the signed greedy calendar period from d to other.
// It prefers years, then months, then days; use DaysUntil for exact day counts.
func (d Date) PeriodUntil(other Date) Period {
	if d.Equal(other) {
		return Period{}
	}

	start, end := d, other
	neg := other.Before(d)
	if neg {
		start, end = other, d
	}
	years := end.year - start.year
	if start.Add(Period{Years: int32(years)}).After(end) { //nolint:gosec // normalized calendar year delta fits Period's int32 fields
		years--
	}
	months := 0
	for start.Add(Period{Years: int32(years), Months: int32(months + 1)}).Compare(end) <= 0 { //nolint:gosec // normalized calendar month delta is bounded by 0..11
		months++
	}
	base := start.Add(Period{Years: int32(years), Months: int32(months)}) //nolint:gosec // calendar deltas fit in int32
	days := base.DaysUntil(end)
	period := Period{Years: int32(years), Months: int32(months), Days: int32(days)} //nolint:gosec // calendar deltas fit in int32
	if neg {
		return period.Negate()
	}
	return period
}

func (d Date) dayNumber() int {
	year := d.year
	month := int(d.month)
	if month <= 2 {
		year--
	}

	era := divFloor(year, 400)
	yearOfEra := year - era*400
	monthPrime := month - 3
	if monthPrime < 0 {
		monthPrime += 12
	}
	dayOfYear := (153*monthPrime+2)/5 + d.day - 1
	dayOfEra := yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear
	return era*146097 + dayOfEra
}

func divFloor(n, d int) int {
	q := n / d
	r := n % d
	if r != 0 && (r < 0) != (d < 0) {
		q--
	}
	return q
}

// Compare returns -1 if d is before other, 0 if equal, 1 if d is after other.
func (d Date) Compare(other Date) int {
	return cmp.Or(
		cmp.Compare(d.year, other.year),
		cmp.Compare(d.month, other.month),
		cmp.Compare(d.day, other.day),
	)
}

// toTime returns the Date as a time.Time at midnight UTC.
func (d Date) toTime() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}

// Std returns the Date as a time.Time at 00:00:00 in zone z. Use this to
// bridge a Date to any API expecting a stdlib time.Time.
func (d Date) Std(z Zone) time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, z.Location())
}

// Weekday returns the day of the week (Sunday = 0 through Saturday = 6).
func (d Date) Weekday() time.Weekday { return d.toTime().Weekday() }

// ISOWeek returns the ISO 8601 year and week number.
func (d Date) ISOWeek() (year, week int) { return d.toTime().ISOWeek() }

// YearDay returns the day of the year (1–366).
func (d Date) YearDay() int { return d.toTime().YearDay() }

// DaysInMonth returns the number of days in the date's month.
func (d Date) DaysInMonth() int { return daysInMonth(d.year, d.month) }

// IsLeapYear reports whether the date's year is a leap year.
func (d Date) IsLeapYear() bool {
	return d.year%4 == 0 && (d.year%100 != 0 || d.year%400 == 0)
}

// String returns the ISO 8601 date string "YYYY-MM-DD".
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.year, int(d.month), d.day)
}

// MarshalJSON encodes d as {"kind":"date","value":"YYYY-MM-DD","calendar":"iso8601"}.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind     string `json:"kind"`
		Value    string `json:"value"`
		Calendar string `json:"calendar"`
	}{Kind: "date", Value: d.String(), Calendar: "iso8601"})
}

// UnmarshalJSON decodes d from {"kind":"date","value":"YYYY-MM-DD"[,"calendar":"..."]}.
func (d *Date) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind     string `json:"kind"`
		Value    string `json:"value"`
		Calendar string `json:"calendar"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("date", wire.Kind, "date"); err != nil {
		return err
	}
	if err := requireJSONCalendar("date", wire.Calendar); err != nil {
		return err
	}
	if err := requireJSONString("date", "value", wire.Value); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", wire.Value)
	if err != nil {
		return fmt.Errorf("gotime: invalid date value %q: %w", wire.Value, err)
	}
	date, err := NewDate(t.Year(), t.Month(), t.Day())
	if err != nil {
		return err
	}
	*d = date
	return nil
}

// validateDateComponents returns a non-empty error message if the date is invalid,
// or an empty string if valid.
func validateDateComponents(year, month, day int) string {
	if year < 0 || year > 9999 {
		return fmt.Sprintf("invalid year %d: must be 0-9999", year)
	}
	if month < 1 || month > 12 {
		return fmt.Sprintf("invalid month %d: must be 1-12", month)
	}
	if day < 1 || day > 31 {
		return fmt.Sprintf("invalid day %d: must be 1-31", day)
	}
	// More precise validation via time.Date normalization.
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Month() != time.Month(month) || t.Day() != day {
		return fmt.Sprintf("date %04d-%02d-%02d does not exist", year, month, day)
	}
	return ""
}
