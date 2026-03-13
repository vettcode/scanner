package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProgress_Disabled(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, false)
	p.Start()
	p.SetPhase(PhaseAnalyzing)
	p.Stop()
	// Disabled progress should produce no output
	assert.Empty(t, buf.String())
}

func TestProgress_Enabled_ProducesOutput(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, true)
	p.Start()
	p.SetPhase(PhaseAnalyzing)
	time.Sleep(150 * time.Millisecond) // let spinner tick
	p.Stop()
	// Should have produced some output
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "Analyzing")
}

func TestProgress_SetPhaseDetail(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, true)
	p.Start()
	p.SetPhaseDetail(PhaseAnalyzing, "backend")
	time.Sleep(150 * time.Millisecond)
	p.Stop()
	assert.Contains(t, buf.String(), "backend")
}

func TestProgress_Elapsed(t *testing.T) {
	p := NewProgress(nil, false)
	time.Sleep(50 * time.Millisecond)
	assert.GreaterOrEqual(t, p.Elapsed().Milliseconds(), int64(40))
}

func TestProgress_Phases(t *testing.T) {
	// Verify all phases have descriptive strings
	phases := []ScanPhase{
		PhaseLanguageDetection,
		PhaseGrammarDownload,
		PhaseAnalyzing,
		PhaseScoring,
		PhaseOutputGeneration,
		PhaseCosigning,
	}
	for _, p := range phases {
		assert.NotEmpty(t, string(p))
	}
}
