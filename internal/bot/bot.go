package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

// ChromaDBQuery represents a query to ChromaDB
type ChromaDBQuery struct {
	QueryTexts []string `json:"query_texts"`
	NResults   int      `json:"n_results"`
}

// ChromaDBResult represents search results from ChromaDB
type ChromaDBResult struct {
	Documents [][]string                 `json:"documents"`
	Metadatas [][]map[string]interface{} `json:"metadatas"`
	Distances [][]float64                `json:"distances"`
}

// SendRAGMessage sends a message with RAG context from ChromaDB
func (b *Bot) SendRAGMessage(ctx context.Context, role, message, chromaDBURL string) (*llm.Answer, error) {
	// First, search ChromaDB for relevant context
	ragContext, err := b.searchChromaDB(chromaDBURL, message)
	if err != nil {
		// If RAG search fails, fall back to regular message but add a note
		contextualMessage := fmt.Sprintf("(RAG search failed: %v)\n\n%s", err, message)
		return b.SendMessage(ctx, role, contextualMessage)
	}

	// Enhance the message with RAG context
	var enhancedMessage string
	if ragContext != "" {
		enhancedMessage = fmt.Sprintf("Context from knowledge base:\n%s\n\nUser question: %s", ragContext, message)
	} else {
		enhancedMessage = fmt.Sprintf("(No relevant context found in knowledge base)\n\nUser question: %s", message)
	}

	// Send the enhanced message
	return b.SendMessage(ctx, role, enhancedMessage)
}

// searchChromaDB searches the ChromaDB instance for relevant context
func (b *Bot) searchChromaDB(chromaDBURL, query string) (string, error) {
	if chromaDBURL == "" {
		return "", fmt.Errorf("ChromaDB URL not configured")
	}

	// Create ChromaDB query
	chromaQuery := ChromaDBQuery{
		QueryTexts: []string{query},
		NResults:   3, // Get top 3 most relevant results
	}

	queryData, err := json.Marshal(chromaQuery)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ChromaDB query: %v", err)
	}

	// Make request to ChromaDB (assuming a collection named "documents")
	// This is a basic implementation - you may need to adjust the endpoint based on your ChromaDB setup
	searchURL := fmt.Sprintf("%s/api/v1/collections/documents/query", chromaDBURL)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(queryData))
	if err != nil {
		return "", fmt.Errorf("failed to create ChromaDB request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query ChromaDB: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ChromaDB returned status %d", resp.StatusCode)
	}

	// Parse response
	var result ChromaDBResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode ChromaDB response: %v", err)
	}

	// Extract and format the relevant documents
	var contextBuilder strings.Builder
	if len(result.Documents) > 0 && len(result.Documents[0]) > 0 {
		for i, doc := range result.Documents[0] {
			if i < 3 { // Limit to top 3 results
				contextBuilder.WriteString(fmt.Sprintf("Document %d: %s\n", i+1, doc))
			}
		}
	}

	return contextBuilder.String(), nil
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
