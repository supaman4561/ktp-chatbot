package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/supaman4561/ktp-chatbot/pkg/bot"
)

var (
	langChainBot    *bot.LangChainBot
	allowedChannels map[string]bool
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

	// Initialize LangChain bot
	langChainBot, err = bot.NewLangChainBot()
	if err != nil {
		log.Fatal("Failed to initialize LangChain bot: ", err)
	}

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
		langChainBot.ClearMemory(m.ChannelID)
		_, err := s.ChannelMessageSend(m.ChannelID, "会話履歴をクリアしました。")
		if err != nil {
			log.Printf("Error sending clear response: %v", err)
		}
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		return
	}

	if len(m.Content) > 0 && langChainBot != nil {
		log.Printf("Processing chat message: %s", m.Content)

		// Generate response using LangChain
		ctx := context.Background()
		response, err := langChainBot.GenerateResponse(ctx, m.ChannelID, m.Content)
		if err != nil {
			log.Printf("Error getting LangChain response: %v", err)
			_, sendErr := s.ChannelMessageSend(m.ChannelID, "申し訳ありませんが、応答の生成中にエラーが発生しました。")
			if sendErr != nil {
				log.Printf("Error sending error message: %v", sendErr)
			}
			return
		}

		if len(response) > 2000 {
			response = response[:1997] + "..."
		}

		_, err = s.ChannelMessageSend(m.ChannelID, response)
		if err != nil {
			log.Printf("Error sending chat response: %v", err)
		}
	}
}
