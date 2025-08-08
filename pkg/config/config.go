package config

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetOllamaBaseURL returns the Ollama base URL
func GetOllamaBaseURL() string {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		return "http://localhost:11434"
	}
	return baseURL
}

// GetDefaultModel returns the default Ollama model
func GetDefaultModel() string {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		return "llama2"
	}
	return model
}

// GetDefaultAvailableModels returns the default list of available models
func GetDefaultAvailableModels() []string {
	return []string{"gemma3:4b", "deepseek-r1:8b"}
}

// OllamaModel represents a model from the Ollama API
type OllamaModel struct {
	Name       string            `json:"name"`
	Model      string            `json:"model"`
	ModifiedAt string            `json:"modified_at"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest"`
	Details    map[string]interface{} `json:"details"`
}

// OllamaTagsResponse represents the response from /api/tags endpoint
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// GetAvailableModels returns the list of available Ollama models
// First tries to fetch from Ollama API, falls back to environment variable or defaults
func GetAvailableModels() []string {
	// Try to get models from Ollama API first
	if apiModels := getModelsFromOllamaAPI(); len(apiModels) > 0 {
		return apiModels
	}

	// Fallback to environment variable
	modelsEnv := os.Getenv("OLLAMA_AVAILABLE_MODELS")
	if modelsEnv == "" {
		return GetDefaultAvailableModels()
	}
	
	models := strings.Split(modelsEnv, ",")
	var cleanModels []string
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model != "" {
			cleanModels = append(cleanModels, model)
		}
	}
	return cleanModels
}

// getModelsFromOllamaAPI fetches available models from Ollama API
func getModelsFromOllamaAPI() []string {
	baseURL := GetOllamaBaseURL()
	url := baseURL + "/api/tags"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var tagsResponse OllamaTagsResponse
	if err := json.Unmarshal(body, &tagsResponse); err != nil {
		return nil
	}

	var modelNames []string
	for _, model := range tagsResponse.Models {
		modelNames = append(modelNames, model.Name)
	}

	return modelNames
}

// GetSystemPrompt returns the system prompt for the LLM
func GetSystemPrompt() string {
	return os.Getenv("OLLAMA_SYSTEM_PROMPT")
}

// GetMaxTokens returns the maximum number of tokens
func GetMaxTokens() int {
	if val := os.Getenv("OLLAMA_NUM_PREDICT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return 100
}

// GetMaxContextMessages returns the maximum number of context messages
func GetMaxContextMessages() int {
	if val := os.Getenv("MAX_CONTEXT_MESSAGES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed * 2 // User + AI message pairs
		}
	}
	return 20 // Default to 10 conversation pairs
}