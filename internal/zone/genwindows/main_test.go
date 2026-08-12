package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRejectsInvalidInputsBeforeWriting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	tests := []struct {
		name      string
		input     []byte
		inputPath string
		lockPath  string
		want      string
	}{
		{name: "missing argument", want: "required"},
		{name: "missing lock", inputPath: missing, lockPath: missing, want: "source lock"},
		{name: "missing input", inputPath: missing, want: "locked source"},
		{name: "malformed XML", input: []byte("<supplementalData>"), want: "decode XML"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			caseDir := t.TempDir()
			inputPath := tc.inputPath
			lockPath := tc.lockPath
			if tc.input != nil {
				inputPath = filepath.Join(caseDir, "windowsZones.xml")
				if err := os.WriteFile(inputPath, tc.input, 0o600); err != nil {
					t.Fatalf("WriteFile(input): %v", err)
				}
			}
			if lockPath == "" {
				digest := fmt.Sprintf("%x", sha256.Sum256(tc.input))
				lockPath = writeLock(t, caseDir, digest)
			}
			out := filepath.Join(caseDir, "windows.go")
			const original = "do not replace"
			if err := os.WriteFile(out, []byte(original), 0o600); err != nil {
				t.Fatalf("WriteFile(output): %v", err)
			}

			err := run(inputPath, lockPath, out)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("run() error = %v, want substring %q", err, tc.want)
			}
			got, readErr := os.ReadFile(out)
			if readErr != nil {
				t.Fatalf("ReadFile(output): %v", readErr)
			}
			if string(got) != original {
				t.Fatalf("output changed on validation failure: %q", got)
			}
		})
	}
}

func TestRunReturnsOutputFailure(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile("testdata/windowsZones.xml")
	if err != nil {
		t.Fatalf("ReadFile(input): %v", err)
	}
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "windowsZones.xml")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	lockPath := writeLock(t, dir, fmt.Sprintf("%x", sha256.Sum256(input)))
	if err := run(inputPath, lockPath, dir); err == nil {
		t.Fatal("run(output directory) error = nil, want write failure")
	}
}

func TestReadMappingsReturnsMissingFileError(t *testing.T) {
	t.Parallel()

	_, err := readMappings(filepath.Join(t.TempDir(), "missing-windowsZones.xml"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("readMappings() error = %v, want fs.ErrNotExist", err)
	}
}

func TestRunMatchesGoldenArtifact(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile("testdata/windowsZones.xml")
	if err != nil {
		t.Fatalf("ReadFile(input): %v", err)
	}
	want, err := os.ReadFile("testdata/windows.golden.go")
	if err != nil {
		t.Fatalf("ReadFile(golden): %v", err)
	}
	dir := t.TempDir()
	digest := fmt.Sprintf("%x", sha256.Sum256(input))
	lockPath := writeLock(t, dir, digest)
	for _, name := range []string{"local-one.xml", "local-two.xml"} {
		inputPath := filepath.Join(dir, name)
		if err := os.WriteFile(inputPath, input, 0o600); err != nil {
			t.Fatalf("WriteFile(input): %v", err)
		}
		out := filepath.Join(dir, name+".go")
		if err := run(inputPath, lockPath, out); err != nil {
			t.Fatalf("run(%s) error = %v", name, err)
		}
		got, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("ReadFile(output): %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("generated artifact differs from golden\ngot:\n%s\nwant:\n%s", got, want)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), out, got, parser.AllErrors); err != nil {
			t.Fatalf("generated artifact is not valid Go: %v", err)
		}
	}
}

func TestRunRejectsChecksumMismatchBeforeWriting(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile("testdata/windowsZones.xml")
	if err != nil {
		t.Fatalf("ReadFile(input): %v", err)
	}
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "windowsZones.xml")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	lockPath := writeLock(t, dir, strings.Repeat("0", 64))
	out := filepath.Join(dir, "unchanged.go")
	const original = "do not replace"
	if err := os.WriteFile(out, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile(output): %v", err)
	}
	err = run(inputPath, lockPath, out)
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
		{name: "missing territory", xml: `<supplementalData><windowsZones><mapTimezones><mapZone other="Bad" type="Etc/UTC"/></mapTimezones></windowsZones></supplementalData>`, want: "missing territory"},
		{name: "missing target", xml: `<supplementalData><windowsZones><mapTimezones><mapZone other="Bad" territory="001"/></mapTimezones></windowsZones></supplementalData>`, want: "missing type"},
		{name: "non-001 only", xml: `<supplementalData><windowsZones><mapTimezones><mapZone other="Regional" territory="CA" type="America/Toronto"/></mapTimezones></windowsZones></supplementalData>`, want: "no territory 001"},
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
