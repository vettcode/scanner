package output

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// ScanPhase represents the current phase of the scan.
type ScanPhase string

const (
	PhaseLanguageDetection ScanPhase = "Detecting languages"
	PhaseGrammarDownload   ScanPhase = "Downloading grammars"
	PhaseAnalyzing         ScanPhase = "Analyzing"
	PhaseScoring           ScanPhase = "Scoring & grading"
	PhaseOutputGeneration  ScanPhase = "Generating output"
	PhaseCosigning         ScanPhase = "Co-signing"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Progress displays scan progress to stderr with a spinner,
// elapsed time, and current phase.
type Progress struct {
	writer   io.Writer
	mu       sync.Mutex
	phase    ScanPhase
	detail   string
	start    time.Time
	stop     chan struct{}
	done     chan struct{}
	enabled  bool
	stopOnce sync.Once
	started  bool
}

// NewProgress creates a progress indicator that writes to w.
// If enabled is false, all operations are no-ops.
func NewProgress(w io.Writer, enabled bool) *Progress {
	return &Progress{
		writer:  w,
		start:   time.Now(),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
		enabled: enabled,
	}
}

// Start begins the spinner animation in a background goroutine.
func (p *Progress) Start() {
	if !p.enabled {
		close(p.done)
		return
	}
	p.started = true
	go p.run()
}

// SetPhase updates the displayed phase.
func (p *Progress) SetPhase(phase ScanPhase) {
	p.mu.Lock()
	p.phase = phase
	p.detail = ""
	p.mu.Unlock()
}

// SetPhaseDetail updates the phase with additional detail (e.g. repo name).
func (p *Progress) SetPhaseDetail(phase ScanPhase, detail string) {
	p.mu.Lock()
	p.phase = phase
	p.detail = detail
	p.mu.Unlock()
}

// Stop stops the spinner and clears the line.
// Safe to call multiple times or without calling Start().
func (p *Progress) Stop() {
	if !p.enabled || !p.started {
		return
	}
	p.stopOnce.Do(func() {
		close(p.stop)
		<-p.done
		// Clear the spinner line
		fmt.Fprintf(p.writer, "\r\033[K")
	})
}

// Elapsed returns the time since the progress was created.
func (p *Progress) Elapsed() time.Duration {
	return time.Since(p.start)
}

func (p *Progress) run() {
	defer close(p.done)
	frame := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.mu.Lock()
			phase := p.phase
			detail := p.detail
			p.mu.Unlock()

			elapsed := time.Since(p.start).Truncate(time.Second)
			spinner := spinnerFrames[frame%len(spinnerFrames)]

			line := fmt.Sprintf("%s %s", spinner, phase)
			if detail != "" {
				line += fmt.Sprintf(" (%s)", detail)
			}
			line += fmt.Sprintf("  [%s]", elapsed)

			fmt.Fprintf(p.writer, "\r\033[K%s", line)
			frame++
		}
	}
}
