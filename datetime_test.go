package gotime

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	testZoneTokyo   = MustLoadZone("Asia/Tokyo")
	testZoneNewYork = MustLoadZone("America/New_York")
)

func makeDateTime(year int, month time.Month, day, hour, min, sec int, z Zone) DateTime {
	d := mustDate(year, month, day)
	ct := mustTime(hour, min, sec)
	return mustDateTime(d, ct, z)
}

func TestNewDateTime_Components(t *testing.T) {
	d := mustDate(2026, time.March, 27)
	ct := mustTime(13, 0, 0)
	dt, err := NewDateTime(d, ct, testZoneTokyo)
	if err != nil {
		t.Fatalf("NewDateTime() error: %v", err)
	}

	if !dt.Date().Equal(d) {
		t.Errorf("Date() = %v, want %v", dt.Date(), d)
	}
	if !dt.Clock().Equal(ct) {
		t.Errorf("Time() = %v, want %v", dt.Clock(), ct)
	}
	if !dt.Zone().Equal(testZoneTokyo) {
		t.Errorf("Zone() = %v, want Tokyo", dt.Zone())
	}
}

func TestNewDateTime_NonexistentLocalTime(t *testing.T) {
	d := mustDate(2026, time.March, 8)
	ct := mustTime(2, 30, 0)
	if _, err := NewDateTime(d, ct, testZoneNewYork); err == nil {
		t.Fatal("NewDateTime(nonexistent local time) error = nil, want error")
	}
}

func TestNewDateTime_DuplicateLocalTime(t *testing.T) {
	d := mustDate(2026, time.November, 1)
	ct := mustTime(1, 30, 0)
	if _, err := NewDateTime(d, ct, testZoneNewYork); err == nil {
		t.Fatal("NewDateTime(duplicate local time) error = nil, want error")
	}
}

func TestNewDateTime_DuplicateLocalTime_NonHourDSTTransition(t *testing.T) {
	d := mustDate(2026, time.April, 5)
	ct := mustTime(1, 45, 0)
	z := MustLoadZone("Australia/Lord_Howe")
	_, err := NewDateTime(d, ct, z)
	if !errors.Is(err, ErrDuplicateTime) {
		t.Fatalf("NewDateTime(non-hour duplicate local time) error = %v, want ErrDuplicateTime", err)
	}
}

func TestNewDateTime_NonexistentLocalDate(t *testing.T) {
	d := mustDate(2011, time.December, 30)
	ct := mustTime(12, 0, 0)
	z := MustLoadZone("Pacific/Apia")
	_, err := NewDateTime(d, ct, z)
	if !errors.Is(err, ErrNonexistentTime) {
		t.Fatalf("NewDateTime(skipped local date) error = %v, want ErrNonexistentTime", err)
	}
}

func TestDateTimeFromTime_RoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 27, 13, 0, 0, 0, testZoneTokyo.Location())
	dt, err := DateTimeFromTime(ts, testZoneTokyo)
	if err != nil {
		t.Fatalf("DateTimeFromTime() error = %v", err)
	}

	want := InstantFromTime(ts)
	if !dt.Instant().Equal(want) {
		t.Errorf("Instant() round-trip failed: got %v, want %v", dt.Instant(), want)
	}
}

func TestDateTimeFromTime_RejectsProjectedYearOutsideCivilDomain(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		zone Zone
	}{
		{
			name: "below minimum after westward projection",
			t:    time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC),
			zone: testZoneNewYork,
		},
		{
			name: "above maximum after eastward projection",
			t:    time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC),
			zone: testZoneTokyo,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DateTimeFromTime(tc.t, tc.zone)
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("DateTimeFromTime(%v, %v) error = %v, want ErrOverflow", tc.t, tc.zone, err)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("DateTimeFromTime(%v, %v) error = %#v, want TimeError with hint", tc.t, tc.zone, err)
			}
		})
	}
}

func TestDateTime_Instant_In_RoundTrip(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	roundTripped := mustInstantIn(t, dt.Instant(), testZoneTokyo)
	if !roundTripped.Equal(dt) {
		t.Errorf("Instant().In(zone) round-trip failed: got %v, want %v", roundTripped, dt)
	}
}

func TestDateTime_In_SameMoment(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	dtUTC := mustDateTimeIn(t, dt, UTC)

	if !dtUTC.Instant().Equal(dt.Instant()) {
		t.Error("In(UTC).Instant() should equal original Instant()")
	}
	if !dtUTC.Zone().Equal(UTC) {
		t.Errorf("after In(UTC), Zone() = %v, want UTC", dtUTC.Zone())
	}
}

func TestDateTime_In_ChangesZone(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	dtNY := mustDateTimeIn(t, dt, testZoneNewYork)
	if !dtNY.Zone().Equal(testZoneNewYork) {
		t.Errorf("Zone after In(NY) = %v, want New_York", dtNY.Zone())
	}
}

func TestDateTime_InRejectsProjectedYearOutsideCivilDomain(t *testing.T) {
	tests := []struct {
		name string
		dt   DateTime
		zone Zone
	}{
		{
			name: "below minimum after westward projection",
			dt:   makeDateTime(0, time.January, 1, 0, 0, 0, UTC),
			zone: testZoneNewYork,
		},
		{
			name: "above maximum after eastward projection",
			dt:   makeDateTime(9999, time.December, 31, 23, 59, 59, UTC),
			zone: testZoneTokyo,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.dt.In(tc.zone)
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("DateTime.In(%v) error = %v, want ErrOverflow", tc.zone, err)
			}
		})
	}
}

func TestDateTime_Add_Sub(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	dt2 := mustDateTimeAdd(t, dt, 2*Hour)
	got, err := dt2.Sub(dt)
	if err != nil {
		t.Fatalf("Add(2*Hour).Sub() error = %v", err)
	}
	if got.InHours() != 2.0 {
		t.Errorf("Add(2*Hour).Sub() = %v hours, want 2.0", got.InHours())
	}
}

func TestDateTime_AddRejectsCivilDomainOverflow(t *testing.T) {
	maxDate := mustDate(9999, time.December, 31)
	maxClock := mustTimeNanos(23, 59, 59, 999_999_999)
	maxDateTime := mustDateTime(maxDate, maxClock, UTC)
	tests := []struct {
		name string
		dt   DateTime
		add  Duration
	}{
		{
			name: "below minimum",
			dt:   makeDateTime(0, time.January, 1, 0, 0, 0, UTC),
			add:  -Nanosecond,
		},
		{
			name: "above maximum",
			dt:   maxDateTime,
			add:  Nanosecond,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.dt.Add(tc.add)
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("DateTime.Add(%v) error = %v, want ErrOverflow", tc.add, err)
			}
		})
	}
}

func TestDateTime_AddZeroPreservesCivilBoundary(t *testing.T) {
	dt := makeDateTime(0, time.January, 1, 0, 0, 0, UTC)
	got, err := dt.Add(0)
	if err != nil {
		t.Fatalf("DateTime.Add(0) error = %v", err)
	}
	if !got.Equal(dt) || !got.Zone().Equal(dt.Zone()) || !got.Clock().Equal(dt.Clock()) {
		t.Fatalf("DateTime.Add(0) = %v, want exact boundary value %v", got, dt)
	}
}

func TestDateTime_Sub_Negative(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	dt2 := mustDateTimeAdd(t, dt, -30*Minute)
	got, err := dt2.Sub(dt)
	if err != nil {
		t.Fatalf("Add(-30*Minute).Sub() error = %v", err)
	}
	if got.InMinutes() != -30.0 {
		t.Errorf("Add(-30*Minute).Sub() = %v minutes, want -30.0", got.InMinutes())
	}
}

func TestDateTime_SubRejectsOverflow(t *testing.T) {
	t.Parallel()

	start := makeDateTime(0, time.January, 1, 0, 0, 0, UTC)
	end := makeDateTime(9999, time.December, 31, 23, 59, 59, UTC)
	_, err := end.Sub(start)
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Sub() error = %v, want ErrOverflow", err)
	}
}

func TestDateTime_BeforeAfter(t *testing.T) {
	dt1 := makeDateTime(2026, time.March, 27, 9, 0, 0, testZoneTokyo)
	dt2 := makeDateTime(2026, time.March, 27, 18, 0, 0, testZoneTokyo)

	if !dt1.Before(dt2) {
		t.Error("earlier datetime should be Before later")
	}
	if !dt2.After(dt1) {
		t.Error("later datetime should be After earlier")
	}
}

func TestDateTime_Equal_AcrossZones(t *testing.T) {
	// Same absolute moment, different zones
	dt1 := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo) // UTC+9 → 04:00 UTC
	dt2 := mustDateTimeIn(t, dt1, UTC)
	if !dt1.Equal(dt2) {
		t.Error("same absolute moment in different zones should be Equal")
	}
}

func TestDateTime_Compare(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	dtCopy := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	if dt.Compare(dtCopy) != 0 {
		t.Errorf("Compare(equal) = %d, want 0", dt.Compare(dtCopy))
	}

	earlier := makeDateTime(2026, time.March, 27, 12, 0, 0, testZoneTokyo)
	if dt.Compare(earlier) != 1 {
		t.Errorf("Compare(earlier) = %d, want 1", dt.Compare(earlier))
	}
	if earlier.Compare(dt) != -1 {
		t.Errorf("earlier.Compare(later) = %d, want -1", earlier.Compare(dt))
	}
}

func TestDateTime_IsZero(t *testing.T) {
	var dt DateTime
	if !dt.IsZero() {
		t.Error("zero DateTime should be zero")
	}
	if makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo).IsZero() {
		t.Error("non-zero DateTime should not be zero")
	}
}

func TestDateTime_AddCalendar_Basic(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)

	// +1 calendar day: same H:M:S, next date
	dt1 := mustDateTimeAddPeriod(t, dt, Days(1))
	if !dt1.Date().Equal(mustDate(2026, time.March, 28)) {
		t.Errorf("AddPeriod(+1) date = %v, want 2026-03-28", dt1.Date())
	}
	if !dt1.Clock().Equal(dt.Clock()) {
		t.Errorf("AddPeriod(+1) time = %v, want %v", dt1.Clock(), dt.Clock())
	}
	if !dt1.Zone().Equal(testZoneTokyo) {
		t.Errorf("AddPeriod(+1) zone = %v, want Tokyo", dt1.Zone())
	}
}

func TestDateTime_AddCalendar_Negative(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	dt1 := mustDateTimeAddPeriod(t, dt, Days(-1))
	if !dt1.Date().Equal(mustDate(2026, time.March, 26)) {
		t.Errorf("AddPeriod(-1) date = %v, want 2026-03-26", dt1.Date())
	}
	if !dt1.Clock().Equal(dt.Clock()) {
		t.Errorf("AddPeriod(-1) time = %v, want %v", dt1.Clock(), dt.Clock())
	}
}

func TestDateTime_AddCalendar_Zero(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	dt0 := mustDateTimeAddPeriod(t, dt, Days(0))
	if !dt0.Equal(dt) {
		t.Errorf("AddPeriod(0) = %v, want %v", dt0, dt)
	}
}

func TestDateTime_AddCalendar_DST(t *testing.T) {
	// US/Eastern: 2026-03-08 02:00 clocks spring forward to 03:00 (UTC-5 → UTC-4)
	// A datetime at 01:30 on 2026-03-08 should produce 01:30 on 2026-03-09 after +1 calendar day
	zoneNY := testZoneNewYork
	dt := makeDateTime(2026, time.March, 8, 1, 30, 0, zoneNY)
	dt1 := mustDateTimeAddPeriod(t, dt, Days(1))

	want := mustDate(2026, time.March, 9)
	if !dt1.Date().Equal(want) {
		t.Errorf("DST AddPeriod(+1) date = %v, want %v", dt1.Date(), want)
	}
	// Wall-clock time should still be 01:30 even though DST changed
	wantTime := mustTime(1, 30, 0)
	if !dt1.Clock().Equal(wantTime) {
		t.Errorf("DST AddPeriod(+1) time = %v, want 01:30", dt1.Clock())
	}
}

func TestDateTime_AddCalendar_Months_Simple(t *testing.T) {
	// Jan 15 + 1 month = Feb 15 (no clamping needed)
	dt := makeDateTime(2026, time.January, 15, 10, 0, 0, UTC)
	got := mustDateTimeAddPeriod(t, dt, Months(1))
	want := makeDateTime(2026, time.February, 15, 10, 0, 0, UTC)
	if !got.Equal(want) {
		t.Errorf("AddPeriod(Months(1)) = %v, want %v", got, want)
	}
}

func TestDateTime_AddCalendar_Months_CrossYear(t *testing.T) {
	// Nov 15 + 2 months = Jan 15 next year
	dt := makeDateTime(2025, time.November, 15, 10, 0, 0, UTC)
	got := mustDateTimeAddPeriod(t, dt, Months(2))
	want := makeDateTime(2026, time.January, 15, 10, 0, 0, UTC)
	if !got.Equal(want) {
		t.Errorf("AddPeriod(Months(2)) = %v, want %v", got, want)
	}
}

func TestDateTime_AddCalendar_Months_Negative(t *testing.T) {
	// Mar 15 - 1 month = Feb 15
	dt := makeDateTime(2026, time.March, 15, 10, 0, 0, UTC)
	got := mustDateTimeAddPeriod(t, dt, Months(-1))
	want := makeDateTime(2026, time.February, 15, 10, 0, 0, UTC)
	if !got.Equal(want) {
		t.Errorf("AddPeriod(Months(-1)) = %v, want %v", got, want)
	}
}

func TestDateTime_AddCalendar_EndOfMonth_Clamping(t *testing.T) {
	tests := []struct {
		name     string
		dt       DateTime
		d        Period
		wantDate Date
	}{
		{
			name:     "Jan31+1month=Feb28 non-leap",
			dt:       makeDateTime(2025, time.January, 31, 12, 0, 0, UTC),
			d:        Months(1),
			wantDate: mustDate(2025, time.February, 28),
		},
		{
			name:     "Jan31+1month=Feb29 leap",
			dt:       makeDateTime(2024, time.January, 31, 12, 0, 0, UTC),
			d:        Months(1),
			wantDate: mustDate(2024, time.February, 29),
		},
		{
			name:     "Mar31-1month=Feb28 non-leap",
			dt:       makeDateTime(2025, time.March, 31, 12, 0, 0, UTC),
			d:        Months(-1),
			wantDate: mustDate(2025, time.February, 28),
		},
		{
			name:     "Nov30+3months=Feb28 non-leap",
			dt:       makeDateTime(2024, time.November, 30, 12, 0, 0, UTC),
			d:        Months(3),
			wantDate: mustDate(2025, time.February, 28),
		},
		{
			name:     "Feb29+1year=Feb28 leap-to-non-leap",
			dt:       makeDateTime(2024, time.February, 29, 12, 0, 0, UTC),
			d:        Years(1),
			wantDate: mustDate(2025, time.February, 28),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mustDateTimeAddPeriod(t, tc.dt, tc.d)
			if !got.Date().Equal(tc.wantDate) {
				t.Errorf("got date %v, want %v", got.Date(), tc.wantDate)
			}
			// Wall-clock time preserved
			if !got.Clock().Equal(tc.dt.Clock()) {
				t.Errorf("time-of-day changed: got %v, want %v", got.Clock(), tc.dt.Clock())
			}
		})
	}
}

func TestDateTime_AddCalendar_Years_Simple(t *testing.T) {
	dt := makeDateTime(2024, time.March, 15, 10, 0, 0, UTC)
	got := mustDateTimeAddPeriod(t, dt, Years(1))
	want := makeDateTime(2025, time.March, 15, 10, 0, 0, UTC)
	if !got.Equal(want) {
		t.Errorf("AddPeriod(Years(1)) = %v, want %v", got, want)
	}
}

func TestDateTime_AddCalendar_Months_LargeDelta(t *testing.T) {
	tests := []struct {
		name string
		dt   DateTime
		p    Period
		want DateTime
	}{
		{
			name: "positive delta clamps",
			dt:   makeDateTime(2024, time.January, 31, 10, 0, 0, UTC),
			p:    Months(25),
			want: makeDateTime(2026, time.February, 28, 10, 0, 0, UTC),
		},
		{
			name: "negative delta clamps",
			dt:   makeDateTime(2026, time.March, 31, 10, 0, 0, UTC),
			p:    Months(-25),
			want: makeDateTime(2024, time.February, 29, 10, 0, 0, UTC),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mustDateTimeAddPeriod(t, tc.dt, tc.p)
			if !got.Equal(tc.want) {
				t.Errorf("AddPeriod(%v) = %v, want %v", tc.p, got, tc.want)
			}
		})
	}
}

func TestDateTime_AddCalendar_Months_DST(t *testing.T) {
	// US/Eastern: adding 1 month to 2026-02-08 01:30 should yield 2026-03-08 01:30
	// (wall-clock preserved; result is in DST zone, 2026-03-08 is after spring-forward)
	zoneNY := testZoneNewYork
	dt := makeDateTime(2026, time.February, 8, 1, 30, 0, zoneNY)
	got := mustDateTimeAddPeriod(t, dt, Months(1))
	wantDate := mustDate(2026, time.March, 8)
	wantTime := mustTime(1, 30, 0)
	if !got.Date().Equal(wantDate) {
		t.Errorf("date = %v, want %v", got.Date(), wantDate)
	}
	if !got.Clock().Equal(wantTime) {
		t.Errorf("time = %v, want 01:30", got.Clock())
	}
}

func TestDateTime_String(t *testing.T) {
	dt := makeDateTime(2026, time.March, 27, 13, 0, 0, testZoneTokyo)
	s := dt.String()
	// Should contain date and time
	if !strings.Contains(s, "2026") || !strings.Contains(s, "13:00") {
		t.Errorf("String() = %q, expected date 2026 and time 13:00", s)
	}
	// Should contain timezone offset
	if !strings.Contains(s, "+09:00") {
		t.Errorf("String() = %q, expected +09:00 offset for Tokyo", s)
	}
}
