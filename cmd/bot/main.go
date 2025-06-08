package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/supaman4561/ktp-chatbot/pkg/bot"
)

var (
	ollamaClient    *bot.OllamaClient
	allowedChannels map[string]bool
	contextManager  *bot.ContextManager
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable is required")
	}
	log.Printf("Bot token loaded (length: %d characters)", len(token))

	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "llama2"
	}

	ollamaClient = bot.NewOllamaClient(ollamaURL, ollamaModel)
	log.Printf("Ollama client initialized: %s with model %s", ollamaURL, ollamaModel)

	// Initialize context manager
	maxMessages := 10
	if val := os.Getenv("MAX_CONTEXT_MESSAGES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxMessages = parsed
		}
	}
	contextManager = bot.NewContextManager(maxMessages)
	log.Printf("Context manager initialized with max %d messages per channel", maxMessages)

	// Setup allowed channels
	allowedChannels = make(map[string]bool)
	allowedChannelIDs := os.Getenv("ALLOWED_CHANNEL_IDS")
	if allowedChannelIDs != "" {
		channelIDs := strings.Split(allowedChannelIDs, ",")
		for _, id := range channelIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				allowedChannels[id] = true
			}
		}
		log.Printf("Bot will only respond in %d specified channels", len(allowedChannels))
	} else {
		log.Println("No channel restrictions - bot will respond in all channels")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(ready)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	log.Println("Opening Discord connection...")
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection: ", err)
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Shutting down bot...")
	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as: %s#%s (ID: %s)", event.User.Username, event.User.Discriminator, event.User.ID)
	log.Printf("Bot is in %d guilds", len(event.Guilds))
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Printf("Message received: '%s' from %s (ID: %s) in channel %s", m.Content, m.Author.Username, m.Author.ID, m.ChannelID)

	if m.Author.ID == s.State.User.ID {
		log.Println("Ignoring message from bot itself")
		return
	}

	// Check if channel is allowed (if restrictions are set)
	if len(allowedChannels) > 0 && !allowedChannels[m.ChannelID] {
		log.Printf("Channel %s not in allowed channels list, ignoring message", m.ChannelID)
		return
	}

	if m.Content == "!ping" {
		log.Println("Responding to !ping command")
		_, err := s.ChannelMessageSend(m.ChannelID, "Pong!")
		if err != nil {
			log.Printf("Error sending ping response: %v", err)
		}
	}

	if m.Content == "!hello" {
		log.Printf("Responding to !hello command for user: %s", m.Author.Username)
		_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello, %s!", m.Author.Username))
		if err != nil {
			log.Printf("Error sending hello response: %v", err)
		}
		return
	}

	if m.Content == "!clear" {
		log.Printf("!clear command received from user %s in channel %s", m.Author.Username, m.ChannelID)

		// Get conversation history before clearing to show what was cleared
		historyBefore := contextManager.GetConversationHistory(m.ChannelID)
		if historyBefore != "" {
			log.Printf("Conversation history before clear:\n%s", historyBefore)
		} else {
			log.Printf("No conversation history found for channel %s", m.ChannelID)
		}

		contextManager.ClearContext(m.ChannelID)
		log.Printf("Context cleared for channel %s", m.ChannelID)

		// Verify that context was actually cleared
		historyAfter := contextManager.GetConversationHistory(m.ChannelID)
		if historyAfter == "" {
			log.Printf("Context successfully cleared for channel %s", m.ChannelID)
		} else {
			log.Printf("Warning: Context may not have been cleared properly for channel %s", m.ChannelID)
		}

		_, err := s.ChannelMessageSend(m.ChannelID, "会話履歴をクリアしました。")
		if err != nil {
			log.Printf("Error sending clear response: %v", err)
		}
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		return
	}

	if len(m.Content) > 0 && ollamaClient != nil {
		log.Printf("Processing chat message: %s", m.Content)

		// Get conversation history
		conversationHistory := contextManager.GetConversationHistory(m.ChannelID)

		// Add user message to context
		contextManager.AddMessage(m.ChannelID, m.Author.Username, m.Content, false)

		// Generate response with context
		response, err := ollamaClient.GenerateResponseWithContext(m.Content, conversationHistory)
		if err != nil {
			log.Printf("Error getting Ollama response: %v", err)
			_, sendErr := s.ChannelMessageSend(m.ChannelID, "申し訳ありませんが、応答の生成中にエラーが発生しました。")
			if sendErr != nil {
				log.Printf("Error sending error message: %v", sendErr)
			}
			return
		}

		if len(response) > 2000 {
			response = response[:1997] + "..."
		}

		// Add bot response to context
		contextManager.AddMessage(m.ChannelID, "ktp-chan", response, true)

		_, err = s.ChannelMessageSend(m.ChannelID, response)
		if err != nil {
			log.Printf("Error sending chat response: %v", err)
		}
	}
}
