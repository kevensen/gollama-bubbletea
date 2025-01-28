package motd

import (
	"math/rand"

	"github.com/kevensen/gollama-bubbletea/internal/tools"

	"github.com/parakeet-nest/parakeet/llm"
)

const name = "motd"

var messages = []string{
	"Hello, Gopher!",
	"Have a great day!",
	"Make it happen!",
	"Keep coding and keep smiling!",
	"Go is awesome!",
}

func init() {
	t := &MOTDTool{
		messages: messages,
		tool: llm.Tool{
			Type: "function",
			Function: llm.Function{
				Name:        "motd",
				Description: "Get a message of the day",
			},
		},
		name: "motd",
	}

	tools.Register(name, t)
}

type MOTDTool struct {
	messages []string
	tool     llm.Tool
	name     string
}

func (MOTDTool) Call(map[string]interface{}) (string, error) {
	greeting := messages[rand.Intn(len(messages))]
	return greeting, nil
}

func (t *MOTDTool) LLMTool() llm.Tool {
	return t.tool
}

func (t *MOTDTool) Name() string {
	return t.name
}
