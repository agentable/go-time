package gotime

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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

			got := tc.d.Add(tc.p.Negate())
			if !got.Equal(tc.want) {
				t.Errorf("Add(%v.Negate()) = %v, want %v", tc.p, got, tc.want)
			}
		})
	}
}

func TestDate_Sub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		d     Date
		other Date
		want  Period
	}{
		{name: "same date", d: mustDate(2026, time.March, 27), other: mustDate(2026, time.March, 27), want: Period{}},
		{name: "borrows days", d: mustDate(2026, time.March, 1), other: mustDate(2026, time.January, 31), want: Period{Months: 1, Days: 1}},
		{name: "negative", d: mustDate(2026, time.January, 31), other: mustDate(2026, time.March, 1), want: Period{Months: -1, Days: -1}},
		{name: "across years", d: mustDate(2026, time.March, 2), other: mustDate(2024, time.December, 31), want: Period{Years: 1, Months: 2, Days: 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.d.Sub(tc.other)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Sub() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDateTime_AddPeriod_Negated(t *testing.T) {
	t.Parallel()

	dt := makeDateTime(2026, time.March, 31, 9, 30, 0, UTC)
	got := dt.AddPeriod(Months(1).Negate())
	want := makeDateTime(2026, time.February, 28, 9, 30, 0, UTC)
	if !got.Equal(want) {
		t.Errorf("AddPeriod(Months(1).Negate()) = %v, want %v", got, want)
	}
	if !got.Clock().Equal(dt.Clock()) {
		t.Errorf("calendar subtraction changed wall-clock time: got %v, want %v", got.Clock(), dt.Clock())
	}
}
