package duplication

import (
	"bufio"
	"hash/fnv"
	"os"
	"sort"
	"strings"

	"github.com/vettcode/scanner/internal/walker"
)

// Result holds the duplication detection results.
type Result struct {
	DuplicationPct    float64
	DuplicatedLOC     int
	TotalLOC          int
	DuplicateBlocks   int
}

// minBlockLines is the minimum number of lines for a duplicate block.
const minBlockLines = 6

// windowSize is the number of lines in a rolling hash window.
const windowSize = 6

// Analyze detects code duplication across files using line-hash fingerprinting.
func Analyze(files []walker.FileInfo) *Result {
	r := &Result{}

	// Step 1: Build a hash-to-locations index using rolling line windows
	type location struct {
		fileIdx int
		lineNum int
	}

	hashIndex := make(map[uint64][]location)
	allLines := make([][]string, len(files))

	for i, f := range files {
		if f.IsTest {
			continue // skip test files for duplication
		}
		lines := readNormalizedLines(f.Path)
		allLines[i] = lines
		r.TotalLOC += len(lines)

		if len(lines) < windowSize {
			continue
		}

		for j := 0; j <= len(lines)-windowSize; j++ {
			h := hashWindow(lines[j : j+windowSize])
			hashIndex[h] = append(hashIndex[h], location{fileIdx: i, lineNum: j})
		}
	}

	// Step 2: Find hashes that appear 2+ times (potential duplicates)
	duplicatedLines := make(map[int]map[int]bool) // fileIdx -> lineNums

	for _, locs := range hashIndex {
		if len(locs) < 2 {
			continue
		}

		// Check that at least 2 locations are in different files or non-overlapping
		hasDuplicate := false
		for i := 0; i < len(locs) && !hasDuplicate; i++ {
			for j := i + 1; j < len(locs); j++ {
				if locs[i].fileIdx != locs[j].fileIdx ||
					abs(locs[i].lineNum-locs[j].lineNum) >= windowSize {
					hasDuplicate = true
					break
				}
			}
		}

		if !hasDuplicate {
			continue
		}

		// Mark all lines in this window as duplicated (except the first occurrence)
		for idx := 1; idx < len(locs); idx++ {
			loc := locs[idx]
			if duplicatedLines[loc.fileIdx] == nil {
				duplicatedLines[loc.fileIdx] = make(map[int]bool)
			}
			for k := loc.lineNum; k < loc.lineNum+windowSize; k++ {
				duplicatedLines[loc.fileIdx][k] = true
			}
		}
	}

	// Step 3: Count duplicated LOC and merge adjacent lines into blocks
	for _, lineSet := range duplicatedLines {
		r.DuplicatedLOC += len(lineSet)

		// Count contiguous blocks: sort line numbers and count runs
		if len(lineSet) > 0 {
			lines := make([]int, 0, len(lineSet))
			for l := range lineSet {
				lines = append(lines, l)
			}
			sortInts(lines)
			blocks := 1
			for i := 1; i < len(lines); i++ {
				if lines[i] != lines[i-1]+1 {
					blocks++
				}
			}
			r.DuplicateBlocks += blocks
		}
	}

	if r.TotalLOC > 0 {
		r.DuplicationPct = float64(r.DuplicatedLOC) / float64(r.TotalLOC) * 100.0
	}

	return r
}

// readNormalizedLines reads a file and returns normalized lines (trimmed, no blanks).
func readNormalizedLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Normalize: lowercase, collapse whitespace
		line = strings.Join(strings.Fields(line), " ")
		lines = append(lines, line)
	}
	// Partial results returned even on scanner error (e.g., line too long)
	_ = scanner.Err()
	return lines
}

// hashWindow computes a hash of a window of lines.
func hashWindow(lines []string) uint64 {
	h := fnv.New64a()
	for _, line := range lines {
		h.Write([]byte(line))
		h.Write([]byte{0})
	}
	return h.Sum64()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sortInts(a []int) {
	sort.Ints(a)
}
