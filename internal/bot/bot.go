package bot

import (
	"context"
	"fmt"

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

	modelManager, err := models.NewManager(apiEndpoint, initialModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create model manager: %v", err)
	}

	b.ModelManager = modelManager

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

func (b *Bot) ClearMessages() {
	b.MessageManager.Clear()
}
