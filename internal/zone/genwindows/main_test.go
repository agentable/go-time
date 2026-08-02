package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVerifiesLockedSourceBeforeWriting(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile("testdata/windowsZones.xml")
	if err != nil {
		t.Fatalf("ReadFile(input): %v", err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(input))
	dir := t.TempDir()
	xmlPaths := []string{
		filepath.Join(dir, "first-local-name.xml"),
		filepath.Join(dir, "second-local-name.xml"),
	}
	for _, path := range xmlPaths {
		if err := os.WriteFile(path, input, 0o600); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	t.Run("verified input", func(t *testing.T) {
		lockPath := writeLock(t, dir, digest)
		generated := make([][]byte, 0, len(xmlPaths))
		for i, xmlPath := range xmlPaths {
			out := filepath.Join(dir, fmt.Sprintf("windows-%d.go", i))
			if err := run(xmlPath, lockPath, out); err != nil {
				t.Fatalf("run(%s) error = %v", xmlPath, err)
			}
			got, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("ReadFile(output): %v", err)
			}
			generated = append(generated, got)
		}
		if !bytes.Equal(generated[0], generated[1]) {
			t.Fatal("verified source bytes generated different artifacts under different local filenames")
		}
		if !strings.Contains(string(generated[0]), "Unicode CLDR release-48-1 windowsZones.xml") {
			t.Fatalf("generated header does not use locked source identity:\n%s", generated[0])
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		lockPath := writeLock(t, dir, strings.Repeat("0", 64))
		out := filepath.Join(dir, "unchanged.go")
		const original = "do not replace"
		if err := os.WriteFile(out, []byte(original), 0o600); err != nil {
			t.Fatalf("WriteFile(output): %v", err)
		}
		err := run(xmlPaths[0], lockPath, out)
		if err == nil || !strings.Contains(err.Error(), "SHA-256") {
			t.Fatalf("run() error = %v, want SHA-256 mismatch", err)
		}
		got, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("ReadFile(output): %v", err)
		}
		if string(got) != original {
			t.Fatalf("output changed on verification failure: %q", got)
		}
	})
}

func writeLock(t *testing.T, dir, cldrDigest string) string {
	t.Helper()

	path := filepath.Join(dir, "sources-"+cldrDigest[:8]+".json")
	body := fmt.Sprintf(`{
  "iana": {"version": "2025b", "url": "https://example.test/zone.tab", "sha256": %q},
  "cldr_windows": {"version": "release-48-1", "url": "https://example.test/windowsZones.xml", "sha256": %q}
}`, strings.Repeat("1", 64), cldrDigest)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile(lock): %v", err)
	}
	return path
}

func TestGenerate_SortedTerritory001AndStable(t *testing.T) {
	t.Parallel()

	mappings, err := readMappings("testdata/windowsZones.xml")
	if err != nil {
		t.Fatalf("readMappings() error = %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("len(mappings) = %d, want 2", len(mappings))
	}
	if mappings[0].windows != "Alpha Standard Time" || mappings[0].iana != "America/Godthab" {
		t.Fatalf("mappings[0] = %+v, want sorted Alpha mapping with source link preserved", mappings[0])
	}

	first, err := render("release-test", "windowsZones.xml", mappings)
	if err != nil {
		t.Fatalf("render() error = %v", err)
	}
	second, err := render("release-test", "windowsZones.xml", mappings)
	if err != nil {
		t.Fatalf("second render() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("render() is not byte-identical across calls")
	}

	got := string(first)
	alpha := strings.Index(got, `"Alpha Standard Time": "America/Godthab"`)
	zulu := strings.Index(got, `"Zulu Standard Time":  "Etc/UTC"`)
	if alpha < 0 || zulu < 0 || alpha >= zulu {
		t.Fatalf("generated mappings are missing or unsorted:\n%s", got)
	}
	if strings.Contains(got, "Ignored Regional Time") {
		t.Fatalf("generated source contains non-001 mapping:\n%s", got)
	}
	if !strings.Contains(got, "Unicode CLDR release-test") || !strings.Contains(got, `territory "001"`) {
		t.Fatalf("generated header lacks pinned release or territory:\n%s", got)
	}
}

func TestReadMappings_RejectsDuplicateWindowsName(t *testing.T) {
	t.Parallel()

	xml := `<supplementalData><windowsZones><mapTimezones>
<mapZone other="Duplicate" territory="001" type="Etc/UTC"/>
<mapZone other="Duplicate" territory="001" type="Europe/London"/>
</mapTimezones></windowsZones></supplementalData>`
	_, err := decodeMappings(strings.NewReader(xml))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("decodeMappings() error = %v, want duplicate error", err)
	}
}

func TestReadMappings_RejectsMalformedInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xml  string
		want string
	}{
		{name: "malformed XML", xml: `<supplementalData>`, want: "decode XML"},
		{name: "wrong hierarchy", xml: `<supplementalData><mapZone other="Bad" territory="001" type="Etc/UTC"/></supplementalData>`, want: "no territory 001"},
		{name: "missing Windows name", xml: `<supplementalData><windowsZones><mapTimezones><mapZone territory="001" type="Etc/UTC"/></mapTimezones></windowsZones></supplementalData>`, want: "missing other"},
		{name: "multiple default targets", xml: `<supplementalData><windowsZones><mapTimezones><mapZone other="Bad" territory="001" type="A B"/></mapTimezones></windowsZones></supplementalData>`, want: "exactly one IANA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeMappings(strings.NewReader(tt.xml))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("decodeMappings() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
