package bot

import (
	"slices"

	"github.com/supaman4561/ktp-chatbot/pkg/config"
)

// GetAvailableModels returns the list of available Ollama models
func GetAvailableModels() []string {
	return config.GetAvailableModels()
}

// IsModelAvailable checks if a model is in the available models list
func IsModelAvailable(modelName string) bool {
	availableModels := GetAvailableModels()
	return slices.Contains(availableModels, modelName)
}
