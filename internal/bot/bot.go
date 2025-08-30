package bot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kevensen/gollama-bubbletea/internal/bot/messages"
	"github.com/kevensen/gollama-bubbletea/internal/bot/models"

	"github.com/parakeet-nest/parakeet/completion"
	"github.com/parakeet-nest/parakeet/enums/option"
	"github.com/parakeet-nest/parakeet/llm"
)

type Bot struct {
	ollamaUrl      string
	MessageManager *messages.Manager
	ModelManager   *models.Manager
}

func NewBot(ctx context.Context, apiEndpoint string, initialModel string) (*Bot, error) {
	b := &Bot{
		ollamaUrl: apiEndpoint,
	}

	b.MessageManager = messages.NewManager()

	// Only try to create model manager if we can connect
	if apiEndpoint != "" && TestConnection(apiEndpoint) == nil {
		modelManager, err := models.NewManager(apiEndpoint, initialModel)
		if err != nil {
			return nil, fmt.Errorf("failed to create model manager: %v", err)
		}
		b.ModelManager = modelManager
	}
	// If no connection, ModelManager will be nil and we'll handle that in TUI

	return b, nil
}

func (b *Bot) SendMessage(ctx context.Context, role, message string) (*llm.Answer, error) {
	var msgsForSending []llm.Message
	var err error
	msg := llm.Message{Role: role, Content: message}

	msgsForSending, err = b.MessageManager.MessagesForSending()
	msgsForSending = append(msgsForSending, msg)

	query := llm.Query{
		Model:    b.ModelManager.CurrentModel(),
		Messages: msgsForSending,
		Options: llm.SetOptions(map[string]any{
			option.Temperature:   0.5,
			option.RepeatLastN:   2,
			option.RepeatPenalty: 2.0,
			option.Verbose:       false,
		}),
	}

	b.MessageManager.AddMessage(msg)

	var ans llm.Answer
	ans, err = completion.Chat(b.ollamaUrl, query)
	if err != nil {
		return nil, err
	}

	return &ans, nil
}

func (b *Bot) MessageLen() int {
	return b.MessageManager.Len()
}

func (b *Bot) EstimateTokens() int {
	if b.MessageManager == nil {
		return 0
	}
	return b.MessageManager.EstimateTokens()
}

// GetContextWindowSize returns the context window size of the current model
func (b *Bot) GetContextWindowSize() (int, error) {
	if b.ModelManager == nil {
		return 0, fmt.Errorf("no model manager available")
	}
	return b.ModelManager.GetContextWindowSize()
}

func (b *Bot) ClearMessages() {
	b.MessageManager.Clear()
}

// TestConnection tests if the Ollama server is reachable
func TestConnection(url string) error {
	if url == "" {
		return fmt.Errorf("no Ollama URL configured")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Test the /api/tags endpoint which should be available on Ollama
	resp, err := client.Get(url + "/api/tags")
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama server at %s returned status %d", url, resp.StatusCode)
	}

	return nil
}

// InitializeModelManager creates the model manager with the given URL and model
func (b *Bot) InitializeModelManager(apiEndpoint string, initialModel string) error {
	if apiEndpoint == "" {
		return fmt.Errorf("API endpoint cannot be empty")
	}

	// Test connection first
	if err := TestConnection(apiEndpoint); err != nil {
		return err
	}

	b.ollamaUrl = apiEndpoint

	modelManager, err := models.NewManager(apiEndpoint, initialModel)
	if err != nil {
		return fmt.Errorf("failed to create model manager: %v", err)
	}

	b.ModelManager = modelManager
	return nil
}

// HasValidConnection returns true if the bot has a working model manager
func (b *Bot) HasValidConnection() bool {
	return b.ModelManager != nil
}
