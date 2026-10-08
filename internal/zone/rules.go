package zone

import (
	"encoding/binary"
	"errors"
	"slices"
	"strings"
	"time"
)

// Rules holds one immutable snapshot of a location and its possible offsets.
// A nil Rules represents UTC.
type Rules struct {
	location *time.Location
	offsets  []int
}

// Location returns the stdlib projection of the held rules.
func (r *Rules) Location() *time.Location {
	if r == nil {
		return time.UTC
	}
	return r.location
}

var errInvalidTZif = errors.New("invalid TZif data")

func loadRules(name string, data []byte) (*Rules, error) {
	offsets, err := tzifOffsets(data)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocationFromTZData(name, data)
	if err != nil {
		return nil, err
	}
	slices.Sort(offsets)
	return &Rules{location: loc, offsets: slices.Compact(offsets)}, nil
}

// tzifOffsets reads only offset metadata; stdlib owns transition evaluation.
func tzifOffsets(data []byte) ([]int, error) {
	if len(data) < 44 || string(data[:4]) != "TZif" {
		return nil, errInvalidTZif
	}
	version := data[4]
	if version != 0 && version != '2' && version != '3' {
		return nil, errInvalidTZif
	}
	block, rest, err := tzifBlock(data, 4)
	if err != nil {
		return nil, err
	}
	if version != 0 {
		block, rest, err = tzifBlock(rest, 8)
		if err != nil {
			return nil, err
		}
	}
	offsets := make([]int, 0, len(block)/6)
	for len(block) >= 6 {
		// TZif stores UTC offsets as signed, two's-complement 32-bit seconds.
		offsets = append(offsets, int(int32(binary.BigEndian.Uint32(block)))) //nolint:gosec // Intentional signed reinterpretation of TZif int32.
		block = block[6:]
	}
	// Match the footer envelope accepted by LoadLocationFromTZData, including
	// v1 files with an extension. Extra inactive offsets are safe: resolution
	// always checks the actual stdlib projection before accepting a candidate.
	if len(rest) > 2 && rest[0] == '\n' && rest[len(rest)-1] == '\n' {
		offsets = append(offsets, footerOffsets(string(rest[1:len(rest)-1]))...)
	}
	return offsets, nil
}

// tzifBlock bounds every count before either reader allocates from input.
// It returns local-time type records and the bytes following the entire block.
func tzifBlock(data []byte, width uint64) (types, rest []byte, err error) {
	if len(data) < 44 {
		return nil, nil, errInvalidTZif
	}
	var n [6]uint64
	for i := range n {
		n[i] = uint64(binary.BigEndian.Uint32(data[20+i*4:]))
	}
	start := 44 + n[3]*(width+1)
	end := start + n[4]*6
	total := end + n[5] + n[2]*(width+4) + n[1] + n[0]
	if n[4] == 0 || total > uint64(len(data)) {
		return nil, nil, errInvalidTZif
	}
	return data[start:end], data[total:], nil
}

// footerOffsets reads the POSIX std/DST prefix, not the transition rules.
// Its accepted prefix covers time.tzset; invalid trailing rules merely leave
// unused candidate offsets, which the stdlib forward check rejects.
func footerOffsets(s string) []int {
	s, ok := skipTZName(s)
	if !ok {
		return nil
	}
	standard, s, ok := readTZOffset(s)
	if !ok {
		return nil
	}
	// POSIX offsets are added to local time to obtain UTC.
	offsets := []int{-standard}
	if s == "" || s[0] == ',' {
		return offsets
	}
	s, ok = skipTZName(s)
	if !ok {
		return offsets
	}
	if s == "" || s[0] == ',' {
		return append(offsets, -standard+3600)
	}
	daylight, _, ok := readTZOffset(s)
	if ok {
		offsets = append(offsets, -daylight)
	}
	return offsets
}

func skipTZName(s string) (string, bool) {
	if strings.HasPrefix(s, "<") {
		end := strings.IndexByte(s, '>')
		if end < 0 {
			return "", false
		}
		return s[end+1:], true
	}
	end := strings.IndexAny(s, "0123456789,-+")
	if end < 0 {
		return "", len(s) >= 3
	}
	return s[end:], end >= 3
}

func readTZOffset(s string) (int, string, bool) {
	sign := 1
	if s != "" && (s[0] == '+' || s[0] == '-') {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	var offset int
	for i, limit := range []int{168, 59, 59} {
		value, digits := 0, 0
		for digits < len(s) && s[digits] >= '0' && s[digits] <= '9' {
			value = value*10 + int(s[digits]-'0')
			if value > limit {
				return 0, "", false
			}
			digits++
		}
		if digits == 0 {
			return 0, "", false
		}
		s = s[digits:]
		switch i {
		case 0:
			offset += value * 3600
		case 1:
			offset += value * 60
		case 2:
			offset += value
		}
		if i == 2 || s == "" || s[0] != ':' {
			return sign * offset, s, true
		}
		s = s[1:]
	}
	return 0, "", false
}
