package tools

import (
	"github.com/parakeet-nest/parakeet/llm"
	"golang.org/x/exp/maps"
)

var registered map[string]CallableTool

type CallableTool interface {
	Call(map[string]interface{}) (string, error)
	LLMTool() llm.Tool
	Name() string
}

func init() {
	registered = make(map[string]CallableTool, 0)
	registered["<none>"] = &EmptyTool{}
}

func Register(name string, t CallableTool) {
	registered[name] = t
}

func Registered() map[string]CallableTool {
	return registered
}

type EmptyTool struct{}

func (e EmptyTool) Call(map[string]interface{}) (string, error) {
	return "", nil
}

func (e EmptyTool) LLMTool() llm.Tool {
	return llm.Tool{}
}

func (e EmptyTool) Name() string {
	return "<none>"
}

type ToolManager struct {
	tools        map[string]CallableTool
	selectedTool string
}

func NewToolManager(tools map[string]CallableTool) *ToolManager {
	return &ToolManager{
		tools:        tools,
		selectedTool: "<none>",
	}
}

func (tm *ToolManager) ByName(name string) (CallableTool, bool) {
	t, ok := registered[name]
	return t, ok
}

func (tm *ToolManager) Select(name string) (ok bool) {
	if _, ok = tm.ByName(name); ok {
		tm.selectedTool = name
	}
	return
}

func (tm *ToolManager) Current() CallableTool {
	return tm.tools[tm.selectedTool]
}

func (tm *ToolManager) CurrentName() string {
	return tm.selectedTool
}

func (tm *ToolManager) ToolNames() []string {
	return maps.Keys(tm.tools)
}

func (tm *ToolManager) ToolCount() int {
	return len(tm.tools)
}

func (tm *ToolManager) LLMTools() []llm.Tool {
	var tools []llm.Tool
	for _, t := range tm.tools {
		tools = append(tools, t.LLMTool())
	}
	return tools
}

func (tm *ToolManager) UsingTool() bool {
	return tm.selectedTool != "<none>"
}
