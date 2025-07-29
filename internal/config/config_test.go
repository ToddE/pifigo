package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// --- Test Case 1: Valid Configuration File ---
	validYAML := `
boot_manager:
  timeout_seconds: 300
network:
  wireless_interface: "wlan_test"
`
	// Create a temporary directory and file for the test.
	tmpDir := t.TempDir()
	validConfigFile := filepath.Join(tmpDir, "valid_config.yaml")
	if err := os.WriteFile(validConfigFile, []byte(validYAML), 0644); err != nil {
		t.Fatalf("Failed to write valid test config: %v", err)
	}

	// Load the valid config.
	cfg, err := LoadConfig(validConfigFile)
	if err != nil {
		t.Fatalf("LoadConfig failed with a valid config file: %v", err)
	}

	// Assert that the values were parsed correctly.
	if cfg.BootManager.TimeoutSeconds != 300 {
		t.Errorf("Expected TimeoutSeconds to be 300, got %d", cfg.BootManager.TimeoutSeconds)
	}
	if cfg.Network.WirelessInterface != "wlan_test" {
		t.Errorf("Expected WirelessInterface to be 'wlan_test', got '%s'", cfg.Network.WirelessInterface)
	}

	// --- Test Case 2: Invalid (Malformed) YAML ---
	invalidYAML := `
boot_manager:
  timeout_seconds: 300
network:
  - wireless_interface: "wlan_test" # Incorrect indentation
`
	invalidConfigFile := filepath.Join(tmpDir, "invalid_config.yaml")
	if err := os.WriteFile(invalidConfigFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid test config: %v", err)
	}

	// Attempt to load the invalid config and assert that it produces an error.
	_, err = LoadConfig(invalidConfigFile)
	if err == nil {
		t.Errorf("LoadConfig succeeded with invalid YAML, but an error was expected")
	}

	// --- Test Case 3: File Not Found ---
	_, err = LoadConfig(filepath.Join(tmpDir, "non_existent_file.yaml"))
	if err == nil {
		t.Errorf("LoadConfig succeeded with a non-existent file, but an error was expected")
	}
}
