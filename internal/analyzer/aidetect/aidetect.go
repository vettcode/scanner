package aidetect

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/vettcode/scanner/internal/walker"
)

// Result holds the AI detection results.
type Result struct {
	HasLLMAPI          bool
	LLMProviders       []string
	HasVectorDB        bool
	VectorDBProviders  []string
	HasRAGPipeline     bool
	HasMCP             bool
	HasFineTuning      bool
	HasTrainingPipeline bool
	HasProprietaryData bool
}

// llmDependencies maps dependency names to LLM providers.
var llmDependencies = map[string]string{
	"openai":                    "OpenAI",
	"@openai/api":               "OpenAI",
	"anthropic":                 "Anthropic",
	"@anthropic-ai/sdk":         "Anthropic",
	"cohere":                    "Cohere",
	"cohere-ai":                 "Cohere",
	"google-generativeai":       "Google AI",
	"@google/generative-ai":     "Google AI",
	"google-cloud-aiplatform":   "Google Vertex AI",
	"langchain":                 "LangChain",
	"langchain-core":            "LangChain",
	"@langchain/core":           "LangChain",
	"llama-index":               "LlamaIndex",
	"llama_index":               "LlamaIndex",
	"replicate":                 "Replicate",
	"together":                  "Together AI",
	"groq":                      "Groq",
	"mistralai":                 "Mistral AI",
	"ollama":                    "Ollama",
	"huggingface_hub":           "Hugging Face",
	"@huggingface/inference":    "Hugging Face",
}

// vectorDBDependencies maps dependency names to vector DB providers.
var vectorDBDependencies = map[string]string{
	"pinecone-client":       "Pinecone",
	"@pinecone-database/pinecone": "Pinecone",
	"weaviate-client":       "Weaviate",
	"chromadb":              "ChromaDB",
	"qdrant-client":         "Qdrant",
	"qdrant_client":         "Qdrant",
	"milvus":                "Milvus",
	"pymilvus":              "Milvus",
	"pgvector":              "pgvector",
	"faiss":                 "FAISS",
	"faiss-cpu":             "FAISS",
	"faiss-gpu":             "FAISS",
}

// mcpDependencies maps dependency names to MCP detection.
var mcpDependencies = map[string]bool{
	"@modelcontextprotocol/sdk":    true,
	"mcp":                          true,
	"mcp-server":                   true,
	"modelcontextprotocol":         true,
}

// trainingDependencies maps dependency names to training frameworks.
var trainingDependencies = map[string]bool{
	"torch":             true,
	"pytorch":           true,
	"tensorflow":        true,
	"tf":                true,
	"transformers":      true,
	"datasets":          true,
	"accelerate":        true,
	"peft":              true,
	"trl":               true,
	"keras":             true,
	"jax":               true,
	"flax":              true,
}

// fineTuningPatterns are file path patterns indicating fine-tuning code.
var fineTuningPatterns = []string{
	"fine_tune", "finetune", "fine-tune",
	"train", "training",
}

// llmDirPatterns are directory names that indicate a project implements
// LLM functionality (as opposed to merely consuming an LLM API).
var llmDirPatterns = map[string]string{
	"llm":       "LLM (native)",
	"llama":     "LLM (native)",
	"inference": "LLM (native)",
	"tokenizer": "LLM (native)",
}

// vectorDBDirPatterns are directory names that indicate vector DB implementation.
var vectorDBDirPatterns = map[string]string{
	"embedding":  "Embeddings (native)",
	"embeddings": "Embeddings (native)",
}

// mcpDirPatterns are directory names that indicate MCP implementation.
var mcpDirPatterns = []string{
	"mcp",
}

// dataPipelineDeps maps dependency names to data pipeline frameworks.
var dataPipelineDeps = map[string]bool{
	"airflow":     true,
	"prefect":     true,
	"dagster":     true,
	"luigi":       true,
	"dbt":         true,
	"dbt-core":    true,
}

// Detect detects AI/ML usage from dependencies and file patterns.
func Detect(deps []string, files []walker.FileInfo) *Result {
	r := &Result{}

	llmSet := make(map[string]bool)
	vecSet := make(map[string]bool)

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		// LLM API detection
		if provider, ok := llmDependencies[dep]; ok {
			llmSet[provider] = true
		} else if provider, ok := llmDependencies[depLower]; ok {
			llmSet[provider] = true
		}

		// Vector DB detection
		if provider, ok := vectorDBDependencies[dep]; ok {
			vecSet[provider] = true
		} else if provider, ok := vectorDBDependencies[depLower]; ok {
			vecSet[provider] = true
		}

		// MCP detection
		if mcpDependencies[dep] || mcpDependencies[depLower] {
			r.HasMCP = true
		}

		// Training pipeline detection
		if trainingDependencies[dep] || trainingDependencies[depLower] {
			r.HasTrainingPipeline = true
		}

		// Data pipeline detection
		if dataPipelineDeps[dep] || dataPipelineDeps[depLower] {
			r.HasProprietaryData = true
		}
	}

	// File and directory pattern matching
	dirsSeen := make(map[string]bool)
	for _, f := range files {
		baseLower := strings.ToLower(filepath.Base(f.Path))

		// Fine-tuning file patterns
		if !r.HasFineTuning {
			for _, pat := range fineTuningPatterns {
				if strings.Contains(baseLower, pat) {
					r.HasFineTuning = true
					break
				}
			}
		}

		// Collect unique directory names from the file path
		dir := filepath.Dir(f.Path)
		for dir != "." && dir != "/" && dir != "" {
			name := strings.ToLower(filepath.Base(dir))
			if dirsSeen[name] {
				break
			}
			dirsSeen[name] = true
			dir = filepath.Dir(dir)
		}
	}

	// Directory-based LLM detection (native implementations)
	for dirName := range dirsSeen {
		if provider, ok := llmDirPatterns[dirName]; ok {
			llmSet[provider] = true
		}
	}

	// Directory-based vector DB / embeddings detection
	for dirName := range dirsSeen {
		if provider, ok := vectorDBDirPatterns[dirName]; ok {
			vecSet[provider] = true
		}
	}

	// Directory-based MCP detection
	for _, pat := range mcpDirPatterns {
		if dirsSeen[pat] {
			r.HasMCP = true
			break
		}
	}

	r.LLMProviders = setToSortedSlice(llmSet)
	r.HasLLMAPI = len(r.LLMProviders) > 0

	r.VectorDBProviders = setToSortedSlice(vecSet)
	r.HasVectorDB = len(r.VectorDBProviders) > 0

	// RAG pipeline = LLM API + Vector DB
	r.HasRAGPipeline = r.HasLLMAPI && r.HasVectorDB

	return r
}

func setToSortedSlice(s map[string]bool) []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
