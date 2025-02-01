package bot

import (
	"context"
	"fmt"

	"github.com/kevensen/gollama-bubbletea/internal/bot/messages"
	"github.com/kevensen/gollama-bubbletea/internal/bot/models"
	"github.com/kevensen/gollama-bubbletea/internal/tools"

	"github.com/parakeet-nest/parakeet/completion"
	"github.com/parakeet-nest/parakeet/enums/option"
	"github.com/parakeet-nest/parakeet/llm"
)

type Bot struct {
	ollamaUrl      string
	ToolManager    *tools.ToolManager
	MessageManager *messages.Manager
	ModelManager   *models.Manager
}

func NewBot(ctx context.Context, apiEndpoint string, initialModel string, ts map[string]tools.CallableTool) (*Bot, error) {
	b := &Bot{
		ollamaUrl: apiEndpoint,
	}

	b.ToolManager = tools.NewToolManager(ts)
	b.MessageManager = messages.NewManager()

	modelManager, err := models.NewManager(apiEndpoint, initialModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create model manager: %v", err)
	}

	b.ModelManager = modelManager

	return b, nil
}

func (b *Bot) SendMessage(ctx context.Context, role, message string, toolsEnabled bool) (*llm.Answer, error) {
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

	if toolsEnabled {
		query.Options.Temperature = 0.0
		query.Format = "json"
		query.Tools = b.ToolManager.LLMTools()
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
