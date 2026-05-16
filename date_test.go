package gotime

import (
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
	d := DateFromTime(ts)
	if d.Year() != 2026 || d.Month() != time.March || d.Day() != 27 {
		t.Errorf("DateFromTime() = %v, want 2026-03-27", d)
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
		days int
		want Date
	}{
		{"add 3 days", mustDate(2026, time.March, 27), 3, mustDate(2026, time.March, 30)},
		{"subtract 1 day", mustDate(2026, time.March, 27), -1, mustDate(2026, time.March, 26)},
		{"add 0 days", mustDate(2026, time.March, 27), 0, mustDate(2026, time.March, 27)},
		{"month boundary", mustDate(2026, time.March, 31), 1, mustDate(2026, time.April, 1)},
		{"year boundary", mustDate(2026, time.December, 31), 1, mustDate(2027, time.January, 1)},
	}
	for _, tc := range tests {
		got := tc.d.Add(Days(tc.days))
		if !got.Equal(tc.want) {
			t.Errorf("%s: Add(Days(%v)) = %v, want %v", tc.name, tc.days, got, tc.want)
		}
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
		if got := tc.date.Weekday(); got != tc.want {
			t.Errorf("%v.Weekday() = %v, want %v", tc.date, got, tc.want)
		}
	}
}

func TestDate_ISOWeek(t *testing.T) {
	// 2026-01-01 is Thursday → ISO week 1 of 2026
	d := mustDate(2026, time.January, 1)
	year, week := d.ISOWeek()
	if year != 2026 || week != 1 {
		t.Errorf("ISOWeek() = (%d, %d), want (2026, 1)", year, week)
	}
	// 2025-12-29 (Monday) → ISO week 1 of 2026
	d2 := mustDate(2025, time.December, 29)
	year2, week2 := d2.ISOWeek()
	if year2 != 2026 || week2 != 1 {
		t.Errorf("ISOWeek() = (%d, %d), want (2026, 1)", year2, week2)
	}
}

func TestDate_YearDay(t *testing.T) {
	if got := mustDate(2026, time.January, 1).YearDay(); got != 1 {
		t.Errorf("Jan 1 YearDay() = %d, want 1", got)
	}
	if got := mustDate(2026, time.December, 31).YearDay(); got != 365 {
		t.Errorf("Dec 31 (non-leap) YearDay() = %d, want 365", got)
	}
	if got := mustDate(2024, time.December, 31).YearDay(); got != 366 {
		t.Errorf("Dec 31 (leap) YearDay() = %d, want 366", got)
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
		if got := tc.date.DaysInMonth(); got != tc.want {
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
	}
	for _, tc := range tests {
		d := mustDate(tc.year, time.January, 1)
		if got := d.IsLeapYear(); got != tc.want {
			t.Errorf("year %d: IsLeapYear() = %v, want %v", tc.year, got, tc.want)
		}
	}
}

func TestDate_StringLeadingZeros(t *testing.T) {
	d := mustDate(2026, time.January, 5)
	if d.String() != "2026-01-05" {
		t.Errorf("String() = %q, want %q", d.String(), "2026-01-05")
	}
}

func TestDate_Std_ZoneProjection(t *testing.T) {
	d := mustDate(2026, time.March, 27)
	tokyo := MustLoadZone("Asia/Tokyo")
	got := d.Std(tokyo)
	want := time.Date(2026, time.March, 27, 0, 0, 0, 0, tokyo.Location())
	if !got.Equal(want) || got.Location().String() != tokyo.Location().String() {
		t.Errorf("Std(tokyo) = %v (%v), want %v (%v)", got, got.Location(), want, tokyo.Location())
	}
	if utc := d.Std(UTC); !utc.Equal(time.Date(2026, time.March, 27, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Std(UTC) = %v, want 2026-03-27 00:00:00 UTC", utc)
	}
}
