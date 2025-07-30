package bootmanager

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"pifigo/internal/config"
)

// Exported variables to allow for mocking during tests.
var (
	ExecCommand       = exec.Command
	HotspotConfigFile = "/etc/netplan/01-pifigo-hotspot.yaml"
)

const (
	lastGoodSymlink    = "/etc/pifigo/last-good-wifi.yaml"
	activeClientConfig = "/etc/netplan/99-pifigo-client.yaml"
)

// SyncHotspotConfig ensures the hotspot netplan file is up-to-date with the main config.
func SyncHotspotConfig(cfg *config.Config) error {
	log.Println("Syncing hotspot configuration...")

	if err := validateHotspotConfig(cfg); err != nil {
		return fmt.Errorf("invalid hotspot configuration: %w", err)
	}

	// CORRECTED: Added the 'channel' field to the template.
	hotspotTemplate := `network:
  version: 2
  renderer: NetworkManager
  wifis:
    {{.Network.WirelessInterface}}:
      dhcp4: no
      addresses:
        - {{.Network.ApIpAddress}}
      access-points:
        "{{.Network.ApSSID}}":
          password: "{{.Network.ApPassword}}"
          mode: ap
          channel: {{.Network.ApChannel}}
`
	tmpl, err := template.New("hotspot").Parse(hotspotTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse internal hotspot template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("failed to execute hotspot template: %w", err)
	}
	expectedContent := buf.Bytes()

	currentContent, err := os.ReadFile(HotspotConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read current hotspot config: %w", err)
	}

	if !bytes.Equal(currentContent, expectedContent) {
		log.Println("Hotspot configuration is out of sync. Updating...")
		if err := os.WriteFile(HotspotConfigFile, expectedContent, 0644); err != nil {
			return fmt.Errorf("failed to write updated hotspot config: %w", err)
		}

		cmd := ExecCommand("netplan", "apply")
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Printf("ERROR applying synced hotspot config: %s", string(output))
			return fmt.Errorf("failed to apply synced hotspot config: %w", err)
		}
		log.Println("Successfully synced and applied new hotspot configuration.")
	} else {
		log.Println("Hotspot configuration is already in sync.")
	}

	return nil
}

func validateHotspotConfig(cfg *config.Config) error {
	if cfg.Network.WirelessInterface == "" { return fmt.Errorf("network.wireless_interface cannot be empty") }
	if cfg.Network.ApSSID == "" { return fmt.Errorf("network.ap_ssid cannot be empty") }
	if len(cfg.Network.ApPassword) < 8 { return fmt.Errorf("network.ap_password must be at least 8 characters long") }
	if !strings.Contains(cfg.Network.ApIpAddress, "/") { return fmt.Errorf("network.ap_ip_address must include a CIDR suffix (e.g., /24)") }
	return nil
}

func Start(cfg *config.Config, stopSignal <-chan bool) {
	timeout := time.NewTimer(time.Duration(cfg.BootManager.TimeoutSeconds) * time.Second)
	defer timeout.Stop()
	log.Printf("Boot manager started. Waiting %d seconds for user configuration...", cfg.BootManager.TimeoutSeconds)
	select {
	case <-stopSignal:
		log.Println("Boot manager received stop signal. Exiting.")
		return
	case <-timeout.C:
		log.Println("Boot manager timeout reached. Attempting to connect to last known WiFi network.")
		revertToLastGoodConfig()
	}
}

func revertToLastGoodConfig() {
	if _, err := os.Lstat(lastGoodSymlink); os.IsNotExist(err) { log.Println("No last-good WiFi configuration symlink found. Remaining in hotspot mode."); return }
	stopCmd := ExecCommand("sh", "-c", "systemctl stop hostapd dnsmasq pifigo")
	if output, err := stopCmd.CombinedOutput(); err != nil { log.Printf("ERROR: Boot manager failed to stop hotspot services: %v\nOutput: %s", err, string(output)) }
	copyCmd := ExecCommand("cp", lastGoodSymlink, activeClientConfig)
	if err := copyCmd.Run(); err != nil { log.Printf("ERROR: Boot manager failed to copy last-good config: %v", err); return }
	applyCmd := ExecCommand("netplan", "apply")
	if err := applyCmd.Run(); err != nil { log.Printf("ERROR: Boot manager failed to apply last-good WiFi config: %v", err) }
}

func ForceHotspotMode() error {
	if err := os.Remove(activeClientConfig); err != nil && !os.IsNotExist(err) { log.Printf("Warning: Could not remove active client config: %v", err) }
	applyCmd := ExecCommand("netplan", "apply")
	if output, err := applyCmd.CombinedOutput(); err != nil { log.Printf("ERROR: Failed to apply hotspot config: %v\nOutput: %s", err, string(output)); return err }
	restartCmd := ExecCommand("sh", "-c", "systemctl restart hostapd dnsmasq")
	if output, err := restartCmd.CombinedOutput(); err != nil { log.Printf("ERROR: Failed to restart hotspot services: %v\nOutput: %s", err, string(output)); return err }
	return nil
}
