package gotime

import (
	"errors"
	"testing"
	"time"
)

func TestDate_Add_NegatedPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    Date
		p    Period
		want Date
	}{
		{name: "days", d: mustDate(2026, time.March, 27), p: Days(3), want: mustDate(2026, time.March, 24)},
		{name: "month clamps end of month", d: mustDate(2026, time.March, 31), p: Months(1), want: mustDate(2026, time.February, 28)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			negated, err := tc.p.Negate()
			if err != nil {
				t.Fatalf("Negate() error = %v", err)
			}
			got, err := tc.d.Add(negated)
			if err != nil {
				t.Fatalf("Add(%v.Negate()) error = %v", tc.p, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("Add(%v.Negate()) = %v, want %v", tc.p, got, tc.want)
			}
		})
	}
}

func TestDate_DaysUntil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		d     Date
		other Date
		want  int
	}{
		{name: "same date", d: mustDate(2026, time.March, 27), other: mustDate(2026, time.March, 27), want: 0},
		{name: "end of month to clamp target", d: mustDate(2026, time.January, 31), other: mustDate(2026, time.February, 28), want: 28},
		{name: "across leap day", d: mustDate(2024, time.February, 28), other: mustDate(2024, time.March, 1), want: 2},
		{name: "negative", d: mustDate(2026, time.March, 1), other: mustDate(2026, time.January, 31), want: -29},
		{name: "large civil range", d: mustDate(1, time.January, 1), other: mustDate(9999, time.December, 31), want: 3652058},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.d.DaysUntil(tc.other)
			if err != nil {
				t.Fatalf("DaysUntil() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("DaysUntil() = %d, want %d", got, tc.want)
			}
			reverse, err := tc.other.DaysUntil(tc.d)
			if err != nil {
				t.Fatalf("reverse DaysUntil() error = %v", err)
			}
			if reverse != -tc.want {
				t.Errorf("reverse DaysUntil() = %d, want %d", reverse, -tc.want)
			}
		})
	}
}

func TestDate_DaysUntilRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	valid := mustDate(2026, time.March, 27)
	for _, tc := range []struct {
		name  string
		start Date
		end   Date
	}{
		{name: "invalid start", start: Date{}, end: valid},
		{name: "invalid end", start: valid, end: Date{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.start.DaysUntil(tc.end)
			if !errors.Is(err, ErrInvalidDate) {
				t.Fatalf("DaysUntil() error = %v, want ErrInvalidDate", err)
			}
			var detail *TimeError
			if !errors.As(err, &detail) || detail.Hint == "" {
				t.Fatalf("DaysUntil() error = %#v, want TimeError with hint", err)
			}
		})
	}
}

func TestDateTime_AddPeriod_Negated(t *testing.T) {
	t.Parallel()

	dt := makeDateTime(2026, time.March, 31, 9, 30, 0, UTC)
	negated, err := Months(1).Negate()
	if err != nil {
		t.Fatalf("Negate() error = %v", err)
	}
	got := mustDateTimeAddPeriod(t, dt, negated)
	want := makeDateTime(2026, time.February, 28, 9, 30, 0, UTC)
	if !got.Equal(want) {
		t.Errorf("AddPeriod(Months(1).Negate()) = %v, want %v", got, want)
	}
	if !got.Clock().Equal(dt.Clock()) {
		t.Errorf("calendar subtraction changed wall-clock time: got %v, want %v", got.Clock(), dt.Clock())
	}
}

func TestDateTime_AddPeriodMatchesExplicitLocalResolution(t *testing.T) {
	tests := []struct {
		name       string
		dt         DateTime
		period     Period
		wantStatus LocalResolutionStatus
	}{
		{
			name:       "end of month",
			dt:         makeDateTime(2026, time.January, 31, 12, 0, 0, testZoneNewYork),
			period:     Months(1),
			wantStatus: LocalResolved,
		},
		{
			name:       "spring gap",
			dt:         makeDateTime(2026, time.March, 7, 2, 30, 0, testZoneNewYork),
			period:     Days(1),
			wantStatus: LocalNonexistent,
		},
		{
			name:       "fall overlap",
			dt:         makeDateTime(2026, time.October, 31, 1, 30, 0, testZoneNewYork),
			period:     Days(1),
			wantStatus: LocalAmbiguous,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.dt.AddPeriod(tc.period)
			if err != nil {
				t.Fatalf("AddPeriod(%v) error = %v", tc.period, err)
			}
			targetDate, err := tc.dt.Date().Add(tc.period)
			if err != nil {
				t.Fatalf("Date.Add(%v) error = %v", tc.period, err)
			}
			want := NewLocalDateTime(targetDate, tc.dt.Clock()).Resolve(tc.dt.Zone())
			if got.Status != tc.wantStatus || got.Status != want.Status {
				t.Fatalf("AddPeriod(%v) status = %v, want %v", tc.period, got.Status, tc.wantStatus)
			}
			if !got.Zone.Equal(want.Zone) || !got.Local.Date.Equal(want.Local.Date) || !got.Local.Time.Equal(want.Local.Time) {
				t.Fatalf("AddPeriod(%v) context = %+v, want %+v", tc.period, got, want)
			}
			if len(got.Candidates) != len(want.Candidates) {
				t.Fatalf("AddPeriod(%v) candidates = %d, want %d", tc.period, len(got.Candidates), len(want.Candidates))
			}
			for i := range got.Candidates {
				if !got.Candidates[i].Equal(want.Candidates[i]) {
					t.Fatalf("AddPeriod(%v) candidate %d = %v, want %v", tc.period, i, got.Candidates[i], want.Candidates[i])
				}
				if i > 0 && !got.Candidates[i-1].Before(got.Candidates[i]) {
					t.Fatalf("AddPeriod(%v) candidates are not chronological: %v", tc.period, got.Candidates)
				}
			}
			switch tc.wantStatus {
			case LocalResolved:
				if _, err := got.Only(); err != nil {
					t.Fatalf("AddPeriod(%v).Only() error = %v", tc.period, err)
				}
			case LocalNonexistent:
				if _, err := got.Only(); !errors.Is(err, ErrNonexistentTime) {
					t.Fatalf("AddPeriod(%v).Only() error = %v, want ErrNonexistentTime", tc.period, err)
				}
			case LocalAmbiguous:
				if _, err := got.Only(); !errors.Is(err, ErrDuplicateTime) {
					t.Fatalf("AddPeriod(%v).Only() error = %v, want ErrDuplicateTime", tc.period, err)
				}
			case LocalInvalid:
				t.Fatalf("AddPeriod(%v) returned unexpected LocalInvalid", tc.period)
			default:
				t.Fatalf("AddPeriod(%v) returned unknown status %q", tc.period, tc.wantStatus)
			}
		})
	}
}

func TestDateTime_AddPeriodRejectsCivilDomainOverflow(t *testing.T) {
	dt := makeDateTime(9999, time.December, 31, 12, 0, 0, UTC)
	resolution, err := dt.AddPeriod(Days(1))
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("AddPeriod(Days(1)) error = %v, want ErrOverflow", err)
	}
	if resolution.Status != "" || !resolution.Zone.IsZero() || resolution.Local != (LocalDateTime{}) || len(resolution.Candidates) != 0 {
		t.Fatalf("AddPeriod(Days(1)) resolution = %+v, want zero", resolution)
	}
}
