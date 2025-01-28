package models

import (
	"fmt"
	"slices"

	"github.com/parakeet-nest/parakeet/llm"
)

type Manager struct {
	list         llm.ModelList
	currentModel string
}

func NewManager(ollamaURL string, initialModel string) (*Manager, error) {
	list, _, err := llm.GetModelsList(ollamaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %v", err)
	}

	mgr := &Manager{
		list: list,
	}

	// Check if the initial model exists
	if !slices.Contains(mgr.ModelNames(), initialModel) {
		return nil, fmt.Errorf("model not found: %s", initialModel)
	}

	mgr.currentModel = initialModel
	return mgr, nil
}

func (m *Manager) ModelNames() []string {
	var ms []string
	for _, model := range m.list.Models {
		ms = append(ms, model.Name)
	}

	return ms
}

func (m *Manager) MaxModelNameLength() int {
	maxLen := 0
	for _, model := range m.list.Models {
		if len(model.Name) > maxLen {
			maxLen = len(model.Name)
		}
	}

	return maxLen
}

func (m *Manager) CurrentModel() string {
	return m.currentModel
}

func (m *Manager) UseModel(model string) error {
	if !slices.Contains(m.ModelNames(), model) {
		return fmt.Errorf("model not found: %s", model)
	}

	m.currentModel = model
	return nil
}

func (m *Manager) ModelExists(model string) bool {
	return slices.Contains(m.ModelNames(), model)
}
