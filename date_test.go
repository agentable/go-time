package gotime

import (
	"errors"
	"testing"
	"time"
)

func TestNewDate(t *testing.T) {
	d, err := NewDate(2026, time.March, 27)
	if err != nil {
		t.Fatalf("NewDate() error: %v", err)
	}
	if d.Year() != 2026 {
		t.Errorf("Year() = %d, want 2026", d.Year())
	}
	if d.Month() != time.March {
		t.Errorf("Month() = %v, want March", d.Month())
	}
	if d.Day() != 27 {
		t.Errorf("Day() = %d, want 27", d.Day())
	}
}

func TestNewDate_Invalid(t *testing.T) {
	if _, err := NewDate(2026, time.February, 29); err == nil {
		t.Fatal("NewDate(invalid leap day) error = nil, want error")
	}
}

func TestDateFromTime(t *testing.T) {
	loc := time.FixedZone("JST", 9*3600)
	ts := time.Date(2026, 3, 27, 13, 0, 0, 0, loc)
	d, err := DateFromTime(ts)
	if err != nil {
		t.Fatalf("DateFromTime() error = %v", err)
	}
	if d.Year() != 2026 || d.Month() != time.March || d.Day() != 27 {
		t.Errorf("DateFromTime() = %v, want 2026-03-27", d)
	}
}

func TestDateFromTime_RejectsYearOutsideCivilDomain(t *testing.T) {
	for _, year := range []int{-1, 10_000} {
		t.Run(time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006"), func(t *testing.T) {
			_, err := DateFromTime(time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC))
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("DateFromTime(year %d) error = %v, want ErrOverflow", year, err)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("DateFromTime(year %d) error = %#v, want TimeError with hint", year, err)
			}
		})
	}
}

func TestDate_Equal(t *testing.T) {
	d1 := mustDate(2026, time.March, 27)
	d2 := mustDate(2026, time.March, 27)
	d3 := mustDate(2026, time.March, 28)
	if !d1.Equal(d2) {
		t.Error("same dates should be equal")
	}
	if d1.Equal(d3) {
		t.Error("different dates should not be equal")
	}
}

func TestDate_Before(t *testing.T) {
	d1 := mustDate(2026, time.March, 27)
	d2 := mustDate(2026, time.March, 28)
	if !d1.Before(d2) {
		t.Error("earlier date should be Before later date")
	}
	if d2.Before(d1) {
		t.Error("later date should not be Before earlier date")
	}
	if d1.Before(d1) {
		t.Error("date should not be Before itself")
	}
}

func TestDate_After(t *testing.T) {
	d1 := mustDate(2026, time.March, 27)
	d2 := mustDate(2026, time.March, 28)
	if !d2.After(d1) {
		t.Error("later date should be After earlier date")
	}
	if d1.After(d2) {
		t.Error("earlier date should not be After later date")
	}
}

func TestDate_BeforeAcrossMonths(t *testing.T) {
	jan := mustDate(2026, time.January, 31)
	feb := mustDate(2026, time.February, 1)
	if !jan.Before(feb) {
		t.Error("Jan 31 should be before Feb 1")
	}
}

func TestDate_BeforeAcrossYears(t *testing.T) {
	y2025 := mustDate(2025, time.December, 31)
	y2026 := mustDate(2026, time.January, 1)
	if !y2025.Before(y2026) {
		t.Error("2025-12-31 should be before 2026-01-01")
	}
}

func TestDate_IsZero(t *testing.T) {
	var d Date
	if !d.IsZero() {
		t.Error("zero Date should be zero")
	}
	if mustDate(2026, time.March, 27).IsZero() {
		t.Error("non-zero Date should not be zero")
	}
}

func TestDate_Add(t *testing.T) {
	tests := []struct {
		name string
		d    Date
		days int32
		want Date
	}{
		{"add 3 days", mustDate(2026, time.March, 27), 3, mustDate(2026, time.March, 30)},
		{"subtract 1 day", mustDate(2026, time.March, 27), -1, mustDate(2026, time.March, 26)},
		{"add 0 days", mustDate(2026, time.March, 27), 0, mustDate(2026, time.March, 27)},
		{"month boundary", mustDate(2026, time.March, 31), 1, mustDate(2026, time.April, 1)},
		{"year boundary", mustDate(2026, time.December, 31), 1, mustDate(2027, time.January, 1)},
	}
	for _, tc := range tests {
		got, err := tc.d.Add(Days(tc.days))
		if err != nil {
			t.Fatalf("%s: Add(Days(%v)) error = %v", tc.name, tc.days, err)
		}
		if !got.Equal(tc.want) {
			t.Errorf("%s: Add(Days(%v)) = %v, want %v", tc.name, tc.days, got, tc.want)
		}
	}
}

func TestDate_AddRejectsInvalidOrOutOfDomainResult(t *testing.T) {
	tests := []struct {
		name string
		date Date
		add  Period
		want error
	}{
		{
			name: "below minimum",
			date: mustDate(0, time.January, 1),
			add:  Days(-1),
			want: ErrOverflow,
		},
		{
			name: "above maximum",
			date: mustDate(9999, time.December, 31),
			add:  Days(1),
			want: ErrOverflow,
		},
		{
			name: "invalid zero receiver",
			date: Date{},
			add:  Period{},
			want: ErrInvalidDate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.date.Add(tc.add)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Date.Add(%v) error = %v, want %v", tc.add, err, tc.want)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("Date.Add(%v) error = %#v, want TimeError with hint", tc.add, err)
			}
		})
	}
}

func TestDate_Compare(t *testing.T) {
	d1 := mustDate(2026, time.March, 27)
	d2 := mustDate(2026, time.March, 28)
	d3 := mustDate(2026, time.March, 27)

	if d1.Compare(d2) != -1 {
		t.Errorf("Compare(later) = %d, want -1", d1.Compare(d2))
	}
	if d2.Compare(d1) != 1 {
		t.Errorf("Compare(earlier) = %d, want 1", d2.Compare(d1))
	}
	if d1.Compare(d3) != 0 {
		t.Errorf("Compare(equal) = %d, want 0", d1.Compare(d3))
	}
}

func TestDate_Compare_ConsistentWithBeforeAfter(t *testing.T) {
	d1 := mustDate(2026, time.March, 27)
	d2 := mustDate(2026, time.March, 28)
	d1copy := mustDate(2026, time.March, 27)

	if (d1.Compare(d2) < 0) != d1.Before(d2) {
		t.Error("Compare < 0 should match Before")
	}
	if (d1.Compare(d2) > 0) != d1.After(d2) {
		t.Error("Compare > 0 should match After")
	}
	if (d1.Compare(d1copy) == 0) != d1.Equal(d1copy) {
		t.Error("Compare == 0 should match Equal")
	}
}

func TestDate_String(t *testing.T) {
	d := mustDate(2026, time.March, 27)
	if d.String() != "2026-03-27" {
		t.Errorf("String() = %q, want %q", d.String(), "2026-03-27")
	}
}

func TestDate_Weekday(t *testing.T) {
	tests := []struct {
		date Date
		want time.Weekday
	}{
		{mustDate(2026, time.March, 27), time.Friday},
		{mustDate(2026, time.January, 1), time.Thursday},
		{mustDate(2024, time.February, 29), time.Thursday}, // leap day
		{mustDate(2026, time.April, 5), time.Sunday},
	}
	for _, tc := range tests {
		got, err := tc.date.Weekday()
		if err != nil {
			t.Fatalf("%v.Weekday() error = %v", tc.date, err)
		}
		if got != tc.want {
			t.Errorf("%v.Weekday() = %v, want %v", tc.date, got, tc.want)
		}
	}

	for _, tc := range invalidDateQueryReceivers() {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.date.Weekday()
			if got != time.Weekday(0) {
				t.Errorf("Weekday() = %v, want zero weekday", got)
			}
			assertInvalidDateQueryError(t, err)
		})
	}
}

func TestDate_ISOWeek(t *testing.T) {
	tests := []struct {
		date     Date
		wantYear int
		wantWeek int
	}{
		{mustDate(2026, time.January, 1), 2026, 1},
		{mustDate(2025, time.December, 29), 2026, 1},
	}
	for _, tc := range tests {
		year, week, err := tc.date.ISOWeek()
		if err != nil {
			t.Fatalf("%v.ISOWeek() error = %v", tc.date, err)
		}
		if year != tc.wantYear || week != tc.wantWeek {
			t.Errorf("%v.ISOWeek() = (%d, %d), want (%d, %d)", tc.date, year, week, tc.wantYear, tc.wantWeek)
		}
	}

	for _, tc := range invalidDateQueryReceivers() {
		t.Run(tc.name, func(t *testing.T) {
			year, week, err := tc.date.ISOWeek()
			if year != 0 || week != 0 {
				t.Errorf("ISOWeek() = (%d, %d), want (0, 0)", year, week)
			}
			assertInvalidDateQueryError(t, err)
		})
	}
}

func TestDate_YearDay(t *testing.T) {
	tests := []struct {
		date Date
		want int
	}{
		{mustDate(2026, time.January, 1), 1},
		{mustDate(2026, time.December, 31), 365},
		{mustDate(2024, time.December, 31), 366},
	}
	for _, tc := range tests {
		got, err := tc.date.YearDay()
		if err != nil {
			t.Fatalf("%v.YearDay() error = %v", tc.date, err)
		}
		if got != tc.want {
			t.Errorf("%v.YearDay() = %d, want %d", tc.date, got, tc.want)
		}
	}

	for _, tc := range invalidDateQueryReceivers() {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.date.YearDay()
			if got != 0 {
				t.Errorf("YearDay() = %d, want 0", got)
			}
			assertInvalidDateQueryError(t, err)
		})
	}
}

func TestDate_DaysInMonth(t *testing.T) {
	tests := []struct {
		date Date
		want int
	}{
		{mustDate(2026, time.January, 15), 31},
		{mustDate(2026, time.February, 1), 28},
		{mustDate(2024, time.February, 1), 29}, // leap year
		{mustDate(2026, time.April, 10), 30},
	}
	for _, tc := range tests {
		got, err := tc.date.DaysInMonth()
		if err != nil {
			t.Fatalf("%v.DaysInMonth() error = %v", tc.date, err)
		}
		if got != tc.want {
			t.Errorf("%v.DaysInMonth() = %d, want %d", tc.date, got, tc.want)
		}
	}
}

func TestDate_DaysInMonth_RejectsInvalidYearOrMonth(t *testing.T) {
	tests := []struct {
		name string
		date Date
	}{
		{name: "negative year", date: Date{year: -1, month: time.January}},
		{name: "year above domain", date: Date{year: 10_000, month: time.January}},
		{name: "zero month", date: Date{year: 2026, month: 0}},
		{name: "month above domain", date: Date{year: 2026, month: 13}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.date.DaysInMonth()
			if got != 0 {
				t.Errorf("DaysInMonth() = %d, want 0", got)
			}
			assertInvalidDateQueryError(t, err)
		})
	}
}

func TestDate_DaysInMonth_DoesNotRequireValidDay(t *testing.T) {
	tests := []struct {
		date Date
		want int
	}{
		{date: Date{year: 2024, month: time.February, day: 0}, want: 29},
		{date: Date{year: 2026, month: time.February, day: 30}, want: 28},
		{date: Date{year: 2026, month: time.April, day: 31}, want: 30},
	}
	for _, tc := range tests {
		got, err := tc.date.DaysInMonth()
		if err != nil {
			t.Fatalf("%v.DaysInMonth() error = %v", tc.date, err)
		}
		if got != tc.want {
			t.Errorf("%v.DaysInMonth() = %d, want %d", tc.date, got, tc.want)
		}
	}
}

func TestDate_IsLeapYear(t *testing.T) {
	tests := []struct {
		year int
		want bool
	}{
		{2024, true},  // divisible by 4
		{2026, false}, // not divisible by 4
		{1900, false}, // divisible by 100 but not 400
		{2000, true},  // divisible by 400
		{0, true},     // year zero is in the supported civil domain
	}
	for _, tc := range tests {
		d := Date{year: tc.year}
		if got := d.IsLeapYear(); got != tc.want {
			t.Errorf("year %d: IsLeapYear() = %v, want %v", tc.year, got, tc.want)
		}
	}
}

func invalidDateQueryReceivers() []struct {
	name string
	date Date
} {
	return []struct {
		name string
		date Date
	}{
		{name: "zero value", date: Date{}},
		{name: "invalid year", date: Date{year: -1, month: time.January, day: 1}},
		{name: "invalid month", date: Date{year: 2026, month: 13, day: 1}},
		{name: "invalid day", date: Date{year: 2026, month: time.February, day: 30}},
	}
}

func assertInvalidDateQueryError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrInvalidDate) {
		t.Fatalf("error = %v, want ErrInvalidDate", err)
	}
	var te *TimeError
	if !errors.As(err, &te) {
		t.Fatalf("error = %T, want *TimeError", err)
	}
	if te.Code != CodeInvalidDate {
		t.Errorf("TimeError.Code = %q, want %q", te.Code, CodeInvalidDate)
	}
	if te.Hint == "" {
		t.Error("TimeError.Hint is empty")
	}
}

func TestDate_StringLeadingZeros(t *testing.T) {
	d := mustDate(2026, time.January, 5)
	if d.String() != "2026-01-05" {
		t.Errorf("String() = %q, want %q", d.String(), "2026-01-05")
	}
}
