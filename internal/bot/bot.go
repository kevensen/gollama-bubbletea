package bot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/ollama/ollama/api"
)

type Bot struct {
	ollamaClient *api.Client
	model        string
	messages     []api.Message
}

func NewBot(ctx context.Context, apiEndpoint string, model string) (*Bot, error) {
	u, err := url.Parse(apiEndpoint)
	if err != nil {
		return nil, err
	}
	oc := api.NewClient(u, http.DefaultClient)
	b := &Bot{ollamaClient: oc, model: model, messages: make([]api.Message, 0)}

	if err := b.UseModel(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to use model: %v", err)
	}

	return b, nil
}

func (b *Bot) SendMessage(ctx context.Context, role, message string, respFunc api.ChatResponseFunc) error {
	b.AddMessage(role, message)
	req := &api.ChatRequest{Model: b.model, Messages: b.messages}

	// Assuming ollamaClient has a method Send that takes the endpoint and data
	err := b.ollamaClient.Chat(ctx, req, respFunc)
	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) AddMessage(role, message string) {
	b.messages = append(b.messages, api.Message{Content: message, Role: role})
}

func (b *Bot) RenderMessages() []api.Message {
	return b.messages
}

func (b *Bot) MessageLen() int {
	return len(b.messages)
}

func (b *Bot) Models(ctx context.Context) ([]string, error) {
	resp, err := b.ollamaClient.List(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]string, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = m.Name
	}
	return models, nil
}

func (b *Bot) CurrentModel() string {
	return b.model
}

func (b *Bot) UseModel(ctx context.Context, model string) error {
	ms, err := b.Models(ctx)
	if err != nil {
		return fmt.Errorf("failed to get models: %v", err)
	}

	if !slices.Contains(ms, model) {
		return fmt.Errorf("model not found: %s", model)
	}

	b.model = model
	return nil
}

func (b *Bot) ModelExists(ctx context.Context, model string) (bool, error) {
	ms, err := b.Models(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get models: %v", err)
	}

	return slices.Contains(ms, model), nil
}
