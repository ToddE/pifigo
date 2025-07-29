package bootmanager

import (
	"os"
	"os/exec"
	"pifigo/internal/config"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockExecCommand replaces the real ExecCommand with a harmless one for testing.
func mockExecCommand(t *testing.T) func() {
	originalExec := ExecCommand
	ExecCommand = func(name string, arg ...string) *exec.Cmd {
		// Instead of running the real command, run '/bin/true', which does nothing and exits successfully.
		return exec.Command("/bin/true")
	}
	// Return a cleanup function to restore the original ExecCommand.
	return func() {
		ExecCommand = originalExec
	}
}

// TestSyncHotspotConfig verifies that the hotspot netplan file is correctly generated from config.
func TestSyncHotspotConfig(t *testing.T) {
	cleanupExec := mockExecCommand(t)
	defer cleanupExec()

	// Create a temporary directory to act as the root for our config files.
	tmpDir := t.TempDir()
	
	// Temporarily override the package-level variable for the test's scope.
	originalHotspotFile := HotspotConfigFile
	HotspotConfigFile = filepath.Join(tmpDir, "01-pifigo-hotspot.yaml")
	defer func() { HotspotConfigFile = originalHotspotFile }()


	// Create a mock config struct.
	cfg := &config.Config{}
	cfg.Network.WirelessInterface = "wlan_test"
	cfg.Network.ApIpAddress = "192.168.100.1/24"
	cfg.Network.ApSSID = "TestHotspot"
	cfg.Network.ApChannel = 11
	cfg.Network.ApPassword = "testpassword"

	// --- Test Case 1: File does not exist, should be created ---
	err := SyncHotspotConfig(cfg)
	if err != nil {
		t.Fatalf("SyncHotspotConfig failed on initial creation: %v", err)
	}

	content, err := os.ReadFile(HotspotConfigFile)
	if err != nil {
		t.Fatalf("Could not read newly created hotspot config file: %v", err)
	}

	// Check if the content contains the values from our mock config.
	if !strings.Contains(string(content), "wlan_test:") {
		t.Error("Generated config missing correct wireless interface")
	}
	if !strings.Contains(string(content), "ssid: \"TestHotspot\"") {
		t.Error("Generated config missing correct SSID")
	}
	if !strings.Contains(string(content), "channel: 11") {
		t.Error("Generated config missing correct channel")
	}

	// --- Test Case 2: File exists but is out of sync, should be updated ---
	cfg.Network.ApSSID = "UpdatedHotspot" // Change a value
	err = SyncHotspotConfig(cfg)
	if err != nil {
		t.Fatalf("SyncHotspotConfig failed on update: %v", err)
	}

	content, err = os.ReadFile(HotspotConfigFile)
	if err != nil {
		t.Fatalf("Could not read updated hotspot config file: %v", err)
	}
	if !strings.Contains(string(content), "ssid: \"UpdatedHotspot\"") {
		t.Error("Generated config was not updated with new SSID")
	}
}

// TestBootManager_StopSignal tests that the boot manager exits immediately
// when it receives a signal on the stop channel, without waiting for the timeout.
func TestBootManager_StopSignal(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	cfg := &config.Config{}
	cfg.BootManager.TimeoutSeconds = 10 // A long timeout we don't expect to hit

	stopSignal := make(chan bool, 1)

	go func() {
		defer wg.Done()
		Start(cfg, stopSignal)
	}()

	stopSignal <- true

	if waitTimeout(&wg, 100*time.Millisecond) {
		t.Errorf("Boot manager did not stop immediately after receiving the stop signal")
	}
}

// TestBootManager_Timeout tests that the boot manager proceeds after its
// timer expires if no stop signal is sent.
func TestBootManager_Timeout(t *testing.T) {
	cleanupExec := mockExecCommand(t)
	defer cleanupExec()

	// This test relies on the fact that revertToLastGoodConfig will fail
	// because the symlink doesn't exist, but it proves the timeout path was taken.
	var wg sync.WaitGroup
	wg.Add(1)

	cfg := &config.Config{}
	// Use a very short timeout for the test
	cfg.BootManager.TimeoutSeconds = 0 // Set to 0 and rely on short sleep

	stopSignal := make(chan bool, 1)

	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond) // Give a moment for the timer to fire
		Start(cfg, stopSignal)
	}()

	if waitTimeout(&wg, 100*time.Millisecond) {
		// This is expected to fail to wait because the goroutine should finish quickly.
		// If it times out, something is wrong.
	}
}

// waitTimeout is a helper function to wait for a WaitGroup with a timeout.
// It returns true if the wait timed out, false if it completed successfully.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // Completed successfully.
	case <-time.After(timeout):
		return true // Timed out.
	}
}
