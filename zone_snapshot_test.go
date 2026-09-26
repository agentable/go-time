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
