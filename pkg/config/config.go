package config

import (
	"os"
	"strconv"
	"strings"
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

// GetAvailableModels returns the list of available Ollama models
func GetAvailableModels() []string {
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