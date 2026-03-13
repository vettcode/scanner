package filesize

import "github.com/vettcode/scanner/internal/walker"

// Distribution holds the file size distribution analysis.
type Distribution struct {
	// FilesOver500LOC is the count of files with more than 500 lines of code.
	FilesOver500LOC int
	// PctOver500LOC is the percentage of files over 500 LOC.
	PctOver500LOC float64
	// Buckets holds file count per LOC range.
	Buckets map[string]int
	// LargestFile is the LOC count of the largest file.
	LargestFile int
	// TotalFiles is the number of files analyzed.
	TotalFiles int
}

// Analyze computes file size distribution from walk results.
func Analyze(files []walker.FileInfo) *Distribution {
	d := &Distribution{
		Buckets: map[string]int{
			"0-100":    0,
			"101-300":  0,
			"301-500":  0,
			"501-1000": 0,
			"1001+":    0,
		},
	}

	for _, f := range files {
		if f.IsTest {
			continue // only source files for file size scoring
		}
		d.TotalFiles++

		if f.LOC > d.LargestFile {
			d.LargestFile = f.LOC
		}

		switch {
		case f.LOC <= 100:
			d.Buckets["0-100"]++
		case f.LOC <= 300:
			d.Buckets["101-300"]++
		case f.LOC <= 500:
			d.Buckets["301-500"]++
		case f.LOC <= 1000:
			d.Buckets["501-1000"]++
			d.FilesOver500LOC++
		default:
			d.Buckets["1001+"]++
			d.FilesOver500LOC++
		}
	}

	if d.TotalFiles > 0 {
		d.PctOver500LOC = float64(d.FilesOver500LOC) / float64(d.TotalFiles) * 100.0
	}

	return d
}
