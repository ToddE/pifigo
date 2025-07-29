package locale

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLanguageStrings(t *testing.T) {
	// --- Test Case 1: Valid Locale File ---
	validYAML := `
page_title: "Test Title"
connect_button_text: "Connect Now"
`
	tmpDir := t.TempDir()
	validLocaleFile := filepath.Join(tmpDir, "valid_locale.yaml")
	if err := os.WriteFile(validLocaleFile, []byte(validYAML), 0644); err != nil {
		t.Fatalf("Failed to write valid test locale file: %v", err)
	}

	// Load the valid locale.
	lang, err := LoadLanguageStrings(validLocaleFile)
	if err != nil {
		t.Fatalf("LoadLanguageStrings failed with a valid locale file: %v", err)
	}

	// Assert that the values were parsed correctly.
	if lang.PageTitle != "Test Title" {
		t.Errorf("Expected PageTitle to be 'Test Title', got '%s'", lang.PageTitle)
	}
	if lang.ConnectButtonText != "Connect Now" {
		t.Errorf("Expected ConnectButtonText to be 'Connect Now', got '%s'", lang.ConnectButtonText)
	}

	// --- Test Case 2: Invalid (Malformed) YAML ---
	invalidYAML := `
page_title: "Test Title"
  connect_button_text: "Connect Now" # Incorrect indentation
`
	invalidLocaleFile := filepath.Join(tmpDir, "invalid_locale.yaml")
	if err := os.WriteFile(invalidLocaleFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid test locale file: %v", err)
	}

	// Attempt to load the invalid locale and assert that it produces an error.
	_, err = LoadLanguageStrings(invalidLocaleFile)
	if err == nil {
		t.Errorf("LoadLanguageStrings succeeded with invalid YAML, but an error was expected")
	}
}
