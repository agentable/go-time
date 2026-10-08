package zone

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const maxTZifSize = 10 << 20

var zoneinfo = sync.OnceValue(func() string { return os.Getenv("ZONEINFO") })

// Load reads one TZif snapshot for both candidate discovery and projection.
func Load(name string) (*Rules, error) {
	if name == "UTC" {
		return nil, nil
	}
	if name == "" || name == "Local" || !fs.ValidPath(name) || strings.ContainsAny(name, "\\:") || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid IANA zone name %q", name)
	}
	return loadSources(name, []string{
		zoneinfo(),
		"/usr/share/zoneinfo", "/usr/share/lib/zoneinfo", "/usr/lib/locale/TZ", "/etc/zoneinfo",
	})
}

func loadSources(name string, sources []string) (*Rules, error) {
	var firstErr error
	for _, source := range sources {
		if source == "" {
			continue
		}
		data, err := readZone(source, name)
		if err == nil {
			var rules *Rules
			rules, err = loadRules(name, data)
			if err == nil {
				return rules, nil
			}
		}
		if firstErr == nil && !errors.Is(err, fs.ErrNotExist) {
			firstErr = err
		}
	}
	data, err := readBundledZone(name)
	if err == nil {
		return loadRules(name, data)
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, fmt.Errorf("load time zone %q: %w", name, err)
}

func readZone(source, name string) ([]byte, error) {
	if strings.HasSuffix(source, ".zip") {
		archive, err := zip.OpenReader(source)
		if archive != nil {
			defer func() { _ = archive.Close() }()
		}
		if err != nil {
			return nil, err
		}
		file, err := archive.Open(name)
		if err != nil {
			return nil, err
		}
		defer func() { _ = file.Close() }()
		return readTZif(file)
	}
	file, err := os.Open(filepath.Join(source, name)) //nolint:gosec // Source is trusted configuration; Load validates the relative IANA name.
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return readTZif(file)
}

func readTZif(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxTZifSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTZifSize {
		return nil, fmt.Errorf("TZif data exceeds %d bytes", maxTZifSize)
	}
	return data, nil
}
