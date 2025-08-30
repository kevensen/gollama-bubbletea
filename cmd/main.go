package main

import (
	"context"
	"log"

	"github.com/kevensen/gollama-bubbletea/internal/bot"
	"github.com/kevensen/gollama-bubbletea/internal/settings"
	"github.com/kevensen/gollama-bubbletea/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	ctx := context.Background()

	// Load settings to get Ollama URL
	appSettings, err := settings.Load()
	if err != nil {
		log.Printf("Warning: Could not load settings, using defaults: %v", err)
		appSettings = settings.DefaultSettings()
	}

	// Use default model for now - will be overridden by settings if valid
	defaultModel := "tinyllama:latest"

	// Always try to create bot, even if no connection
	// The TUI will handle the no-connection case
	ollamaURL := appSettings.OllamaURL

	b, err := bot.NewBot(ctx, ollamaURL, defaultModel)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	t := tui.New(b)

	p := tea.NewProgram(t)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
