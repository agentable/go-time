package genlock

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndVerify(t *testing.T) {
	t.Parallel()

	input := []byte("source data\n")
	digest := fmt.Sprintf("%x", sha256.Sum256(input))
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	lockPath := filepath.Join(dir, "sources.json")
	lockJSON := fmt.Sprintf(`{
  "iana": {"version": "2025b", "url": "https://example.test/zone.tab", "sha256": %q},
  "cldr_windows": {"version": "release-48-1", "url": "https://example.test/windowsZones.xml", "sha256": %q}
}`, digest, digest)
	if err := os.WriteFile(lockPath, []byte(lockJSON), 0o600); err != nil {
		t.Fatalf("WriteFile(lock): %v", err)
	}

	lock, err := Load(lockPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if lock.IANA.Version != "2025b" || lock.CLDRWindows.Version != "release-48-1" {
		t.Fatalf("Load() = %+v, want locked versions", lock)
	}
	if lock.IANA.Filename() != "zone.tab" || lock.CLDRWindows.Filename() != "windowsZones.xml" {
		t.Fatalf("Load() filenames = %q, %q, want locked URL basenames", lock.IANA.Filename(), lock.CLDRWindows.Filename())
	}
	if err := lock.IANA.Verify(inputPath); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if err := os.WriteFile(inputPath, []byte("changed\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(changed input): %v", err)
	}
	err = lock.IANA.Verify(inputPath)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("Verify() error = %v, want SHA-256 mismatch", err)
	}
}

func TestLoadAndVerifyMissingFiles(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := Load(missing); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load(missing) error = %v, want fs.ErrNotExist", err)
	}
	if err := (Source{SHA256: strings.Repeat("0", 64)}).Verify(missing); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Verify(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestLoadRejectsInvalidLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed JSON", body: `{`, want: "decode"},
		{name: "unknown field", body: `{"other": {}}`, want: "unknown"},
		{name: "missing source", body: `{"iana": {"version": "2025b", "url": "https://example.test/zone.tab", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`, want: "cldr_windows"},
		{name: "missing version", body: validLockJSON("", strings.Repeat("a", 64)), want: "version"},
		{name: "non HTTPS URL", body: strings.Replace(validLockJSON("2025b", strings.Repeat("a", 64)), "https://example.test/zone.tab", "http://example.test/zone.tab", 1), want: "https"},
		{name: "URL without source file", body: strings.Replace(validLockJSON("2025b", strings.Repeat("a", 64)), "https://example.test/zone.tab", "https://example.test/", 1), want: "source file"},
		{name: "invalid SHA-256 length", body: validLockJSON("2025b", "not-a-digest"), want: "sha256"},
		{name: "invalid SHA-256 hexadecimal", body: validLockJSON("2025b", strings.Repeat("g", 64)), want: "sha256"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "sources.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatalf("WriteFile(): %v", err)
			}
			_, err := Load(path)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("Load() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLoadAcceptsUppercaseSHA256(t *testing.T) {
	t.Parallel()

	input := []byte("source data\n")
	digest := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256(input)))
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	lockPath := filepath.Join(dir, "sources.json")
	if err := os.WriteFile(lockPath, []byte(validLockJSON("2025b", digest)), 0o600); err != nil {
		t.Fatalf("WriteFile(lock): %v", err)
	}
	lock, err := Load(lockPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := lock.IANA.Verify(inputPath); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func validLockJSON(ianaVersion, digest string) string {
	return fmt.Sprintf(`{
  "iana": {"version": %q, "url": "https://example.test/zone.tab", "sha256": %q},
  "cldr_windows": {"version": "release-48-1", "url": "https://example.test/windowsZones.xml", "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
}`, ianaVersion, digest)
}
