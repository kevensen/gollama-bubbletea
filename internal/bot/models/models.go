package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/parakeet-nest/parakeet/llm"
)

type Manager struct {
	list         llm.ModelList
	currentModel string
	ollamaURL    string // Store URL for detailed model queries
}

// ModelDetails represents detailed model information from Ollama
type ModelDetails struct {
	License    string    `json:"license"`
	ModelFile  string    `json:"modelfile"`
	Parameters string    `json:"parameters"`
	Template   string    `json:"template"`
	Details    Details   `json:"details"`
	ModelInfo  ModelInfo `json:"model_info"`
}

type Details struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

type ModelInfo struct {
	GeneralArchitecture               string  `json:"general.architecture"`
	GeneralBasename                   string  `json:"general.basename"`
	GeneralFileType                   int     `json:"general.file_type"`
	GeneralFinetune                   string  `json:"general.finetune"`
	GeneralParameterCount             int64   `json:"general.parameter_count"`
	GeneralQuantizationVersion        int     `json:"general.quantization_version"`
	GeneralSizeLabel                  string  `json:"general.size_label"`
	GeneralType                       string  `json:"general.type"`
	LlamaAttentionHeadCount           int     `json:"llama.attention.head_count"`
	LlamaAttentionHeadCountKV         int     `json:"llama.attention.head_count_kv"`
	LlamaAttentionLayerNormRMSEpsilon float64 `json:"llama.attention.layer_norm_rms_epsilon"`
	LlamaBlockCount                   int     `json:"llama.block_count"`
	LlamaContextLength                int     `json:"llama.context_length"` // This is what we want!
	LlamaEmbeddingLength              int     `json:"llama.embedding_length"`
	LlamaFeedForwardLength            int     `json:"llama.feed_forward_length"`
	LlamaRopeDimensionCount           int     `json:"llama.rope.dimension_count"`
	LlamaRopeFreqBase                 float64 `json:"llama.rope.freq_base"`
	LlamaVocabSize                    int     `json:"llama.vocab_size"`
}

func NewManager(ollamaURL string, initialModel string) (*Manager, error) {
	list, _, err := llm.GetModelsList(ollamaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %v", err)
	}

	mgr := &Manager{
		list:      list,
		ollamaURL: ollamaURL,
	}

	// Check if the initial model exists
	if !slices.Contains(mgr.ModelNames(), initialModel) {
		return nil, fmt.Errorf("model not found: %s", initialModel)
	}

	mgr.currentModel = initialModel
	return mgr, nil
}

func (m *Manager) ModelNames() []string {
	var ms []string
	for _, model := range m.list.Models {
		ms = append(ms, model.Name)
	}

	return ms
}

func (m *Manager) MaxModelNameLength() int {
	maxLen := 0
	for _, model := range m.list.Models {
		if len(model.Name) > maxLen {
			maxLen = len(model.Name)
		}
	}

	return maxLen
}

func (m *Manager) CurrentModel() string {
	return m.currentModel
}

func (m *Manager) UseModel(model string) error {
	if !slices.Contains(m.ModelNames(), model) {
		return fmt.Errorf("model not found: %s", model)
	}

	m.currentModel = model
	return nil
}

func (m *Manager) ModelExists(model string) bool {
	return slices.Contains(m.ModelNames(), model)
}

// GetContextWindowSize retrieves the context window size for the current model
func (m *Manager) GetContextWindowSize() (int, error) {
	return m.GetContextWindowSizeForModel(m.currentModel)
}

// GetContextWindowSizeForModel retrieves the context window size for a specific model
func (m *Manager) GetContextWindowSizeForModel(modelName string) (int, error) {
	if !m.ModelExists(modelName) {
		return 0, fmt.Errorf("model not found: %s", modelName)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request body with model name
	body := fmt.Sprintf(`{"name": "%s"}`, modelName)
	bodyReader := strings.NewReader(body)

	// Make request to Ollama's /api/show endpoint
	url := fmt.Sprintf("%s/api/show", m.ollamaURL)
	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get model details: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get model details: HTTP %d", resp.StatusCode)
	}

	var details ModelDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return 0, fmt.Errorf("failed to parse model details: %v", err)
	}

	// Try to get context length from different possible fields
	if details.ModelInfo.LlamaContextLength > 0 {
		return details.ModelInfo.LlamaContextLength, nil
	}

	// If we can't find the context length, return a reasonable default
	// Most modern models have at least 4k context
	return 4096, nil
}
