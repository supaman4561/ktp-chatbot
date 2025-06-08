package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

type LangChainBot struct {
	llm         llms.Model
	memories    map[string]*memory.ConversationBuffer
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
		llm:      llm,
		memories: make(map[string]*memory.ConversationBuffer),
	}, nil
}

func (lc *LangChainBot) getMemory(channelID string) *memory.ConversationBuffer {
	lc.memoryMutex.Lock()
	defer lc.memoryMutex.Unlock()

	if mem, exists := lc.memories[channelID]; exists {
		return mem
	}

	// Create new conversation buffer for this channel
	maxMessages := 10
	if val := os.Getenv("MAX_CONTEXT_MESSAGES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxMessages = parsed
		}
	}

	mem := memory.NewConversationBuffer(memory.WithMaxTokenLimit(maxMessages * 200))
	lc.memories[channelID] = mem
	log.Printf("Created new memory buffer for channel %s with max %d messages", channelID, maxMessages)
	return mem
}

func (lc *LangChainBot) GenerateResponse(ctx context.Context, channelID, userMessage string) (string, error) {
	mem := lc.getMemory(channelID)

	// Create system message if specified
	systemPrompt := os.Getenv("OLLAMA_SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = "100単語以内で簡潔に日本語で返信してください。"
	}

	// Build messages array with system prompt, history, and user message
	messages := []schema.ChatMessage{
		schema.SystemChatMessage{Content: systemPrompt},
	}

	// Add conversation history
	historyMessages, err := mem.ChatHistory.Messages(ctx)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
	} else {
		messages = append(messages, historyMessages...)
	}

	// Add current user message
	messages = append(messages, schema.HumanChatMessage{Content: userMessage})

	// Log the conversation for debugging
	log.Printf("Sending %d messages to LLM for channel %s", len(messages), channelID)
	for i, msg := range messages {
		log.Printf("Message %d [%s]: %s", i+1, msg.GetType(), msg.GetContent())
	}

	// Generate response
	response, err := lc.llm.GenerateContent(ctx, messages, llms.WithMaxTokens(getMaxTokens()))
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	responseText := response.Choices[0].Content

	// Add user message and AI response to memory
	err = mem.ChatHistory.AddUserMessage(ctx, userMessage)
	if err != nil {
		log.Printf("Error adding user message to memory: %v", err)
	}

	err = mem.ChatHistory.AddAIMessage(ctx, responseText)
	if err != nil {
		log.Printf("Error adding AI message to memory: %v", err)
	}

	return responseText, nil
}

func (lc *LangChainBot) ClearMemory(channelID string) {
	lc.memoryMutex.Lock()
	defer lc.memoryMutex.Unlock()

	if mem, exists := lc.memories[channelID]; exists {
		mem.Clear(context.Background())
		log.Printf("Memory cleared for channel %s", channelID)
	} else {
		log.Printf("No memory found for channel %s", channelID)
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

