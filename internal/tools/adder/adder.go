package adder

import (
	"fmt"

	"github.com/kevensen/gollama-bubbletea/internal/tools"
	"github.com/parakeet-nest/parakeet/llm"
)

const name = "adder"

func init() {
	t := &Tool{
		tool: llm.Tool{
			Type: "function",
			Function: llm.Function{
				Name:        name,
				Description: "Add any two integers togther and get the sum.",
				Parameters: llm.Parameters{
					Type: "object",
					Properties: map[string]llm.Property{
						"a": {
							Type:        "number",
							Description: "first operand",
						},
						"b": {
							Type:        "number",
							Description: "second operand",
						},
					},
					Required: []string{"a", "b"},
				},
			},
		},
		name: name,
	}

	tools.Register(name, t)
}

type Tool struct {
	tool llm.Tool
	name string
}

func (Tool) Call(args map[string]interface{}) (string, error) {
	a := args["a"].(float64)
	b := args["b"].(float64)
	sum := a + b
	return fmt.Sprintf("%f", sum), nil
}

func addInterfaces(ifaces []interface{}) float64 {
	var sum float64
	for _, v := range ifaces {
		if _, ok := v.(float64); ok {
			sum += v.(float64)
		}

		if _, ok := v.(int); ok {
			sum += float64(v.(int))
		}

		if _, ok := v.([]interface{}); ok {
			sum += addInterfaces(v.([]interface{}))
		}
	}
	return sum
}

func (t *Tool) LLMTool() llm.Tool {
	return t.tool
}

func (t *Tool) Name() string {
	return t.name
}
