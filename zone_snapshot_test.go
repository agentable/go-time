package gotime

import (
	"bytes"
	"encoding/binary"
	"encoding/json/v2"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/text/language"
)

func TestZoneEncodingKeepsLoadedRules(t *testing.T) {
	if os.Getenv("GOTIME_RULE_TEST") != "1" {
		dir := t.TempDir()
		cmd := exec.Command(os.Args[0], "-test.run=^TestZoneEncodingKeepsLoadedRules$")
		cmd.Env = append(os.Environ(), "GOTIME_RULE_TEST=1", "ZONEINFO="+dir)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("subprocess: %v\n%s", err, output)
		}
		return
	}
	dir := os.Getenv("ZONEINFO")
	if err := os.MkdirAll(filepath.Join(dir, "America"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "America", "New_York")
	writeRules := func(offset int32) {
		// TZif v1: no transitions, one local type and its NUL-terminated name.
		data := make([]byte, 54)
		copy(data, "TZif")
		binary.BigEndian.PutUint32(data[36:40], 1)
		binary.BigEndian.PutUint32(data[40:44], 4)
		binary.BigEndian.PutUint32(data[44:48], uint32(offset))
		copy(data[50:], "TST\x00")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeRules(-12 * 3600)
	z, err := LoadZone("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	i, err := ParseInstant("9999-12-31T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	dt, err := i.In(z)
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(dt)
	if err != nil {
		t.Fatal(err)
	}
	zoneBefore, err := json.Marshal(z)
	if err != nil {
		t.Fatal(err)
	}
	writeRules(14 * 3600)
	for step := range 2 {
		after, err := json.Marshal(dt)
		if err != nil || !bytes.Equal(before, after) {
			t.Errorf("step %d: %s %v want %s", step, after, err, before)
		}
		zoneAfter, err := json.Marshal(z)
		if err != nil || !bytes.Equal(zoneBefore, zoneAfter) {
			t.Errorf("zone step %d: %s %v", step, zoneAfter, err)
		}
		if step == 0 {
			var decoded DateTime
			if err := json.Unmarshal(before, &decoded); !errors.Is(err, ErrOverflow) {
				t.Errorf("decode should load new rules: %v", err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestDateTimeConstructionAndWireDomains(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		year      int
		month     time.Month
		day, hour int
		zone      string
	}{{0, time.January, 1, 0, "Asia/Tokyo"}, {9999, time.December, 31, 23, "America/New_York"}} {
		d, err := NewDate(tc.year, tc.month, tc.day)
		if err != nil {
			t.Fatal(err)
		}
		clock, err := NewTime(tc.hour, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		dt, err := NewDateTime(d, clock, MustLoadZone(tc.zone))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := json.Marshal(dt); !errors.Is(err, ErrOverflow) {
			t.Errorf("%s: %v", dt, err)
		}
	}
}

func TestZoneResolutionShortRulePeriod(t *testing.T) {
	if os.Getenv("GOTIME_SHORT_RULE_TEST") != "1" {
		dir := t.TempDir()
		cmd := exec.Command(os.Args[0], "-test.run=^TestZoneResolutionShortRulePeriod$")
		cmd.Env = append(os.Environ(), "GOTIME_SHORT_RULE_TEST=1", "ZONEINFO="+dir)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("subprocess: %v\n%s", err, output)
		}
		return
	}
	dir := os.Getenv("ZONEINFO")
	if err := os.MkdirAll(filepath.Join(dir, "America"), 0700); err != nil {
		t.Fatal(err)
	}
	// TZif v1 with a six-minute rule period at offset -600 seconds.
	data := make([]byte, 70)
	copy(data, "TZif")
	binary.BigEndian.PutUint32(data[32:36], 2)
	binary.BigEndian.PutUint32(data[36:40], 2)
	binary.BigEndian.PutUint32(data[40:44], 4)
	for i, minute := range []int{2, 8} {
		transition := time.Date(2026, 1, 1, 0, minute, 0, 0, time.UTC).Unix()
		binary.BigEndian.PutUint32(data[44+i*4:], uint32(transition))
	}
	data[52] = 1
	binary.BigEndian.PutUint32(data[60:64], 4294966696) // int32(-600)
	copy(data[66:], "TST\x00")
	if err := os.WriteFile(filepath.Join(dir, "America", "New_York"), data, 0600); err != nil {
		t.Fatal(err)
	}
	z, err := LoadZone("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	local, err := ParseLocalDateTime("2025-12-31T23:55:00")
	if err != nil {
		t.Fatal(err)
	}
	want := []Instant{InstantFromTime(time.Date(2025, 12, 31, 23, 55, 0, 0, time.UTC)), InstantFromTime(time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC))}
	for _, instant := range want {
		projected, err := instant.In(z)
		if err != nil || projected.Date() != local.Date || projected.Clock() != local.Time {
			t.Fatalf("fixture oracle mismatch: %v %v", projected, err)
		}
	}
	r := local.Resolve(z)
	if r.Status != LocalAmbiguous || len(r.Candidates) != 2 {
		t.Errorf("Resolve: status=%s candidates=%v, want both %v", r.Status, r.Candidates, want)
	} else {
		for i, candidate := range r.Candidates {
			if !candidate.Instant().Equal(want[i]) {
				t.Errorf("candidate %d = %v, want %v", i, candidate, want[i])
			}
		}
	}
	if _, err := r.Only(); !errors.Is(err, ErrDuplicateTime) {
		t.Errorf("Only: %v", err)
	}
	if _, err := ParseDateTime(local.String(), WithZone(z)); !errors.Is(err, ErrDuplicateTime) {
		t.Errorf("ParseDateTime: %v", err)
	}
	if got := Parse(local.String(), WithZone(z)); got.Status != StatusAmbiguous || len(got.Candidates) != 2 {
		t.Errorf("Parse: status=%s candidates=%d", got.Status, len(got.Candidates))
	}
	if _, err := NewDateTime(local.Date, local.Time, z); !errors.Is(err, ErrDuplicateTime) {
		t.Errorf("NewDateTime: %v", err)
	}
	previous, err := ParseDateTime("2025-12-30T23:55:00", WithZone(z))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := previous.AddPeriod(Days(1)); err != nil || got.Status != LocalAmbiguous || len(got.Candidates) != 2 {
		t.Errorf("AddPeriod: %v, %v", got, err)
	}
	reference := InstantFromTime(time.Date(2025, 12, 30, 12, 0, 0, 0, time.UTC))
	natural := Parse("tomorrow at 11:55pm", WithInputLocale(language.English), WithReference(reference), WithZone(z))
	if natural.Status != StatusAmbiguous || len(natural.Candidates) != 2 {
		t.Fatalf("natural Parse: %v", natural)
	}
	for i, candidate := range natural.Candidates {
		dt, ok := candidate.DateTime()
		if !ok || !dt.Instant().Equal(want[i]) {
			t.Errorf("natural candidate %d: %v, want %v", i, candidate, want[i])
		}
	}
	_, err = ParseDateTime(local.String(), WithZone(z))
	var detail *TimeError
	if !errors.As(err, &detail) || detail.Input != local.String() || detail.Hint == "" {
		t.Errorf("typed parse diagnostic: %#v", detail)
	}
}

func TestZoneResolutionFooterOffsets(t *testing.T) {
	if runZoneSnapshotProcess(t) {
		return
	}
	for _, tc := range []struct {
		name, footer string
		month        time.Month
		offset       int
	}{
		{"standard", "<STD>-1:02:03", time.January, 3723},
		{"default_dst", "<STD>-1:02:03<DST>,M3.2.0,M11.1.0", time.July, 7323},
		{"explicit_dst", "STD1:02:03DST-2:30:04,M3.2.0,M11.1.0", time.July, 9004},
		{"negative_standard", "STD1:02:03DST-2:30:04,M3.2.0,M11.1.0", time.January, -3723},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Neither footer offset is in the explicit type table.
			data := snapshotTZif([]int64{0}, []byte{0}, []int32{0}, tc.footer)
			z := loadSnapshotZone(t, "Test/"+tc.name, data)
			civil := time.Date(2050, tc.month, 15, 12, 0, 0, 123456789, time.UTC)
			want := civil.Add(-time.Duration(tc.offset) * time.Second)
			assertSnapshotResolution(t, z, civil, []time.Time{want})
		})
	}
}

// Isolate ZONEINFO because both loaders capture their source at first use.
func runZoneSnapshotProcess(t *testing.T) bool {
	t.Helper()
	if os.Getenv("GOTIME_SNAPSHOT_TEST") == t.Name() {
		return false
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), "GOTIME_SNAPSHOT_TEST="+t.Name(), "ZONEINFO="+t.TempDir())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("subprocess: %v\n%s", err, output)
	}
	return true
}

func loadSnapshotZone(t *testing.T, name string, data []byte) Zone {
	t.Helper()
	path := filepath.Join(os.Getenv("ZONEINFO"), name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	z, err := LoadZone(name)
	if err != nil {
		t.Fatal(err)
	}
	return z
}

func snapshotTZif(transitions []int64, indexes []byte, offsets []int32, footer string) []byte {
	var data []byte
	for _, width := range []int{4, 8} {
		header := make([]byte, 44)
		copy(header, "TZif2")
		binary.BigEndian.PutUint32(header[32:], uint32(len(transitions)))
		binary.BigEndian.PutUint32(header[36:], uint32(len(offsets)))
		binary.BigEndian.PutUint32(header[40:], 4)
		data = append(data, header...)
		for _, transition := range transitions {
			if width == 4 {
				data = binary.BigEndian.AppendUint32(data, uint32(transition))
			} else {
				data = binary.BigEndian.AppendUint64(data, uint64(transition))
			}
		}
		data = append(data, indexes...)
		for _, offset := range offsets {
			data = binary.BigEndian.AppendUint32(data, uint32(offset))
			data = append(data, 0, 0)
		}
		data = append(data, "TST\x00"...)
	}
	return append(data, "\n"+footer+"\n"...)
}

func assertSnapshotResolution(t *testing.T, z Zone, civil time.Time, want []time.Time) {
	t.Helper()
	local, err := ParseLocalDateTime(civil.Format("2006-01-02T15:04:05.999999999"))
	if err != nil {
		t.Fatal(err)
	}
	for _, instant := range want {
		projected := instant.In(z.Location())
		if projected.Format("2006-01-02T15:04:05.999999999") != local.String() {
			t.Fatalf("fixture oracle: %s projects to %s, want %s", instant, projected, local)
		}
	}
	r := local.Resolve(z)
	status := LocalResolved
	if len(want) == 0 {
		status = LocalNonexistent
	} else if len(want) > 1 {
		status = LocalAmbiguous
	}
	if r.Status != status || len(r.Candidates) != len(want) {
		t.Fatalf("Resolve(%s): %s %v, want %s %v", local, r.Status, r.Candidates, status, want)
	}
	for i, candidate := range r.Candidates {
		if !candidate.Std().Equal(want[i]) || candidate.Date() != local.Date || candidate.Clock() != local.Time {
			t.Errorf("candidate %d = %v, want %v with original civil fields", i, candidate, want[i])
		}
	}
}

func TestZoneResolutionSnapshotBoundaries(t *testing.T) {
	if runZoneSnapshotProcess(t) {
		return
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return base.Add(time.Duration(minutes) * time.Minute) }
	data := snapshotTZif([]int64{at(2).Unix(), at(8).Unix(), at(12).Unix(), at(18).Unix()}, []byte{1, 0, 2, 0}, []int32{0, -600, -1200}, "")
	z := loadSnapshotZone(t, "Test/Multiple", data)
	for _, tc := range []struct {
		name  string
		civil time.Time
		want  []time.Time
	}{
		{"three_candidates", at(-5).Add(123456789), []time.Time{at(-5).Add(123456789), at(5).Add(123456789), at(15).Add(123456789)}},
		{"overlap_start", at(-8), []time.Time{at(-8), at(2), at(12)}},
		{"overlap_end", at(-2), []time.Time{at(-2)}},
		{"gap_start", at(2), nil},
		{"gap_inside", at(4), nil},
		{"gap_last_nanosecond", at(8).Add(-time.Nanosecond), nil},
		{"gap_end", at(8), []time.Time{at(8)}},
	} {
		t.Run(tc.name, func(t *testing.T) { assertSnapshotResolution(t, z, tc.civil, tc.want) })
	}
	// Replacing rules affects new loads, never the old Location/offset snapshot.
	fresh := loadSnapshotZone(t, "Test/Multiple", snapshotTZif(nil, nil, []int32{3600}, ""))
	assertSnapshotResolution(t, fresh, at(-5), []time.Time{at(-65)})
	assertSnapshotResolution(t, z, at(-5), []time.Time{at(-5), at(5), at(15)})
	if err := os.Remove(filepath.Join(os.Getenv("ZONEINFO"), "Test", "Multiple")); err != nil {
		t.Fatal(err)
	}
	assertSnapshotResolution(t, z, at(-5), []time.Time{at(-5), at(5), at(15)})

	for _, tc := range []struct {
		name   string
		offset int32
		civil  time.Time
	}{
		{"positive", 2147483647, time.Date(0, 1, 1, 0, 0, 0, 1, time.UTC)},
		{"negative", -2147483648, time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixed := loadSnapshotZone(t, "Test/"+tc.name, snapshotTZif(nil, nil, []int32{tc.offset}, ""))
			want := tc.civil.Add(-time.Duration(tc.offset) * time.Second)
			assertSnapshotResolution(t, fixed, tc.civil, []time.Time{want})
		})
	}
}
