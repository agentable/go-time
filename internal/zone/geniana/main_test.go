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
		{name: "malformed row", input: []byte("US\tmissing-coordinate\n"), want: "invalid zone.tab"},
		{name: "oversized row", input: []byte("US\t+0000\t" + strings.Repeat("x", 70*1024) + "\n"), want: "scan zone.tab"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			caseDir := t.TempDir()
			inputPath := tc.inputPath
			lockPath := tc.lockPath
			if tc.input != nil {
				inputPath = filepath.Join(caseDir, "zone.tab")
				if err := os.WriteFile(inputPath, tc.input, 0o600); err != nil {
					t.Fatalf("WriteFile(input): %v", err)
				}
			}
			if lockPath == "" {
				digest := fmt.Sprintf("%x", sha256.Sum256(tc.input))
				lockPath = writeLock(t, caseDir, digest)
			}
			out := filepath.Join(caseDir, "catalog.go")
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

	input, err := os.ReadFile("testdata/zone.tab")
	if err != nil {
		t.Fatalf("ReadFile(input): %v", err)
	}
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "zone.tab")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	lockPath := writeLock(t, dir, fmt.Sprintf("%x", sha256.Sum256(input)))
	if err := run(inputPath, lockPath, dir); err == nil {
		t.Fatal("run(output directory) error = nil, want write failure")
	}
}

func TestRunLeavesOutputUnchangedOnRenderFailure(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile("testdata/zone.tab")
	if err != nil {
		t.Fatalf("ReadFile(input): %v", err)
	}
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "zone.tab")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(input))
	lockPath := writeLockVersion(t, dir, digest, "2025b\npackage broken")
	out := filepath.Join(dir, "catalog.go")
	const original = "do not replace"
	if err := os.WriteFile(out, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile(output): %v", err)
	}

	err = run(inputPath, lockPath, out)
	if err == nil || !strings.Contains(err.Error(), "format generated source") {
		t.Fatalf("run() error = %v, want format generated source", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile(output): %v", err)
	}
	if string(got) != original {
		t.Fatalf("output changed on render failure: %q", got)
	}
}

func TestReadZoneTabReturnsMissingFileError(t *testing.T) {
	t.Parallel()

	_, err := readZoneTab(filepath.Join(t.TempDir(), "missing-zone.tab"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("readZoneTab() error = %v, want fs.ErrNotExist", err)
	}
}

func TestRunMatchesGoldenArtifact(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile("testdata/zone.tab")
	if err != nil {
		t.Fatalf("ReadFile(input): %v", err)
	}
	want, err := os.ReadFile("testdata/catalog.golden.go")
	if err != nil {
		t.Fatalf("ReadFile(golden): %v", err)
	}
	dir := t.TempDir()
	digest := fmt.Sprintf("%x", sha256.Sum256(input))
	lockPath := writeLock(t, dir, digest)
	for _, name := range []string{"local-one.txt", "local-two.txt"} {
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

	dir := t.TempDir()
	input := []byte("US\t+404251-0740023\tAmerica/New_York\n")
	inputPath := filepath.Join(dir, "zone.tab")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	lockPath := writeLock(t, dir, strings.Repeat("0", 64))
	out := filepath.Join(dir, "unchanged.go")
	const original = "do not replace"
	if err := os.WriteFile(out, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile(output): %v", err)
	}
	err := run(inputPath, lockPath, out)
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

func writeLock(t *testing.T, dir, ianaDigest string) string {
	t.Helper()

	return writeLockVersion(t, dir, ianaDigest, "2025b")
}

func writeLockVersion(t *testing.T, dir, ianaDigest, version string) string {
	t.Helper()

	path := filepath.Join(dir, "sources-"+ianaDigest[:8]+".json")
	body := fmt.Sprintf(`{
  "iana": {"version": %q, "url": "https://example.test/zone.tab", "sha256": %q},
  "cldr_windows": {"version": "release-48-1", "url": "https://example.test/windowsZones.xml", "sha256": %q}
}`, version, ianaDigest, strings.Repeat("1", 64))
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile(lock): %v", err)
	}
	return path
}
