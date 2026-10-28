package gotime

import (
	"archive/zip"
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestZoneRejectsLocalIdentity(t *testing.T) {
	t.Parallel()
	for _, load := range []func(string) (Zone, error){LoadZone, ResolveZone} {
		_, err := load("Local")
		var detail *TimeError
		if !errors.Is(err, ErrInvalidZone) || !errors.As(err, &detail) || detail.Hint == "" {
			t.Errorf("Local: %v", err)
		}
	}
	z := MustLoadZone("Asia/Tokyo")
	original := z
	err := json.Unmarshal([]byte(`"Local"`), &z)
	if !errors.Is(err, ErrInvalidZone) || !z.Equal(original) {
		t.Errorf("zone decode: %v %v", z, err)
	}
	i, _ := ParseInstant("2026-01-01T00:00:00Z")
	dt, err := i.In(original)
	if err != nil {
		t.Fatal(err)
	}
	before := dt
	err = json.Unmarshal([]byte(`{"instant":"2026-01-01T00:00:00Z","zone":"Local"}`), &dt)
	if !errors.Is(err, ErrInvalidZone) || !dt.Equal(before) || !dt.Zone().Equal(before.Zone()) {
		t.Errorf("datetime decode: %v %v", dt, err)
	}
	for _, id := range []string{"UTC", "US/Eastern", "Etc/UTC"} {
		if _, err := LoadZone(id); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestZoneNamedIdentity(t *testing.T) {
	for _, source := range []string{"directory", "zip"} {
		t.Run(source, func(t *testing.T) {
			if os.Getenv("GOTIME_IDENTITY_TEST") != t.Name() {
				dir := t.TempDir()
				data := snapshotTZif(nil, nil, []int32{1234}, "")
				names := []string{"Not/IANA", "UTC+8", "+08:00", "Local", "America/New_York", "Asia/Tokyo"}
				path := dir
				if source == "zip" {
					var buf bytes.Buffer
					archive := zip.NewWriter(&buf)
					for _, name := range names {
						entry, err := archive.Create(name)
						if err != nil {
							t.Fatal(err)
						}
						if _, err := entry.Write(data); err != nil {
							t.Fatal(err)
						}
					}
					if err := archive.Close(); err != nil {
						t.Fatal(err)
					}
					path = filepath.Join(dir, "rules.zip")
					if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
						t.Fatal(err)
					}
				} else {
					for _, name := range names {
						file := filepath.Join(dir, name)
						if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(file, data, 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
				cmd.Env = append(os.Environ(), "GOTIME_IDENTITY_TEST="+t.Name(), "ZONEINFO="+path)
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("subprocess: %v\n%s", err, output)
				}
				return
			}
			original := MustLoadZone("Asia/Tokyo")
			instant := InstantFromTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
			before, err := instant.In(original)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Not/IANA", "UTC+8", "+08:00", "Local", "asia/tokyo", "us/eastern", "etc/utc", "utc"} {
				assertInvalid := func(err error) {
					t.Helper()
					var detail *TimeError
					if !errors.Is(err, ErrInvalidZone) || !errors.As(err, &detail) || detail.Hint == "" {
						t.Errorf("%s: want invalid zone diagnostic, got %v", name, err)
					}
				}
				_, err := LoadZone(name)
				assertInvalid(err)
				if name == "Not/IANA" || name == "UTC+8" || name == "+08:00" || name == "Local" {
					_, err := ResolveZone(name)
					assertInvalid(err)
				}
				z := original
				assertInvalid(json.Unmarshal([]byte(fmt.Sprintf("%q", name)), &z))
				if !z.Equal(original) {
					t.Errorf("%s changed Zone on failure", name)
				}
				dt := before
				assertInvalid(json.Unmarshal([]byte(fmt.Sprintf(`{"instant":"2026-01-01T00:00:00Z","zone":%q}`, name)), &dt))
				if !dt.Equal(before) || !dt.Zone().Equal(before.Zone()) {
					t.Errorf("%s changed DateTime on failure", name)
				}
			}
			for _, name := range []string{"UTC", "US/Eastern", "Etc/UTC", "Etc/GMT+8", "Asia/Tokyo", "America/New_York"} {
				z, err := LoadZone(name)
				if err != nil || z.ID() != name {
					t.Fatalf("%s: %v %v", name, z, err)
				}
				if name == "America/New_York" || name == "Asia/Tokyo" {
					_, offset := instant.Std().In(z.Location()).Zone()
					if offset != 1234 {
						t.Errorf("%s ignored ZONEINFO override: %d", name, offset)
					}
				}
			}
		})
	}
}

func TestResolveZoneLinkSpelling(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ input, want string }{
		{"asia/tokyo", "Asia/Tokyo"}, {"us/eastern", "US/Eastern"},
		{"etc/utc", "Etc/UTC"}, {"utc", "UTC"},
		{"US/Eastern", "US/Eastern"}, {"Tokyo Standard Time", "Asia/Tokyo"},
	} {
		z, err := ResolveZone(tc.input)
		if err != nil || z.ID() != tc.want {
			t.Errorf("%s: ID=%s err=%v, want %s", tc.input, z.ID(), err, tc.want)
		}
	}
}
