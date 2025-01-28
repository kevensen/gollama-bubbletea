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

func (m *Manager) StyledMessages() []string {
	var messages []string
	var roleStyled string

	keys := maps.Keys(m.history.Messages)
	slices.Sort(keys)

	for _, key := range keys {
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
		messages = append(messages, msgStyled, "\n")
	}
	return messages
}

func (m *Manager) RenderMessages() string {
	return strings.Join(m.StyledMessages(), "\n")
}
