package zone

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"testing"
	"time"
)

func mustLoadLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}

func TestProjectLocalTime_Normal_UTC(t *testing.T) {
	loc := time.UTC
	result := ProjectLocalTime(loc, 2026, time.March, 15, 10, 30, 0)
	if result.Status != DSTNormal {
		t.Errorf("status = %v, want DSTNormal", result.Status)
	}
	if len(result.Times) != 1 {
		t.Fatalf("len(Times) = %d, want 1", len(result.Times))
	}
	got := result.Times[0]
	if got.Hour() != 10 || got.Minute() != 30 || got.Second() != 0 {
		t.Errorf("time = %v, want 10:30:00", got)
	}
}

func TestProjectLocalTime_Normal_Tokyo(t *testing.T) {
	loc := mustLoadLoc(t, "Asia/Tokyo")
	result := ProjectLocalTime(loc, 2026, time.July, 1, 13, 0, 0)
	if result.Status != DSTNormal {
		t.Errorf("status = %v, want DSTNormal", result.Status)
	}
	if len(result.Times) != 1 {
		t.Fatalf("len(Times) = %d, want 1", len(result.Times))
	}
}

func TestProjectLocalTime_Nonexistent_NYC_SpringForward_2026(t *testing.T) {
	// NYC spring-forward 2026: on 2026-03-08 at 02:00, clocks jump to 03:00.
	// 02:30 is in the gap.
	loc := mustLoadLoc(t, "America/New_York")
	result := ProjectLocalTime(loc, 2026, time.March, 8, 2, 30, 0)
	if result.Status != DSTNonexistent {
		t.Errorf("status = %v, want DSTNonexistent", result.Status)
	}
	if len(result.Times) != 0 {
		t.Errorf("len(Times) = %d, want 0", len(result.Times))
	}
}

func TestProjectLocalTime_Nonexistent_Paris_SpringForward_2013(t *testing.T) {
	// Paris spring-forward 2013-03-31 at 02:00 → 03:00.
	loc := mustLoadLoc(t, "Europe/Paris")
	result := ProjectLocalTime(loc, 2013, time.March, 31, 2, 30, 0)
	if result.Status != DSTNonexistent {
		t.Errorf("status = %v, want DSTNonexistent", result.Status)
	}
}

func TestProjectLocalTime_Normal_JustBeforeSpringForwardGap(t *testing.T) {
	loc := mustLoadLoc(t, "America/New_York")
	result := ProjectLocalTime(loc, 2026, time.March, 8, 1, 59, 0)
	if result.Status != DSTNormal {
		t.Errorf("status = %v, want DSTNormal", result.Status)
	}
	if len(result.Times) != 1 {
		t.Fatalf("len(Times) = %d, want 1", len(result.Times))
	}
}

func TestProjectLocalTime_Normal_JustAfterSpringForwardGap(t *testing.T) {
	loc := mustLoadLoc(t, "America/New_York")
	result := ProjectLocalTime(loc, 2026, time.March, 8, 3, 0, 0)
	if result.Status != DSTNormal {
		t.Errorf("status = %v, want DSTNormal", result.Status)
	}
	if len(result.Times) != 1 {
		t.Fatalf("len(Times) = %d, want 1", len(result.Times))
	}
}

func TestProjectLocalTime_Ambiguous_NYC_FallBack_2026(t *testing.T) {
	// NYC fall-back 2026: on 2026-11-01 at 02:00, clocks fall back to 01:00.
	// 01:30 occurs twice.
	loc := mustLoadLoc(t, "America/New_York")
	result := ProjectLocalTime(loc, 2026, time.November, 1, 1, 30, 0)
	if result.Status != DSTAmbiguous {
		t.Errorf("status = %v, want DSTAmbiguous", result.Status)
	}
	if len(result.Times) != 2 {
		t.Fatalf("len(Times) = %d, want 2", len(result.Times))
	}
}

func TestProjectLocalTime_Ambiguous_Paris_FallBack_2013(t *testing.T) {
	// Paris fall-back 2013-10-27 at 03:00 → 02:00.
	loc := mustLoadLoc(t, "Europe/Paris")
	result := ProjectLocalTime(loc, 2013, time.October, 27, 2, 30, 0)
	if result.Status != DSTAmbiguous {
		t.Errorf("status = %v, want DSTAmbiguous", result.Status)
	}
	if len(result.Times) != 2 {
		t.Fatalf("len(Times) = %d, want 2", len(result.Times))
	}
}

func TestProjectLocalTime_Normal_JustBeforeFallBack(t *testing.T) {
	loc := mustLoadLoc(t, "America/New_York")
	result := ProjectLocalTime(loc, 2026, time.November, 1, 0, 59, 0)
	if result.Status != DSTNormal {
		t.Errorf("status = %v, want DSTNormal", result.Status)
	}
	if len(result.Times) != 1 {
		t.Fatalf("len(Times) = %d, want 1", len(result.Times))
	}
}

func TestProjectLocalTime_Normal_JustAfterFallBack(t *testing.T) {
	loc := mustLoadLoc(t, "America/New_York")
	result := ProjectLocalTime(loc, 2026, time.November, 1, 2, 0, 0)
	if result.Status != DSTNormal {
		t.Errorf("status = %v, want DSTNormal", result.Status)
	}
	if len(result.Times) != 1 {
		t.Fatalf("len(Times) = %d, want 1", len(result.Times))
	}
}

func TestProjectLocalTime_Ambiguous_TwoInstantsDifferByOneHour(t *testing.T) {
	loc := mustLoadLoc(t, "America/New_York")
	result := ProjectLocalTime(loc, 2026, time.November, 1, 1, 30, 0)
	if result.Status != DSTAmbiguous {
		t.Errorf("status = %v, want DSTAmbiguous", result.Status)
	}
	if len(result.Times) != 2 {
		t.Fatalf("len(Times) = %d, want 2", len(result.Times))
	}
	diff := result.Times[1].UTC().Sub(result.Times[0].UTC())
	if diff != time.Hour {
		t.Errorf("UTC diff = %v, want 1h", diff)
	}
}

func TestProjectLocalTime_RoundTrip_UTC(t *testing.T) {
	loc := time.UTC
	result := ProjectLocalTime(loc, 2026, time.June, 15, 14, 45, 30)
	if result.Status != DSTNormal {
		t.Fatalf("status = %v, want DSTNormal", result.Status)
	}
	got := result.Times[0].In(loc)
	if got.Year() != 2026 || got.Month() != time.June || got.Day() != 15 ||
		got.Hour() != 14 || got.Minute() != 45 || got.Second() != 30 {
		t.Errorf("round-trip failed: got %v", got)
	}
}

func TestProjectLocalTime_NilLoc(t *testing.T) {
	// nil loc should default to UTC without panic
	result := ProjectLocalTime(nil, 2026, time.January, 1, 0, 0, 0)
	if result.Status != DSTNormal {
		t.Errorf("status = %v, want DSTNormal", result.Status)
	}
}

func TestProjectLocalTime_TransitionBoundsCorpus(t *testing.T) {
	t.Parallel()

	// Each seed is inside the offset period immediately before the named
	// transition. ZoneBounds supplies the transition and both adjacent offsets;
	// no expected offset or production probe duration is copied into this test.
	tests := []struct {
		name string
		zone string
		seed time.Time
	}{
		{name: "New_York_2026_spring_one_hour_gap", zone: "America/New_York", seed: time.Date(2026, time.March, 8, 6, 30, 0, 0, time.UTC)},
		{name: "New_York_2026_fall_one_hour_fold", zone: "America/New_York", seed: time.Date(2026, time.November, 1, 5, 30, 0, 0, time.UTC)},
		{name: "Lord_Howe_2024_autumn_half_hour_fold", zone: "Australia/Lord_Howe", seed: time.Date(2024, time.April, 6, 14, 30, 0, 0, time.UTC)},
		{name: "Lord_Howe_2024_spring_half_hour_gap", zone: "Australia/Lord_Howe", seed: time.Date(2024, time.October, 5, 15, 0, 0, 0, time.UTC)},
		{name: "Apia_2011_full_day_gap", zone: "Pacific/Apia", seed: time.Date(2011, time.December, 30, 9, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc := mustLoadLoc(t, tt.zone)
			_, transition := tt.seed.In(loc).ZoneBounds()
			if transition.IsZero() || !transition.After(tt.seed) {
				t.Fatalf("zone=%s seed=%s: ZoneBounds end = %s, want a later transition (runtime=%s)", tt.zone, tt.seed, transition, timeZoneRuntime())
			}

			_, oldOffset := transition.Add(-time.Nanosecond).In(loc).Zone()
			_, newOffset := transition.In(loc).Zone()
			if oldOffset == newOffset {
				t.Fatalf("zone=%s transition=%s: adjacent offsets are both %d (runtime=%s)", tt.zone, transition, oldOffset, timeZoneRuntime())
			}

			oldCivil := transition.UTC().Add(time.Duration(oldOffset) * time.Second)
			newCivil := transition.UTC().Add(time.Duration(newOffset) * time.Second)
			firstCivil, lastCivil := oldCivil, newCivil
			if lastCivil.Before(firstCivil) {
				firstCivil, lastCivil = lastCivil, firstCivil
			}
			midpoint := firstCivil.Add(lastCivil.Sub(firstCivil) / 2)

			wantStatus := DSTNonexistent
			var wantTimes []time.Time
			if newOffset < oldOffset {
				wantStatus = DSTAmbiguous
				wantTimes = []time.Time{
					midpoint.Add(-time.Duration(oldOffset) * time.Second),
					midpoint.Add(-time.Duration(newOffset) * time.Second),
				}
				slices.SortFunc(wantTimes, func(a, b time.Time) int { return a.Compare(b) })
			}

			got := ProjectLocalTime(
				loc,
				midpoint.Year(), midpoint.Month(), midpoint.Day(),
				midpoint.Hour(), midpoint.Minute(), midpoint.Second(),
			)
			if got.Status != wantStatus {
				t.Fatalf("zone=%s transition=%s civil=%s offsets=%d->%d: status = %v, want %v (runtime=%s)", tt.zone, transition, midpoint.Format(time.RFC3339), oldOffset, newOffset, got.Status, wantStatus, timeZoneRuntime())
			}
			if len(got.Times) != len(wantTimes) {
				t.Fatalf("zone=%s transition=%s civil=%s offsets=%d->%d: times = %v, want %v (runtime=%s)", tt.zone, transition, midpoint.Format(time.RFC3339), oldOffset, newOffset, got.Times, wantTimes, timeZoneRuntime())
			}
			for i := range wantTimes {
				if !got.Times[i].Equal(wantTimes[i]) {
					t.Errorf("zone=%s transition=%s civil=%s offsets=%d->%d: times[%d] = %s, want %s (runtime=%s)", tt.zone, transition, midpoint.Format(time.RFC3339), oldOffset, newOffset, i, got.Times[i], wantTimes[i], timeZoneRuntime())
				}
			}
		})
	}
}

func timeZoneRuntime() string {
	return fmt.Sprintf(
		"%s; ZONEINFO=%q; embedded time/tzdata fallback enabled",
		runtime.Version(),
		os.Getenv("ZONEINFO"),
	)
}

func TestResolveLocation_ExactCaseInsensitiveAndWindows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		wantCanonical string
	}{
		{name: "exact iana", input: "Asia/Tokyo", wantCanonical: "Asia/Tokyo"},
		{name: "case insensitive iana", input: "asia/tokyo", wantCanonical: "Asia/Tokyo"},
		{name: "windows timezone", input: "Eastern Standard Time", wantCanonical: "America/New_York"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			canonical, loc, ok := ResolveLocation(tt.input)
			if !ok {
				t.Fatalf("ResolveLocation(%q) returned ok=false", tt.input)
			}
			if canonical != tt.wantCanonical {
				t.Fatalf("ResolveLocation(%q) canonical = %q, want %q", tt.input, canonical, tt.wantCanonical)
			}
			if loc == nil {
				t.Fatalf("ResolveLocation(%q) returned nil location", tt.input)
			}
			if loc.String() != tt.wantCanonical {
				t.Errorf("ResolveLocation(%q) location = %q, want %q", tt.input, loc.String(), tt.wantCanonical)
			}
		})
	}
}

func TestWindowsToIANA_TargetsLoad(t *testing.T) {
	t.Parallel()

	for windowsName, ianaID := range WindowsToIANA {
		windowsName, ianaID := windowsName, ianaID
		t.Run(windowsName, func(t *testing.T) {
			t.Parallel()
			loc, err := time.LoadLocation(ianaID)
			if err != nil {
				t.Fatalf("CLDR target %q for Windows zone %q does not load: %v", ianaID, windowsName, err)
			}
			if loc.String() != ianaID {
				t.Fatalf("loaded location = %q, want CLDR target %q", loc.String(), ianaID)
			}
		})
	}
}

func TestResolveLocation_RejectsFixedOffsets(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"+08:30", "-0530", "UTC+8"} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if canonical, loc, ok := ResolveLocation(input); ok || canonical != "" || loc != nil {
				t.Fatalf("ResolveLocation(%q) = (%q, %v, %v), want no match", input, canonical, loc, ok)
			}
		})
	}
}

func TestResolveLocation_InvalidOffsets(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"+24:00", "+99:99", "UTC+24"} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if canonical, loc, ok := ResolveLocation(input); ok || canonical != "" || loc != nil {
				t.Fatalf("ResolveLocation(%q) = (%q, %v, %v), want no match", input, canonical, loc, ok)
			}
		})
	}
}

func TestResolveLocation_InvalidAndAlias(t *testing.T) {
	t.Parallel()

	if canonical, loc, ok := ResolveLocation(""); ok || canonical != "" || loc != nil {
		t.Fatalf("ResolveLocation(\"\") = (%q, %v, %v), want empty result", canonical, loc, ok)
	}

	if canonical, loc, ok := ResolveLocation("not/a-zone"); ok || canonical != "" || loc != nil {
		t.Fatalf("ResolveLocation(not/a-zone) = (%q, %v, %v), want empty result", canonical, loc, ok)
	}

	if _, err := time.LoadLocation("Etc/UTC"); err != nil {
		t.Skipf("Etc/UTC not available in zoneinfo: %v", err)
	}

	canonical, loc, ok := ResolveLocation("Etc/UTC")
	if !ok {
		t.Fatal("ResolveLocation(Etc/UTC) returned ok=false")
	}
	if canonical != "Etc/UTC" {
		t.Fatalf("ResolveLocation(Etc/UTC) canonical = %q, want Etc/UTC", canonical)
	}
	if loc == nil || loc.String() != "Etc/UTC" {
		t.Fatalf("ResolveLocation(Etc/UTC) location = %v, want Etc/UTC", loc)
	}
}
