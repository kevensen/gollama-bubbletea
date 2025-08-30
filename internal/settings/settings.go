package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds the application settings
type Settings struct {
	LastModel   string `json:"lastModel"`
	RAGEnabled  bool   `json:"ragEnabled"`
	OllamaURL   string `json:"ollamaURL"`
	ChromaDBURL string `json:"chromaDBURL"`
	DarkMode    bool   `json:"darkMode"`
}

// Default settings
func DefaultSettings() *Settings {
	return &Settings{
		LastModel:   "",
		RAGEnabled:  false,
		OllamaURL:   "", // No default URL - user must configure
		ChromaDBURL: "", // No default ChromaDB URL - user must configure
		DarkMode:    false,
	}
}

// getSettingsPath returns the path to the settings file
func getSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".config", "gollama")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "settings.json"), nil
}

// Load reads settings from the settings file
func Load() (*Settings, error) {
	settingsPath, err := getSettingsPath()
	if err != nil {
		return DefaultSettings(), err
	}

	// If file doesn't exist, return default settings
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return DefaultSettings(), nil
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return DefaultSettings(), err
	}

	settings := &Settings{}
	if err := json.Unmarshal(data, settings); err != nil {
		return DefaultSettings(), err
	}

	return settings, nil
}

// Save writes settings to the settings file
func (s *Settings) Save() error {
	settingsPath, err := getSettingsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

// SetLastModel updates the last model and saves settings
func (s *Settings) SetLastModel(model string) error {
	s.LastModel = model
	return s.Save()
}

// SetRAGEnabled updates the RAG enabled state and saves settings
func (s *Settings) SetRAGEnabled(enabled bool) error {
	s.RAGEnabled = enabled
	return s.Save()
}

// SetOllamaURL updates the Ollama URL and saves settings
func (s *Settings) SetOllamaURL(url string) error {
	s.OllamaURL = url
	return s.Save()
}

// SetDarkMode updates the dark mode state and saves settings
func (s *Settings) SetDarkMode(enabled bool) error {
	s.DarkMode = enabled
	return s.Save()
}

// SetChromaDBURL updates the ChromaDB URL and saves settings
func (s *Settings) SetChromaDBURL(url string) error {
	s.ChromaDBURL = url
	return s.Save()
}
