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

	dir := t.TempDir()
	input := []byte("US\t+404251-0740023\tAmerica/New_York\n")
	zoneTabs := []string{
		filepath.Join(dir, "first-local-name.txt"),
		filepath.Join(dir, "second-local-name.txt"),
	}
	for _, path := range zoneTabs {
		if err := os.WriteFile(path, input, 0o600); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(input))

	t.Run("verified input", func(t *testing.T) {
		lockPath := writeLock(t, dir, digest)
		generated := make([][]byte, 0, len(zoneTabs))
		for i, zoneTab := range zoneTabs {
			out := filepath.Join(dir, fmt.Sprintf("catalog-%d.go", i))
			if err := run(zoneTab, lockPath, out); err != nil {
				t.Fatalf("run(%s) error = %v", zoneTab, err)
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
		if !strings.Contains(string(generated[0]), "IANA tzdb 2025b zone.tab") {
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
		err := run(zoneTabs[0], lockPath, out)
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

func writeLock(t *testing.T, dir, ianaDigest string) string {
	t.Helper()

	path := filepath.Join(dir, "sources-"+ianaDigest[:8]+".json")
	body := fmt.Sprintf(`{
  "iana": {"version": "2025b", "url": "https://example.test/zone.tab", "sha256": %q},
  "cldr_windows": {"version": "release-48-1", "url": "https://example.test/windowsZones.xml", "sha256": %q}
}`, ianaDigest, strings.Repeat("1", 64))
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile(lock): %v", err)
	}
	return path
}
