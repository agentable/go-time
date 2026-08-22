package gotime

import (
	"cmp"
	"fmt"
	"time"

	"encoding/json/v2"
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
// It returns ErrOverflow when the projected year is outside 0000..9999.
func DateFromTime(t time.Time) (Date, error) {
	if year := t.Year(); year < 0 || year > 9999 {
		return Date{}, newTimeError(
			ErrOverflow,
			"date year is outside the supported civil domain",
			fmt.Sprintf("year=%d", year),
			"use a time whose projected year is between 0000 and 9999",
		)
	}
	return dateFromTimeTrusted(t), nil
}

func dateFromTimeTrusted(t time.Time) Date {
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
// Month/year arithmetic applies end-of-month clamping. To move back, pass a
// negative Period literal or handle the error returned by Period.Negate.
// Add returns ErrInvalidDate for an invalid receiver and ErrOverflow when the
// result would leave the supported civil year domain.
func (d Date) Add(p Period) (Date, error) {
	if msg := validateDateComponents(d.year, int(d.month), d.day); msg != "" {
		return Date{}, newTimeError(
			ErrInvalidDate,
			msg,
			d.String(),
			"construct Date with NewDate before applying calendar arithmetic",
		)
	}

	monthIndex := int64(d.month) - 1 + int64(p.Months)
	year := int64(d.year) + int64(p.Years) + monthIndex/12
	monthIndex %= 12
	if monthIndex < 0 {
		monthIndex += 12
		year--
	}
	if year < 0 || year > 9999 {
		return dateAddOverflow(d, p)
	}

	month := time.Month(monthIndex + 1)
	day := min(d.day, daysInMonth(int(year), month))
	result := time.Date(int(year), month, day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(p.Days))
	if result.Year() < 0 || result.Year() > 9999 {
		return dateAddOverflow(d, p)
	}
	return dateFromTimeTrusted(result), nil
}

func dateAddOverflow(d Date, p Period) (Date, error) {
	return Date{}, newTimeError(
		ErrOverflow,
		"date addition leaves the supported civil domain",
		fmt.Sprintf("date=%s period=%s", d, p.ISO8601()),
		"use a smaller period so the resulting year remains between 0000 and 9999",
	)
}

// DaysUntil returns the signed number of calendar days from d to other.
// It returns ErrInvalidDate if either endpoint is invalid.
func (d Date) DaysUntil(other Date) (int, error) {
	if msg := validateDateComponents(d.year, int(d.month), d.day); msg != "" {
		return 0, newTimeError(
			ErrInvalidDate,
			msg,
			d.String(),
			"construct the start Date with NewDate before calculating a day difference",
		)
	}
	if msg := validateDateComponents(other.year, int(other.month), other.day); msg != "" {
		return 0, newTimeError(
			ErrInvalidDate,
			msg,
			other.String(),
			"construct the end Date with NewDate before calculating a day difference",
		)
	}
	return other.dayNumber() - d.dayNumber(), nil
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

// MarshalJSON encodes d as {"kind":"date","value":"YYYY-MM-DD"}.
func (d Date) MarshalJSON() ([]byte, error) {
	if msg := validateDateComponents(d.year, int(d.month), d.day); msg != "" {
		return nil, newTimeError(
			ErrInvalidDate,
			msg,
			d.String(),
			"marshal a Date constructed with NewDate",
		)
	}
	return json.Marshal(struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}{Kind: "date", Value: d.String()})
}

// UnmarshalJSON decodes d from {"kind":"date","value":"YYYY-MM-DD"}.
func (d *Date) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("date", wire.Kind, "date"); err != nil {
		return err
	}
	if err := requireJSONString("date", "value", wire.Value); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", wire.Value)
	if err != nil {
		return newTimeErrorWithCause(
			ErrInvalidDate,
			err,
			"date value is not a valid calendar date",
			wire.Value,
			"use a real calendar date such as 2026-03-27",
		)
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
