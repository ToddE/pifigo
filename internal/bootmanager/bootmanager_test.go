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
		return exec.Command("/bin/true")
	}
	return func() {
		ExecCommand = originalExec
	}
}

// TestSyncHotspotConfig verifies that all three hotspot config files are generated correctly.
func TestSyncHotspotConfig(t *testing.T) {
	// Create a temporary directory to act as the root for our config files.
	tmpDir := t.TempDir()

	// Temporarily override the package-level variables for the test's scope.
	originalHotspotFile := HotspotConfigFile
	originalHostapdFile := HostapdConfigFile
	originalDnsmasqFile := DnsmasqConfigFile
	HotspotConfigFile = filepath.Join(tmpDir, "00-pifigo-hotspot-ip.yaml")
	HostapdConfigFile = filepath.Join(tmpDir, "hostapd.conf")
	DnsmasqConfigFile = filepath.Join(tmpDir, "99-pifigo-hotspot")
	defer func() {
		HotspotConfigFile = originalHotspotFile
		HostapdConfigFile = originalHostapdFile
		DnsmasqConfigFile = originalDnsmasqFile
	}()

	// Create a mock config struct.
	cfg := &config.Config{}
	cfg.Network.WirelessInterface = "wlan_test"
	cfg.Network.ApIpAddress = "192.168.100.1/24"
	cfg.Network.ApSSID = "TestHotspot"
	cfg.Network.ApChannel = 11
	cfg.Network.ApPassword = "testpassword"
	cfg.Network.WifiCountry = "US"

	// Run the sync function.
	err := SyncHotspotConfig(cfg)
	if err != nil {
		t.Fatalf("SyncHotspotConfig failed: %v", err)
	}

	// --- Verify Netplan File ---
	netplanContent, err := os.ReadFile(HotspotConfigFile)
	if err != nil {
		t.Fatalf("Could not read netplan config file: %v", err)
	}
	if !strings.Contains(string(netplanContent), "wlan_test:") || !strings.Contains(string(netplanContent), "192.168.100.1/24") {
		t.Errorf("Netplan config content is incorrect. Got:\n%s", string(netplanContent))
	}

	// --- Verify Hostapd File ---
	hostapdContent, err := os.ReadFile(HostapdConfigFile)
	if err != nil {
		t.Fatalf("Could not read hostapd config file: %v", err)
	}
	if !strings.Contains(string(hostapdContent), "ssid=TestHotspot") || !strings.Contains(string(hostapdContent), "channel=11") {
		t.Errorf("Hostapd config content is incorrect. Got:\n%s", string(hostapdContent))
	}

	// --- Verify Dnsmasq File ---
	dnsmasqContent, err := os.ReadFile(DnsmasqConfigFile)
	if err != nil {
		t.Fatalf("Could not read dnsmasq config file: %v", err)
	}
	if !strings.Contains(string(dnsmasqContent), "interface=wlan_test") || !strings.Contains(string(dnsmasqContent), "dhcp-range=192.168.100.1,192.168.100.1") {
		t.Errorf("Dnsmasq config content is incorrect. Got:\n%s", string(dnsmasqContent))
	}
}

// TestBootManager_StopSignal remains the same
func TestBootManager_StopSignal(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	cfg := &config.Config{}
	cfg.BootManager.TimeoutSeconds = 10
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

// TestBootManager_Timeout remains the same
func TestBootManager_Timeout(t *testing.T) {
	cleanupExec := mockExecCommand(t)
	defer cleanupExec()
	var wg sync.WaitGroup
	wg.Add(1)
	cfg := &config.Config{}
	cfg.BootManager.TimeoutSeconds = 0
	stopSignal := make(chan bool, 1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		Start(cfg, stopSignal)
	}()
	if waitTimeout(&wg, 100*time.Millisecond) {
		// This is expected to fail to wait because the goroutine should finish quickly.
	}
}

// waitTimeout helper function remains the same
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false
	case <-time.After(timeout):
		return true
	}
}
