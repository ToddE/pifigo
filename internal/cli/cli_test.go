package cli

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestFS creates a temporary directory structure to simulate the real filesystem
// for testing purposes. It returns the path to the temp root and a cleanup function.
func setupTestFS(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	// Override the constants for the duration of the test
	origSavedDir := savedNetworksDir
	origSymlink := lastGoodSymlink
	origActiveConfig := activeClientConfig

	testSavedDir := filepath.Join(tmpDir, "saved_networks")
	testSymlink := filepath.Join(tmpDir, "last-good-wifi.yaml")
	testActiveConfig := filepath.Join(tmpDir, "99-pifigo-client.yaml")

	// Create the saved networks directory for tests
	if err := os.MkdirAll(testSavedDir, 0755); err != nil {
		t.Fatalf("Failed to create test saved_networks dir: %v", err)
	}

	// Monkey-patch the global constants
	savedNetworksDir = testSavedDir
	lastGoodSymlink = testSymlink
	activeClientConfig = testActiveConfig

	cleanup := func() {
		savedNetworksDir = origSavedDir
		lastGoodSymlink = origSymlink
		activeClientConfig = origActiveConfig
	}

	return tmpDir, cleanup
}

func TestCliFunctions(t *testing.T) {
	_, cleanup := setupTestFS(t)
	defer cleanup()

	// Helper to capture stdout
	captureOutput := func(f func()) string {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		f()

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		return buf.String()
	}

	t.Run("ListSavedNetworks", func(t *testing.T) {
		// Test with no networks
		output := captureOutput(func() {
			if err := ListSavedNetworks(); err != nil {
				t.Fatalf("ListSavedNetworks failed: %v", err)
			}
		})
		if !strings.Contains(output, "No networks have been saved yet.") {
			t.Errorf("Expected 'no networks' message, got: %s", output)
		}

		// Add some networks
		os.WriteFile(filepath.Join(savedNetworksDir, "HomeWiFi.yaml"), []byte("..."), 0644)
		os.WriteFile(filepath.Join(savedNetworksDir, "OfficeWiFi.yaml"), []byte("..."), 0644)

		output = captureOutput(func() {
			if err := ListSavedNetworks(); err != nil {
				t.Fatalf("ListSavedNetworks failed: %v", err)
			}
		})
		if !strings.Contains(output, "HomeWiFi") || !strings.Contains(output, "OfficeWiFi") {
			t.Errorf("Expected to see HomeWiFi and OfficeWiFi, got: %s", output)
		}
	})

	t.Run("SetAndShowLastGood", func(t *testing.T) {
		// Test setting a non-existent network
		err := SetLastGood("GuestWiFi")
		if err == nil {
			t.Fatal("Expected error when setting non-existent network, but got none")
		}

		// Set a valid network
		err = SetLastGood("HomeWiFi")
		if err != nil {
			t.Fatalf("SetLastGood failed: %v", err)
		}

		// Check if the symlink is correct
		target, _ := os.Readlink(lastGoodSymlink)
		if filepath.Base(target) != "HomeWiFi.yaml" {
			t.Errorf("Symlink points to wrong file: %s", target)
		}

		// Check the output of ShowLastGood
		output := captureOutput(func() {
			if err := ShowLastGood(); err != nil {
				t.Fatalf("ShowLastGood failed: %v", err)
			}
		})
		if !strings.Contains(output, "Last Good Network: HomeWiFi") {
			t.Errorf("Expected 'Last Good Network: HomeWiFi', got: %s", output)
		}
	})

	t.Run("ForgetNetwork", func(t *testing.T) {
		// Forget a network that is not the last-good
		err := ForgetNetwork("OfficeWiFi")
		if err != nil {
			t.Fatalf("ForgetNetwork failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(savedNetworksDir, "OfficeWiFi.yaml")); !os.IsNotExist(err) {
			t.Error("OfficeWiFi.yaml was not deleted")
		}

		// Forget the network that IS the last-good
		err = ForgetNetwork("HomeWiFi")
		if err != nil {
			t.Fatalf("ForgetNetwork failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(savedNetworksDir, "HomeWiFi.yaml")); !os.IsNotExist(err) {
			t.Error("HomeWiFi.yaml was not deleted")
		}
		if _, err := os.Lstat(lastGoodSymlink); !os.IsNotExist(err) {
			t.Error("Symlink was not deleted when its target was forgotten")
		}
	})

	t.Run("ShowStatus", func(t *testing.T) {
		// Test hotspot mode status
		output := captureOutput(func() {
			if err := ShowStatus(); err != nil {
				log.Fatalf("ShowStatus failed: %v", err)
			}
		})
		if !strings.Contains(output, "Status: Hotspot Mode") {
			t.Errorf("Expected 'Hotspot Mode', got: %s", output)
		}

		// Test client mode status
		os.WriteFile(activeClientConfig, []byte("..."), 0644)
		output = captureOutput(func() {
			if err := ShowStatus(); err != nil {
				log.Fatalf("ShowStatus failed: %v", err)
			}
		})
		if !strings.Contains(output, "Status: Client Mode") {
			t.Errorf("Expected 'Client Mode', got: %s", output)
		}
		os.Remove(activeClientConfig)
	})
}
