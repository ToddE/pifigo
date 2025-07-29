package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"pifigo/internal/config"
	"strings"
	"testing"
)

// setupTestServer creates a mock server instance with a temporary filesystem for testing handlers.
func setupTestServer(t *testing.T) *Server {
	tmpDir := t.TempDir()

	// Create mock config and locale files
	mockConfigContent := `
paths:
  web_root: "testdata"
  locales_dir: "testdata/locales"
network:
  wireless_interface: "wlan_test"
language: "en"
`
	mockLocaleContent := `
page_title: "Test"
reconnect_button_text: "Reconnect"
no_saved_connections_message: "None saved"
`

	// Create directories
	os.MkdirAll(filepath.Join(tmpDir, "testdata", "locales"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "etc", "pifigo"), 0755)

	// Write mock files
	configPath := filepath.Join(tmpDir, "etc", "pifigo", "config.yaml")
	os.WriteFile(configPath, []byte(mockConfigContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "testdata", "locales", "en.yaml"), []byte(mockLocaleContent), 0644)

	// Load the mock config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load mock config: %v", err)
	}

	// Override paths to use temp directory
	cfg.Paths.LocalesDir = filepath.Join(tmpDir, "testdata", "locales")

	stopSignal := make(chan bool, 1)
	return NewServer(cfg, stopSignal)
}

// setupTestNetDirs creates temporary directories for network files and overrides the package variables.
func setupTestNetDirs(t *testing.T) func() {
	tmpDir := t.TempDir()

	origSavedDir := savedNetworksDir
	origActiveConfig := activeClientConfig
	origSymlink := lastGoodSymlink
	origTemplate := netplanTemplate

	savedNetworksDir = filepath.Join(tmpDir, "saved_networks")
	activeClientConfig = filepath.Join(tmpDir, "99-pifigo-client.yaml")
	lastGoodSymlink = filepath.Join(tmpDir, "last-good-wifi.yaml")
	netplanTemplate = filepath.Join(tmpDir, "netplan.tpl")

	os.MkdirAll(savedNetworksDir, 0755)
	os.WriteFile(netplanTemplate, []byte("ssid: {{.SSID}}"), 0644)

	return func() {
		savedNetworksDir = origSavedDir
		activeClientConfig = origActiveConfig
		lastGoodSymlink = origSymlink
		netplanTemplate = origTemplate
	}
}

// mockExecCommand replaces the real exec.Command with a harmless one for testing.
func mockExecCommand(t *testing.T) func() {
	originalExec := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		// Instead of running the real command, run '/bin/true', which does nothing and exits successfully.
		return exec.Command("/bin/true")
	}
	// Return a cleanup function to restore the original execCommand.
	return func() {
		execCommand = originalExec
	}
}

func TestServeDataAPI(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/data", nil)
	rr := httptest.NewRecorder()

	server.serveDataAPI(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), `"PageTitle":"Test"`) {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestHandleConnect(t *testing.T) {
	cleanupNetDirs := setupTestNetDirs(t)
	defer cleanupNetDirs()

	cleanupExec := mockExecCommand(t)
	defer cleanupExec()

	server := setupTestServer(t)

	formData := url.Values{}
	formData.Set("ssid", "MyTestNetwork")
	formData.Set("password", "password123")

	req := httptest.NewRequest("POST", "/connect", strings.NewReader(formData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleConnect(rr, req)

	// Check that the handler completed successfully.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	profilePath := filepath.Join(savedNetworksDir, "MyTestNetwork.yaml")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Errorf("Expected network profile to be saved at %s, but it was not", profilePath)
	}

	if _, err := os.Lstat(lastGoodSymlink); os.IsNotExist(err) {
		t.Errorf("Expected last-good symlink to be created at %s, but it was not", lastGoodSymlink)
	}
}

func TestHandleListSavedNetworks(t *testing.T) {
	cleanup := setupTestNetDirs(t)
	defer cleanup()

	server := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/saved_networks", nil)
	rr := httptest.NewRecorder()

	// Test with no saved networks
	server.handleListSavedNetworks(rr, req)
	if !strings.Contains(rr.Body.String(), "None saved") {
		t.Errorf("Expected 'None saved' message, got: %s", rr.Body.String())
	}

	// Add some saved networks
	os.WriteFile(filepath.Join(savedNetworksDir, "HomeWiFi.yaml"), []byte("..."), 0644)
	os.WriteFile(filepath.Join(savedNetworksDir, "OfficeWiFi.yaml"), []byte("..."), 0644)

	rr = httptest.NewRecorder() // Reset the recorder
	server.handleListSavedNetworks(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, "HomeWiFi") || !strings.Contains(body, "OfficeWiFi") {
		t.Errorf("Expected to see HomeWiFi and OfficeWiFi, got: %s", body)
	}
	if !strings.Contains(body, "Reconnect") {
		t.Errorf("Expected to see Reconnect button text, got: %s", body)
	}
}

func TestHandleReconnect(t *testing.T) {
	cleanupNetDirs := setupTestNetDirs(t)
	defer cleanupNetDirs()

	cleanupExec := mockExecCommand(t)
	defer cleanupExec()

	server := setupTestServer(t)

	// Create a fake saved network profile
	savedSSID := "MyHomeWiFi"
	savedContent := "network: MyHomeWiFi"
	profilePath := filepath.Join(savedNetworksDir, savedSSID+".yaml")
	os.WriteFile(profilePath, []byte(savedContent), 0644)

	// Create the form data for the POST request
	formData := url.Values{}
	formData.Set("ssid", savedSSID)

	req := httptest.NewRequest("POST", "/reconnect", strings.NewReader(formData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleReconnect(rr, req)

	// Check that the handler completed successfully.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the active config was written with the correct content
	activeContent, err := os.ReadFile(activeClientConfig)
	if err != nil {
		t.Fatalf("Could not read active client config: %v", err)
	}
	if string(activeContent) != savedContent {
		t.Errorf("Expected active config content to be '%s', got '%s'", savedContent, string(activeContent))
	}

	// Check that the symlink was updated
	target, _ := os.Readlink(lastGoodSymlink)
	if filepath.Base(target) != savedSSID+".yaml" {
		t.Errorf("Symlink points to wrong file: expected %s.yaml, got %s", savedSSID, filepath.Base(target))
	}
}
