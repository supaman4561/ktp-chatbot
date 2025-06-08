package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type OllamaClient struct {
	BaseURL      string
	Model        string
	client       *http.Client
	NumPredict   int
	Temperature  float64
	TopP         float64
	SystemPrompt string
}

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	numPredict := 100
	if val := os.Getenv("OLLAMA_NUM_PREDICT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			numPredict = parsed
		}
	}

	temperature := 0.7
	if val := os.Getenv("OLLAMA_TEMPERATURE"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			temperature = parsed
		}
	}

	topP := 0.9
	if val := os.Getenv("OLLAMA_TOP_P"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			topP = parsed
		}
	}

	systemPrompt := "以下のチャットに対して、100単語以内で簡潔に日本語で返信してください。長すぎる返信は避けてください。"
	if val := os.Getenv("OLLAMA_SYSTEM_PROMPT"); val != "" {
		systemPrompt = val
	}

	return &OllamaClient{
		BaseURL:      baseURL,
		Model:        model,
		NumPredict:   numPredict,
		Temperature:  temperature,
		TopP:         topP,
		SystemPrompt: systemPrompt,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *OllamaClient) GenerateResponse(prompt string) (string, error) {
	// Use system prompt from environment variable
	fullPrompt := fmt.Sprintf("%s\n\nチャット内容: %s", c.SystemPrompt, prompt)

	requestBody := OllamaRequest{
		Model:  c.Model,
		Prompt: fullPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"num_predict": c.NumPredict,
			"temperature": c.Temperature,
			"top_p":       c.TopP,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Response, nil
}

