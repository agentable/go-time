// Package genlock verifies the upstream files used by timezone generators.
package genlock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"strings"

	"github.com/go-json-experiment/json"
)

// Source identifies one generator input.
type Source struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// Lock identifies every upstream timezone generator input.
type Lock struct {
	IANA        Source `json:"iana"`
	CLDRWindows Source `json:"cldr_windows"`
}

// Load reads a source lock.
func Load(path string) (Lock, error) {
	data, err := os.ReadFile(path) //nolint:gosec // The maintainer explicitly selects the source lock.
	if err != nil {
		return Lock{}, fmt.Errorf("read source lock: %w", err)
	}
	var lock Lock
	if err := json.Unmarshal(data, &lock, json.RejectUnknownMembers(true)); err != nil {
		return Lock{}, fmt.Errorf("decode source lock: %w", err)
	}
	if err := lock.IANA.validate("iana"); err != nil {
		return Lock{}, err
	}
	if err := lock.CLDRWindows.validate("cldr_windows"); err != nil {
		return Lock{}, err
	}
	return lock, nil
}

// Verify checks path against the locked SHA-256.
func (s Source) Verify(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // The maintainer explicitly selects the generator input.
	if err != nil {
		return fmt.Errorf("read locked source: %w", err)
	}
	got := fmt.Sprintf("%x", sha256.Sum256(data))
	if !strings.EqualFold(got, s.SHA256) {
		return fmt.Errorf("source SHA-256 = %s, want %s", got, s.SHA256)
	}
	return nil
}

// Filename returns the canonical source filename from the locked URL.
func (s Source) Filename() string {
	u, _ := url.Parse(s.URL)
	return pathpkg.Base(u.Path)
}

func (s Source) validate(name string) error {
	if s.Version == "" {
		return fmt.Errorf("source lock %s version is required", name)
	}
	u, err := url.Parse(s.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("source lock %s URL must be an absolute https URL", name)
	}
	if u.Path == "" || strings.HasSuffix(u.Path, "/") || pathpkg.Base(u.Path) == "." {
		return fmt.Errorf("source lock %s URL must identify a source file", name)
	}
	if len(s.SHA256) != sha256.Size*2 {
		return fmt.Errorf("source lock %s sha256 must contain 64 hexadecimal characters", name)
	}
	if _, err := hex.DecodeString(s.SHA256); err != nil {
		return fmt.Errorf("source lock %s sha256 is invalid: %w", name, err)
	}
	return nil
}
