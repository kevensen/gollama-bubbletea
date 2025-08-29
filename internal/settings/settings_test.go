package settings

import (
	"os"
	"testing"
)

func TestSettingsLoadAndSave(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test default settings
	settings := DefaultSettings()
	if settings.LastModel != "" {
		t.Errorf("Expected empty LastModel, got %s", settings.LastModel)
	}
	if settings.RAGEnabled != false {
		t.Errorf("Expected RAGEnabled to be false, got %t", settings.RAGEnabled)
	}

	// Test saving settings
	settings.LastModel = "test-model"
	settings.RAGEnabled = true

	err := settings.Save()
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Test loading settings
	loadedSettings, err := Load()
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if loadedSettings.LastModel != "test-model" {
		t.Errorf("Expected LastModel to be 'test-model', got %s", loadedSettings.LastModel)
	}
	if loadedSettings.RAGEnabled != true {
		t.Errorf("Expected RAGEnabled to be true, got %t", loadedSettings.RAGEnabled)
	}

	// Test convenience methods
	err = loadedSettings.SetLastModel("another-model")
	if err != nil {
		t.Fatalf("Failed to set last model: %v", err)
	}

	err = loadedSettings.SetRAGEnabled(false)
	if err != nil {
		t.Fatalf("Failed to set RAG enabled: %v", err)
	}

	// Verify changes were saved
	finalSettings, err := Load()
	if err != nil {
		t.Fatalf("Failed to load final settings: %v", err)
	}

	if finalSettings.LastModel != "another-model" {
		t.Errorf("Expected LastModel to be 'another-model', got %s", finalSettings.LastModel)
	}
	if finalSettings.RAGEnabled != false {
		t.Errorf("Expected RAGEnabled to be false, got %t", finalSettings.RAGEnabled)
	}
}

func TestSettingsLoadNonExistent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test loading when no settings file exists
	settings, err := Load()
	if err != nil {
		t.Fatalf("Load should not error when file doesn't exist: %v", err)
	}

	// Should return default settings
	if settings.LastModel != "" {
		t.Errorf("Expected empty LastModel, got %s", settings.LastModel)
	}
	if settings.RAGEnabled != false {
		t.Errorf("Expected RAGEnabled to be false, got %t", settings.RAGEnabled)
	}
}
