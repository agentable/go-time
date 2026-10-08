package zone

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestLoadWithoutHostData(t *testing.T) {
	rules, err := loadSources("Asia/Tokyo", []string{t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	civil := time.Date(2050, 1, 15, 12, 0, 0, 0, time.UTC)
	got := ProjectLocalTime(rules, 2050, 1, 15, 12, 0, 0)
	want := civil.Add(-9 * time.Hour)
	if got.Status != DSTNormal || len(got.Times) != 1 || !got.Times[0].Equal(want) {
		t.Fatalf("resolution = %v, want %s", got, want)
	}
}

func TestLoadSourcesZIPAndPrecedence(t *testing.T) {
	for _, method := range []uint16{zip.Store, zip.Deflate} {
		t.Run(fmt.Sprint(method), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "rules.zip")
			var buf bytes.Buffer
			archive := zip.NewWriter(&buf)
			entry, err := archive.CreateHeader(&zip.FileHeader{Name: "Test/Fixed", Method: method})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write(fixedTZif(1234)); err != nil {
				t.Fatal(err)
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			rules, err := loadSources("Test/Fixed", []string{filepath.Join(dir, "missing"), path})
			if err != nil {
				t.Fatal(err)
			}
			result := ProjectLocalTime(rules, 2026, 1, 1, 0, 0, 0)
			want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(-1234 * time.Second)
			if len(result.Times) != 1 || !result.Times[0].Equal(want) {
				t.Fatalf("ZIP resolution = %v, want %v", result, want)
			}
		})
	}
	// Invalid override data cannot prevent loading a valid later source.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Asia"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Asia", "Tokyo"), []byte("invalid TZif"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSources("Asia/Tokyo", []string{dir}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadInvalidInput(t *testing.T) {
	for _, name := range []string{"", "Local", "../UTC", "/UTC", `\UTC`, `Test\UTC`, "C:/UTC", "Asia//Tokyo"} {
		if _, err := Load(name); err == nil {
			t.Errorf("Load(%q) succeeded", name)
		}
	}
	data := fixedTZif(0)
	for size := range len(data) {
		if _, err := loadRules("Test/Truncated", data[:size]); err == nil {
			t.Fatalf("accepted truncated TZif at byte %d", size)
		}
	}
	binary.BigEndian.PutUint32(data[32:], ^uint32(0))
	if _, err := loadRules("Test/Counts", data); err == nil {
		t.Fatal("accepted transition count beyond data")
	}
	if _, err := readTZif(bytes.NewReader(make([]byte, maxTZifSize+1))); err == nil {
		t.Fatal("accepted oversized TZif")
	}
	if _, err := loadSources("Test/Missing", []string{t.TempDir()}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing source: %v", err)
	}
}

func fixedTZif(offset int32) []byte {
	data := make([]byte, 54)
	copy(data, "TZif")
	binary.BigEndian.PutUint32(data[36:], 1)
	binary.BigEndian.PutUint32(data[40:], 4)
	binary.BigEndian.PutUint32(data[44:], uint32(offset))
	copy(data[50:], "TST\x00")
	return data
}

func TestBundledRulesProjection(t *testing.T) {
	archive, err := bundledArchive()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range archive.File {
		t.Run(entry.Name, func(t *testing.T) {
			data, err := readBundledZone(entry.Name)
			if err != nil {
				t.Fatal(err)
			}
			rules, err := loadRules(entry.Name, data)
			if err != nil {
				t.Fatal(err)
			}
			for _, year := range []int{1, 1890, 1970, 2026, 2050, 9998} {
				for _, month := range []time.Month{time.January, time.July, time.December} {
					instant := time.Date(year, month, 31, 12, 34, 56, 0, time.UTC)
					assertForwardCandidate(t, rules, instant)
				}
			}
		})
	}
}

func assertForwardCandidate(t *testing.T, rules *Rules, instant time.Time) {
	t.Helper()
	civil := instant.In(rules.Location())
	result := ProjectLocalTime(rules, civil.Year(), civil.Month(), civil.Day(), civil.Hour(), civil.Minute(), civil.Second())
	if !slices.ContainsFunc(result.Times, func(candidate time.Time) bool { return candidate.Equal(instant) }) {
		t.Fatalf("lost forward projection %s -> %s: %v", instant, civil, result)
	}
}

func FuzzRulesProjection(f *testing.F) {
	f.Add(fixedTZif(-1234), int64(0))
	for _, name := range []string{"America/New_York", "Australia/Lord_Howe", "Pacific/Apia"} {
		data, err := readBundledZone(name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data, int64(2524608000))
	}
	f.Fuzz(func(t *testing.T, data []byte, seconds int64) {
		rules, err := loadRules("Test/Fuzz", data)
		if err != nil {
			return
		}
		// Keep UTC and all int32-offset projections well inside the civil domain.
		instant := time.Unix(seconds%200000000000, 0).UTC()
		assertForwardCandidate(t, rules, instant)
	})
}
