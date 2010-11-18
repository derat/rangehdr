// Copyright 2010 Daniel Erat <dan@erat.org>
// All rights reserved.

package rangehdr

import (
	"math"
	"os"
	"strconv"
	"strings"
)

type ByteRange struct {
	offset, length int64
}

func (bytes *ByteRange) String() string {
	return strconv.Itoa64(bytes.offset) + ":" + strconv.Itoa64(bytes.length)
}

// Parse the value from a Range: header into a slice of ByteRange objects.
// Returns an error if the header was syntatically invalid (Range: header should be ignored), and an empty slice if the request was unsatisfiable (416).
func parseRangeHeader(header string, fileLength int64) ([]ByteRange, os.Error) {
	ranges := make([]ByteRange, 0, 10)
	appendRange := func(offset int64, length int64) {
		ranges = ranges[0:len(ranges)+1]
		ranges[len(ranges)-1] = ByteRange{offset, length}
	}

	const bytesPrefix = "bytes="
	if !strings.HasPrefix(header, bytesPrefix) {
		return nil, os.NewError("Range header \"" + header + "\" is missing bytes-unit\n")
	}

	for _, rangeString := range strings.Split(header[len(bytesPrefix):len(header)], ",", -1) {
		if parts := strings.Split(rangeString, "-", -1); len(parts) == 2 {
			if len(parts[0]) == 0 {
				endingBytes, e := strconv.Atoi64(rangeString[1:len(rangeString)])
				if e != nil || endingBytes < 0 {
					return nil, os.NewError("suffix-length not parseable in range \"" +  rangeString + "\"\n")
				}
				if endingBytes == 0 {
					// Unsatisfiable: this suffix-byte-range-spec has a zero suffix-length.
					return nil, nil
				}
				if endingBytes > fileLength {
					endingBytes = fileLength
				}
				appendRange(fileLength - endingBytes, endingBytes)
				continue
			}

			startByte, e := strconv.Atoi64(parts[0])
			if e != nil {
				return nil, os.NewError("Unable to parse first-byte-pos in range \"" + rangeString + "\"\n")
			}
			var endByte int64 = math.MaxInt64
			if len(parts[1]) > 0 {
				endByte, e = strconv.Atoi64(parts[1])
				if e != nil {
					return nil, os.NewError("Unable to parse last-byte-pos in range \"" + rangeString + "\"\n")
				}
			}
			if endByte < startByte {
				return nil, os.NewError("last-byte-pos precedes first-byte-pos in range \"" + rangeString + "\"\n")
			}
			if startByte >= fileLength {
				// Unsatisfiable: first-byte-pos of byte-range-set is >= the length of the entity-body.
				return nil, nil
			}
			if (endByte >= fileLength) {
				endByte = fileLength - 1
			}
			appendRange(startByte, endByte - startByte + 1)
		} else {
			return nil, os.NewError("Invalid range \"" + rangeString + "\"\n")
		}
	}
	return ranges, nil
}

// Naively compress a slice of ByteRanges into a single ByteRange.
// We join consecutive ranges together if they're adjacent or overlapping.
// Any range that can't be joined to the previously-built-up range is dropped.
func joinByteRanges(ranges []ByteRange) ByteRange {
	if len(ranges) == 1 {
		return ranges[0]
	}

	totalRange := ranges[0]
	end := totalRange.offset + totalRange.length
	for _, thisRange := range ranges[1:len(ranges)] {
		if thisRange.offset < totalRange.offset && thisRange.offset + thisRange.length >= totalRange.offset {
			totalRange.offset = thisRange.offset
			totalRange.length = end - totalRange.offset
		}
		if thisRange.offset <= end && thisRange.offset + thisRange.length > end {
			end = thisRange.offset + thisRange.length
			totalRange.length = end - totalRange.offset
		}
	}
	return totalRange
}
