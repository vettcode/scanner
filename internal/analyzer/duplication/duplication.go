package duplication

import (
	"bufio"
	"hash/fnv"
	"os"
	"sort"
	"strings"

	"github.com/vettcode/scanner/internal/exclusion"
	"github.com/vettcode/scanner/internal/walker"
)

// Token represents a normalized token with its source line number.
type Token struct {
	Value string
	Line  int // 1-based
}

// Result holds the duplication detection results.
type Result struct {
	DuplicationPct  float64
	DuplicatedLOC   int
	TotalLOC        int
	DuplicateBlocks int
}

const (
	tokenWindowSize    = 100    // Rabin-Karp window for token-based detection
	lineWindowSize     = 10     // rolling window for line-hash fallback
	minBlockLines      = 10     // minimum lines for a duplicate block
	hashBase           = 1000000007
	samplingThreshold  = 300000 // LOC threshold above which sampling kicks in
)

// Analyze detects code duplication. Uses token-based Rabin-Karp for files with
// token data (Tier 1 languages), falls back to line-hash for others (Tier 2).
// Pass nil tokenStreams for pure line-hash mode.
//
// For repos > 300K LOC, files are sampled (every Nth file) to bound memory
// and runtime. The result is extrapolated to estimate total duplication.
func Analyze(files []walker.FileInfo, tokenStreams ...map[string][]Token) *Result {
	r := &Result{}

	var ts map[string][]Token
	if len(tokenStreams) > 0 {
		ts = tokenStreams[0]
	}

	// Count total LOC (including all files, before sampling)
	totalLOC := 0
	var nonTestFiles []walker.FileInfo
	for _, f := range files {
		if f.IsTest || exclusion.IsAuxiliaryPath(f.RelPath) {
			continue
		}
		totalLOC += f.LOC
		nonTestFiles = append(nonTestFiles, f)
	}
	r.TotalLOC = totalLOC

	// Sampling: for repos > 300K LOC, take every Nth file
	sampleStep := 1
	if totalLOC > samplingThreshold && len(nonTestFiles) > 0 {
		sampleStep = totalLOC / samplingThreshold
		if sampleStep < 2 {
			sampleStep = 2
		}
	}

	var lineFiles []walker.FileInfo
	var tokenFiles []tokFileEntry
	sampledLOC := 0

	for i, f := range nonTestFiles {
		if sampleStep > 1 && i%sampleStep != 0 {
			continue
		}
		sampledLOC += f.LOC
		if tokens, ok := ts[f.Path]; ok && len(tokens) >= tokenWindowSize {
			tokenFiles = append(tokenFiles, tokFileEntry{tokens: tokens, loc: f.LOC})
		} else {
			lineFiles = append(lineFiles, f)
		}
	}

	// Token-based duplication (Tier 1)
	tDupLOC, tBlocks := detectTokenDuplication(tokenFiles)

	// Line-based duplication (Tier 2 fallback)
	lDupLOC, lBlocks := detectLineDuplication(lineFiles)

	rawDupLOC := tDupLOC + lDupLOC
	rawBlocks := tBlocks + lBlocks

	// If sampling was used, compute duplication percentage from the sample
	// and apply it to total LOC for the extrapolated count.
	if sampleStep > 1 && sampledLOC > 0 {
		samplePct := float64(rawDupLOC) / float64(sampledLOC) * 100.0
		r.DuplicationPct = samplePct
		r.DuplicatedLOC = int(samplePct / 100.0 * float64(totalLOC))
		r.DuplicateBlocks = rawBlocks * sampleStep // rough extrapolation
	} else {
		r.DuplicatedLOC = rawDupLOC
		r.DuplicateBlocks = rawBlocks
		if r.TotalLOC > 0 {
			r.DuplicationPct = float64(r.DuplicatedLOC) / float64(r.TotalLOC) * 100.0
		}
	}

	return r
}

// --- Token-based Rabin-Karp duplication detection ---

type tokFileEntry struct {
	tokens []Token
	loc    int
}

func detectTokenDuplication(files []tokFileEntry) (duplicatedLOC, blocks int) {
	if len(files) == 0 {
		return 0, 0
	}

	// Step 1: Map unique token strings to integer IDs
	tokenIDs := make(map[string]uint64)
	nextID := uint64(1)

	type fileData struct {
		ids   []uint64
		lines []int
	}
	allFiles := make([]fileData, len(files))

	for fi, f := range files {
		fd := fileData{
			ids:   make([]uint64, len(f.tokens)),
			lines: make([]int, len(f.tokens)),
		}
		for i, t := range f.tokens {
			id, ok := tokenIDs[t.Value]
			if !ok {
				id = nextID
				tokenIDs[t.Value] = id
				nextID++
			}
			fd.ids[i] = id
			fd.lines[i] = t.Line
		}
		allFiles[fi] = fd
	}

	// Step 2: Rabin-Karp rolling hash over each file's token ID stream
	type windowLoc struct {
		fileIdx   int
		tokenIdx  int
		startLine int
		endLine   int
	}
	hashIndex := make(map[uint64][]windowLoc)

	// Precompute base^windowSize (uint64 overflow acts as mod 2^64)
	basePow := uint64(1)
	for i := 0; i < tokenWindowSize; i++ {
		basePow *= hashBase
	}

	for fi, fd := range allFiles {
		n := len(fd.ids)
		if n < tokenWindowSize {
			continue
		}

		// Initial hash: H = id[0]*B^(w-1) + id[1]*B^(w-2) + ... + id[w-1]
		h := uint64(0)
		for i := 0; i < tokenWindowSize; i++ {
			h = h*hashBase + fd.ids[i]
		}
		hashIndex[h] = append(hashIndex[h], windowLoc{
			fileIdx: fi, tokenIdx: 0,
			startLine: fd.lines[0], endLine: fd.lines[tokenWindowSize-1],
		})

		// Roll: H' = H*B - id[old]*B^w + id[new]
		for i := 1; i <= n-tokenWindowSize; i++ {
			h = h*hashBase - fd.ids[i-1]*basePow + fd.ids[i+tokenWindowSize-1]
			hashIndex[h] = append(hashIndex[h], windowLoc{
				fileIdx: fi, tokenIdx: i,
				startLine: fd.lines[i], endLine: fd.lines[i+tokenWindowSize-1],
			})
		}
	}

	// Step 3: Find duplicate windows (cross-file or non-overlapping in same file)
	duplicatedLines := make(map[int]map[int]bool)

	for _, locs := range hashIndex {
		if len(locs) < 2 {
			continue
		}

		// Need at least two non-overlapping locations
		hasDup := false
		for i := 0; i < len(locs) && !hasDup; i++ {
			for j := i + 1; j < len(locs); j++ {
				if locs[i].fileIdx != locs[j].fileIdx ||
					abs(locs[i].tokenIdx-locs[j].tokenIdx) >= tokenWindowSize {
					hasDup = true
					break
				}
			}
		}
		if !hasDup {
			continue
		}

		// Verify first pair matches (guard against hash collisions)
		ref := locs[0]
		fdRef := allFiles[ref.fileIdx]

		var verified []windowLoc
		verified = append(verified, ref)

		for idx := 1; idx < len(locs); idx++ {
			loc := locs[idx]
			if loc.fileIdx == ref.fileIdx && abs(loc.tokenIdx-ref.tokenIdx) < tokenWindowSize {
				continue // overlapping in same file
			}
			fdLoc := allFiles[loc.fileIdx]
			match := true
			for k := 0; k < tokenWindowSize; k++ {
				if fdRef.ids[ref.tokenIdx+k] != fdLoc.ids[loc.tokenIdx+k] {
					match = false
					break
				}
			}
			if match {
				verified = append(verified, loc)
			}
		}

		if len(verified) < 2 {
			continue
		}

		// Mark ALL verified locations as duplicated (including first copy)
		for _, loc := range verified {
			if duplicatedLines[loc.fileIdx] == nil {
				duplicatedLines[loc.fileIdx] = make(map[int]bool)
			}
			for line := loc.startLine; line <= loc.endLine; line++ {
				duplicatedLines[loc.fileIdx][line] = true
			}
		}
	}

	// Step 4: Merge into contiguous blocks, filter blocks < minBlockLines
	duplicatedLOC, blocks = countBlocksAndLOC(duplicatedLines)
	return
}

// --- Line-hash duplication detection (Tier 2 fallback) ---

func detectLineDuplication(files []walker.FileInfo) (duplicatedLOC, blocks int) {
	if len(files) == 0 {
		return 0, 0
	}

	type location struct {
		fileIdx int
		lineNum int
	}
	hashIndex := make(map[uint64][]location)

	for i, f := range files {
		lines := readNormalizedLines(f.Path)
		if len(lines) < lineWindowSize {
			continue
		}
		for j := 0; j <= len(lines)-lineWindowSize; j++ {
			h := hashLineWindow(lines[j : j+lineWindowSize])
			hashIndex[h] = append(hashIndex[h], location{fileIdx: i, lineNum: j})
		}
	}

	duplicatedLines := make(map[int]map[int]bool)

	for _, locs := range hashIndex {
		if len(locs) < 2 {
			continue
		}

		hasDup := false
		for i := 0; i < len(locs) && !hasDup; i++ {
			for j := i + 1; j < len(locs); j++ {
				if locs[i].fileIdx != locs[j].fileIdx ||
					abs(locs[i].lineNum-locs[j].lineNum) >= lineWindowSize {
					hasDup = true
					break
				}
			}
		}
		if !hasDup {
			continue
		}

		// Mark ALL locations as duplicated
		for _, loc := range locs {
			if duplicatedLines[loc.fileIdx] == nil {
				duplicatedLines[loc.fileIdx] = make(map[int]bool)
			}
			for k := loc.lineNum; k < loc.lineNum+lineWindowSize; k++ {
				duplicatedLines[loc.fileIdx][k] = true
			}
		}
	}

	duplicatedLOC, blocks = countBlocksAndLOC(duplicatedLines)
	return
}

// --- Shared helpers ---

// countBlocksAndLOC merges duplicated lines into contiguous blocks,
// filters blocks < minBlockLines, and returns total LOC and block count.
func countBlocksAndLOC(duplicatedLines map[int]map[int]bool) (totalLOC, totalBlocks int) {
	for _, lineSet := range duplicatedLines {
		lines := make([]int, 0, len(lineSet))
		for l := range lineSet {
			lines = append(lines, l)
		}
		sort.Ints(lines)

		blockStart := 0
		for i := 1; i <= len(lines); i++ {
			if i == len(lines) || lines[i] != lines[i-1]+1 {
				blockLen := i - blockStart
				if blockLen >= minBlockLines {
					totalBlocks++
					totalLOC += blockLen
				}
				blockStart = i
			}
		}
	}
	return
}

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
		line = strings.Join(strings.Fields(line), " ")
		lines = append(lines, line)
	}
	_ = scanner.Err()
	return lines
}

func hashLineWindow(lines []string) uint64 {
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
