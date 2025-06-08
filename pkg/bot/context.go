package bot

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Message struct {
	User      string
	Content   string
	Timestamp time.Time
}

type ConversationContext struct {
	Messages []Message
	mutex    sync.RWMutex
}

type ContextManager struct {
	contexts    map[string]*ConversationContext
	mutex       sync.RWMutex
	maxMessages int
}

func NewContextManager(maxMessages int) *ContextManager {
	return &ContextManager{
		contexts:    make(map[string]*ConversationContext),
		maxMessages: maxMessages,
	}
}

func (cm *ContextManager) AddMessage(channelID, username, content string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.contexts[channelID]; !exists {
		cm.contexts[channelID] = &ConversationContext{
			Messages: make([]Message, 0),
		}
	}

	context := cm.contexts[channelID]
	context.mutex.Lock()
	defer context.mutex.Unlock()

	message := Message{
		User:      username,
		Content:   content,
		Timestamp: time.Now(),
	}

	context.Messages = append(context.Messages, message)

	// Keep only the last N messages
	if len(context.Messages) > cm.maxMessages {
		context.Messages = context.Messages[len(context.Messages)-cm.maxMessages:]
	}
}

func (cm *ContextManager) GetConversationHistory(channelID string) string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	context, exists := cm.contexts[channelID]
	if !exists {
		return ""
	}

	context.mutex.RLock()
	defer context.mutex.RUnlock()

	if len(context.Messages) == 0 {
		return ""
	}

	var history strings.Builder
	history.WriteString("会話履歴:\n")
	
	for _, msg := range context.Messages {
		history.WriteString(fmt.Sprintf("%s: %s\n", msg.User, msg.Content))
	}

	return history.String()
}

func (cm *ContextManager) ClearContext(channelID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if context, exists := cm.contexts[channelID]; exists {
		context.mutex.Lock()
		context.Messages = make([]Message, 0)
		context.mutex.Unlock()
	}
}