package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/kevensen/gollama-bubbletea/internal/bot"
	"github.com/kevensen/gollama-bubbletea/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	ctx := context.Background()
	ollamaHost := flag.String("ollama_host", "", "The host for Ollama")
	ollamaPort := flag.String("ollama_port", "", "The port for Ollama")
	ollamaModel := flag.String("ollama_model", "tinyllama:latest", "The model for Ollama")
	flag.Parse()

	if *ollamaHost == "" {
		*ollamaHost = os.Getenv("OLLAMA_HOST")
	}

	if *ollamaHost == "" {
		log.Fatal("OLLAMA_HOST environment variable or flag is required")
	}

	if *ollamaPort == "" {
		*ollamaPort = os.Getenv("OLLAMA_PORT")
	}

	if *ollamaPort == "" {
		log.Fatal("OLLAMA_PORT environment variable or flag is required")
	}

	ollamaAddr := *ollamaHost + ":" + *ollamaPort

	b, err := bot.NewBot(ctx, ollamaAddr, *ollamaModel)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	t := tui.New(b)

	p := tea.NewProgram(t)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
