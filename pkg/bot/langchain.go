package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

type LangChainBot struct {
	llm         llms.Model
	histories   map[string]schema.ChatMessageHistory
	memoryMutex sync.RWMutex
}

func NewLangChainBot() (*LangChainBot, error) {
	// Get Ollama configuration
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama2"
	}

	// Initialize Ollama LLM with LangChain
	llm, err := ollama.New(
		ollama.WithServerURL(baseURL),
		ollama.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Ollama LLM: %w", err)
	}

	log.Printf("LangChain Ollama client initialized: %s with model %s", baseURL, model)

	return &LangChainBot{
		llm:       llm,
		histories: make(map[string]schema.ChatMessageHistory),
	}, nil
}

func (lc *LangChainBot) getHistory(channelID string) schema.ChatMessageHistory {
	lc.memoryMutex.Lock()
	defer lc.memoryMutex.Unlock()

	if history, exists := lc.histories[channelID]; exists {
		return history
	}

	// Create new chat message history for this channel
	history := memory.NewChatMessageHistory()
	lc.histories[channelID] = history
	log.Printf("Created new message history for channel %s", channelID)
	return history
}

func (lc *LangChainBot) GenerateResponse(ctx context.Context, channelID, userMessage string) (string, error) {
	history := lc.getHistory(channelID)

	// Build conversation messages
	systemPrompt := os.Getenv("OLLAMA_SYSTEM_PROMPT")

	// Create messages array with system prompt and history
	messages := []llms.ChatMessage{
		llms.SystemChatMessage{Content: systemPrompt},
	}

	// Add history messages with context limit
	maxMessages := getMaxContextMessages()
	historyMessages, err := history.Messages(ctx)
	if err != nil {
		log.Printf("Error getting history messages: %v", err)
		historyMessages = []llms.ChatMessage{}
	}
	if len(historyMessages) > maxMessages {
		historyMessages = historyMessages[len(historyMessages)-maxMessages:]
	}
	messages = append(messages, historyMessages...)

	// Add current user message
	userMsg := llms.HumanChatMessage{Content: userMessage}
	messages = append(messages, userMsg)

	log.Printf("Generating response for channel %s with %d total messages", channelID, len(messages))

	// Convert to MessageContent format
	var messageContents []llms.MessageContent
	for _, msg := range messages {
		var role llms.ChatMessageType
		switch msg.GetType() {
		case llms.ChatMessageTypeSystem:
			role = llms.ChatMessageTypeSystem
		case llms.ChatMessageTypeHuman:
			role = llms.ChatMessageTypeHuman
		case llms.ChatMessageTypeAI:
			role = llms.ChatMessageTypeAI
		default:
			role = llms.ChatMessageTypeHuman
		}

		content := llms.MessageContent{
			Role: role,
			Parts: []llms.ContentPart{
				llms.TextPart(msg.GetContent()),
			},
		}
		messageContents = append(messageContents, content)
	}

	// Generate response using LangChain with structured messages
	maxTokens := getMaxTokens()
	result, err := lc.llm.GenerateContent(ctx, messageContents, llms.WithMaxTokens(maxTokens))
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	response := result.Choices[0].Content

	// Remove DeepSeek thinking tags if using deepseek-r1 model
	if strings.Contains(strings.ToLower(os.Getenv("OLLAMA_MODEL")), "deepseek-r1") {
		response = cleanDeepSeekResponse(response)
	}

	// Save conversation to history
	err = history.AddUserMessage(ctx, userMessage)
	if err != nil {
		log.Printf("Error adding user message to history: %v", err)
	}
	err = history.AddAIMessage(ctx, response)
	if err != nil {
		log.Printf("Error adding AI message to history: %v", err)
	}

	return response, nil
}

func (lc *LangChainBot) ClearMemory(channelID string) {
	lc.memoryMutex.Lock()
	defer lc.memoryMutex.Unlock()

	if history, exists := lc.histories[channelID]; exists {
		err := history.Clear(context.Background())
		if err != nil {
			log.Printf("Error clearing history for channel %s: %v", channelID, err)
		} else {
			log.Printf("Message history cleared for channel %s", channelID)
		}
	} else {
		log.Printf("No message history found for channel %s", channelID)
	}
}

func getMaxTokens() int {
	if val := os.Getenv("OLLAMA_NUM_PREDICT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return 100
}

func getMaxContextMessages() int {
	if val := os.Getenv("MAX_CONTEXT_MESSAGES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed * 2 // User + AI message pairs
		}
	}
	return 20 // Default to 10 conversation pairs
}

func cleanDeepSeekResponse(response string) string {
	// Remove <think>...</think> tags and their content
	thinkRegex := regexp.MustCompile(`(?s)<think>.*?</think>`)
	cleaned := thinkRegex.ReplaceAllString(response, "")
	
	// Remove extra whitespace and newlines
	cleaned = strings.TrimSpace(cleaned)
	
	// Remove multiple consecutive newlines
	multiNewlineRegex := regexp.MustCompile(`\n\s*\n\s*\n`)
	cleaned = multiNewlineRegex.ReplaceAllString(cleaned, "\n\n")
	
	return cleaned
}
