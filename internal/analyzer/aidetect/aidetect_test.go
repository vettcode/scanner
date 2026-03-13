package aidetect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/internal/walker"
)

func TestDetect_LLMProviders(t *testing.T) {
	deps := []string{"openai", "anthropic", "express"}
	r := Detect(deps, nil)
	assert.True(t, r.HasLLMAPI)
	assert.Contains(t, r.LLMProviders, "OpenAI")
	assert.Contains(t, r.LLMProviders, "Anthropic")
}

func TestDetect_VectorDB(t *testing.T) {
	deps := []string{"chromadb", "pinecone-client"}
	r := Detect(deps, nil)
	assert.True(t, r.HasVectorDB)
	assert.Contains(t, r.VectorDBProviders, "ChromaDB")
	assert.Contains(t, r.VectorDBProviders, "Pinecone")
}

func TestDetect_RAGPipeline(t *testing.T) {
	deps := []string{"openai", "chromadb"}
	r := Detect(deps, nil)
	assert.True(t, r.HasRAGPipeline)
}

func TestDetect_NoRAGWithoutBoth(t *testing.T) {
	// LLM only - no RAG
	r := Detect([]string{"openai"}, nil)
	assert.False(t, r.HasRAGPipeline)

	// Vector DB only - no RAG
	r = Detect([]string{"chromadb"}, nil)
	assert.False(t, r.HasRAGPipeline)
}

func TestDetect_MCP(t *testing.T) {
	deps := []string{"@modelcontextprotocol/sdk"}
	r := Detect(deps, nil)
	assert.True(t, r.HasMCP)
}

func TestDetect_TrainingPipeline(t *testing.T) {
	deps := []string{"torch", "transformers"}
	r := Detect(deps, nil)
	assert.True(t, r.HasTrainingPipeline)
}

func TestDetect_FineTuning(t *testing.T) {
	files := []walker.FileInfo{
		{Path: "/project/scripts/fine_tune_model.py"},
	}
	r := Detect(nil, files)
	assert.True(t, r.HasFineTuning)
}

func TestDetect_ProprietaryData(t *testing.T) {
	deps := []string{"airflow", "dagster"}
	r := Detect(deps, nil)
	assert.True(t, r.HasProprietaryData)
}

func TestDetect_NothingDetected(t *testing.T) {
	deps := []string{"express", "react", "pg"}
	r := Detect(deps, nil)
	assert.False(t, r.HasLLMAPI)
	assert.False(t, r.HasVectorDB)
	assert.False(t, r.HasRAGPipeline)
	assert.False(t, r.HasMCP)
	assert.False(t, r.HasFineTuning)
	assert.False(t, r.HasTrainingPipeline)
	assert.False(t, r.HasProprietaryData)
}
