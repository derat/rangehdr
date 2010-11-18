// Copyright 2010 Daniel Erat <dan@erat.org>
// All rights reserved.

package rangehdr

import (
	"strconv"
	"strings"
	"testing"
)

func TestParseRangeHeader(t *testing.T) {
	// Invalid because it's missing bytes-unit.
	checkParseRangeHeader(t, "0-5", 10, false, "")

	// Invalid because last-byte-pos is less than first-byte-pos.
	checkParseRangeHeader(t, "bytes=5-4", 10, false, "")

	// Invalid because there's a syntactically invalid byte-range spec, even though there's a valid one first.
	checkParseRangeHeader(t, "bytes=0-2,5-4", 10, false, "")

	// Valid with first-byte-pos and last-byte-pos both supplied.
	checkParseRangeHeader(t, "bytes=0-9", 10, true, "0:10")

	// Valid with missing last-byte-pos.
	checkParseRangeHeader(t, "bytes=0-", 10, true, "0:10")

	// Valid with last-byte-pos extending beyond the end of the file.
	checkParseRangeHeader(t, "bytes=0-20", 10, true, "0:10")

	// Unsatisfiable, with a range that begins after the end of the file.
	checkParseRangeHeader(t, "bytes=15-,0-5", 10, true, "")

	// Valid, with a suffix-length fitting within the file.
	checkParseRangeHeader(t, "bytes=-5", 10, true, "5:5")

	// Valid, with a suffix-length extending beyond the beginning of the file.
	checkParseRangeHeader(t, "bytes=-15", 10, true, "0:10")

	// Examples from 14.35.1 of RFC 2616:
	checkParseRangeHeader(t, "bytes=0-499", 10000, true, "0:500")
	checkParseRangeHeader(t, "bytes=500-999", 10000, true, "500:500")
	checkParseRangeHeader(t, "bytes=-500", 10000, true, "9500:500")
	checkParseRangeHeader(t, "bytes=9500-", 10000, true, "9500:500")
	checkParseRangeHeader(t, "bytes=0-0,-1", 10000, true, "0:1,9999:1")
	checkParseRangeHeader(t, "bytes=500-600,601-999", 10000, true, "500:101,601:399")
	checkParseRangeHeader(t, "bytes=500-700,601-999", 10000, true, "500:201,601:399")
}

func TestJoinByteRanges(t *testing.T) {
	checkJoinByteRanges(t, "0:100", "0:100")
	checkJoinByteRanges(t, "0:10,10:10", "0:20")
	checkJoinByteRanges(t, "0:10,5:10", "0:15")
	checkJoinByteRanges(t, "10:10,5:10", "5:15")
	checkJoinByteRanges(t, "0:10,11:10", "0:10")
	checkJoinByteRanges(t, "5:10,3:1", "5:10")
	checkJoinByteRanges(t, "5:10,15:1,4:1", "4:12")
	checkJoinByteRanges(t, "5:10,4:12", "4:12")
	checkJoinByteRanges(t, "4:12,5:10", "4:12")
}

func convertByteRangesToString(ranges []ByteRange) string {
	var rangesStr string
	for i, byteRange := range ranges {
		if i != 0 {
			rangesStr += ","
		}
		rangesStr += byteRange.String()
	}
	return rangesStr
}

// Takes the header and file size to use as input, the header's expected validity, and a string describing the expected
// comma-separated offset:length ranges (e.g. "0:10,20:5").
func checkParseRangeHeader(t *testing.T, header string, fileLength int64, expectedValid bool, expectedRanges string) {
	ranges, err := parseRangeHeader(header, fileLength)
	if (err == nil) != expectedValid {
		t.Errorf("Range header \"%s\": expected validity %v, actual validity %v\n", header, expectedValid, (err == nil))
		return
	}

	actualRanges := convertByteRangesToString(ranges)
	if actualRanges != expectedRanges {
		t.Errorf("Range header \"%s\": expected ranges %s, actual ranges %s\n", header, expectedRanges, actualRanges)
		return
	}
}

func checkJoinByteRanges(t *testing.T, inputRangesStr string, expectedRangeStr string) {
	inputRanges := make([]ByteRange, 0, 10)
	appendRange := func(offset int64, length int64) {
		inputRanges = inputRanges[0:len(inputRanges)+1]
		inputRanges[len(inputRanges)-1] = ByteRange{offset, length}
	}

	for _, rangeStr := range strings.Split(inputRangesStr, ",", -1) {
		//print("Got \"", rangeStr, "\" from input \"", inputRangesStr, "\"\n")
		if len(rangeStr) == 0 {
			// Uh, this is stupid.
			continue
		}
		parts := strings.Split(rangeStr, ":", 2)
		offset, _ := strconv.Atoi64(parts[0])
		length, _ := strconv.Atoi64(parts[1])
		appendRange(offset, length)
	}
	actualRange := joinByteRanges(inputRanges)
	actualRangeStr := actualRange.String()
	if actualRangeStr != expectedRangeStr {
		t.Errorf("Ranges \"%s\": expected compression to %s, actual %s\n", inputRangesStr, expectedRangeStr, actualRangeStr)
		return
	}
}
