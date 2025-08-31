package messages

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/parakeet-nest/parakeet/history"
	"github.com/parakeet-nest/parakeet/llm"
	"golang.org/x/exp/maps"
)

type Manager struct {
	history          history.MemoryMessages
	currentMessageID int
}

type Message struct {
	Role    string
	Content string
}

func (msg *Message) ToLLMMessage() llm.Message {
	return llm.Message{Role: msg.Role, Content: msg.Content}
}

func NewManager() *Manager {
	return &Manager{
		currentMessageID: 0,
		history: history.MemoryMessages{
			Messages: make(map[string]llm.MessageRecord),
		},
	}
}

func (m *Manager) AddMessage(msg llm.Message) *llm.Message {
	m.currentMessageID++
	m.history.SaveMessage(strconv.Itoa(m.currentMessageID), msg)
	return &msg
}

func (m *Manager) MessagesForSending() ([]llm.Message, error) {
	llms, err := m.history.GetAllMessages()
	if err != nil {
		return nil, err
	}

	var llmMsgs []llm.Message
	for _, msg := range llms {
		if msg.Role != "error" {
			llmMsgs = append(llmMsgs, msg)
		}
	}
	return llmMsgs, nil
}

func (m *Manager) Len() int {
	return len(m.history.Messages)
}

// EstimateTokens provides a rough estimate of tokens in the current context
// Uses ~4 characters per token as a general estimate for English text
func (m *Manager) EstimateTokens() int {
	totalChars := 0

	// Get all messages in the history
	llms, err := m.history.GetAllMessages()
	if err != nil {
		return 0
	}

	// Count characters in all message content
	for _, msg := range llms {
		totalChars += len(msg.Content)
		totalChars += len(msg.Role) + 10 // Add some overhead for role and formatting
	}

	// Rough estimate: ~4 characters per token
	return totalChars / 4
}

func (m *Manager) Clear() {
	m.currentMessageID = 0
	m.history.Messages = make(map[string]llm.MessageRecord)
}

func (m *Manager) StyledMessages() []string {
	var messages []string
	var roleStyled string

	keys := maps.Keys(m.history.Messages)

	// Convert string keys to integers for proper numerical sorting
	intKeys := make([]int, len(keys))
	for i, key := range keys {
		intKey, err := strconv.Atoi(key)
		if err != nil {
			panic(err)
		}
		intKeys[i] = intKey
	}
	slices.Sort(intKeys)

	// Convert back to strings and process messages in correct order
	for i, intKey := range intKeys {
		key := strconv.Itoa(intKey)
		msg, err := m.history.Get(key)
		if err != nil {
			// Don't panic, heh
			panic(err)
		}

		role := strings.ToUpper(msg.Role[:1]) + strings.ToLower(msg.Role[1:])
		var c lipgloss.TerminalColor
		switch {
		case msg.Role == "assistant":
			c = lipgloss.Color("3") // Yellow for assistant
		case msg.Role == "system":
			c = lipgloss.Color("4") // Blue for tool
		case msg.Role == "error":
			c = lipgloss.Color("1") // Red for error
		case msg.Role == "user":
			c = lipgloss.Color("2") // Green for user
		}

		roleStyled = lipgloss.NewStyle().Foreground(c).Render(role)
		msgStyled := roleStyled + ": " + msg.Content
		messages = append(messages, msgStyled)

		// Add extra spacing after assistant messages when followed by a user message
		if msg.Role == "assistant" && i < len(intKeys)-1 {
			// Check if the next message is from user
			nextKey := strconv.Itoa(intKeys[i+1])
			nextMsg, err := m.history.Get(nextKey)
			if err == nil && nextMsg.Role == "user" {
				messages = append(messages, "") // Add blank line
			}
		}
	}
	return messages
}

func (m *Manager) RenderMessages() string {
	return strings.Join(m.StyledMessages(), "\n")
}
