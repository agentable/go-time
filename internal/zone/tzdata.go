package zone

import (
	"archive/zip"
	_ "embed"
	"strings"
	"sync"
)

// See zoneinfo.md for provenance and update instructions.
//
//go:embed zoneinfo.zip
var bundledTZif string

var bundledArchive = sync.OnceValues(func() (*zip.Reader, error) {
	return zip.NewReader(strings.NewReader(bundledTZif), int64(len(bundledTZif)))
})

func readBundledZone(name string) ([]byte, error) {
	archive, err := bundledArchive()
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
