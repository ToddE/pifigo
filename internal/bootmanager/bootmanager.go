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
	HotspotConfigFile = "/etc/netplan/00-pifigo-hotspot-ip.yaml"
	HostapdConfigFile = "/etc/hostapd/hostapd.conf"
	DnsmasqConfigFile = "/etc/dnsmasq.d/99-pifigo-hotspot"
)

const (
	lastGoodSymlink    = "/etc/pifigo/last-good-wifi.yaml"
	activeClientConfig = "/etc/netplan/99-pifigo-client.yaml"
)

// SyncHotspotConfig now generates hostapd and dnsmasq configs directly.
func SyncHotspotConfig(cfg *config.Config) error {
	log.Println("Syncing hotspot configuration...")

	if err := validateHotspotConfig(cfg); err != nil {
		return fmt.Errorf("invalid hotspot configuration: %w", err)
	}

	// 1. Generate the hostapd.conf content
	hostapdTemplate := `interface={{.Network.WirelessInterface}}
driver=nl80211
ssid={{.Network.ApSSID}}
hw_mode=g
channel={{.Network.ApChannel}}
wpa=2
wpa_passphrase={{.Network.ApPassword}}
wpa_key_mgmt=WPA-PSK
rsn_pairwise=CCMP
country_code={{.Network.WifiCountry}}
ieee80211n=1
ieee80211d=1
`
	if err := generateAndWriteFile("hostapd", hostapdTemplate, HostapdConfigFile, cfg); err != nil {
		return err
	}

	// 2. Generate the dnsmasq config content
	dnsmasqTemplate := `interface={{.Network.WirelessInterface}}
listen-address={{ipWithoutCidr .Network.ApIpAddress}}
bind-interfaces
server=8.8.8.8
domain-needed
bogus-priv
dhcp-range={{ipWithoutCidr .Network.ApIpAddress}},{{ipWithoutCidr .Network.ApIpAddress}},255.255.255.0,12h
`
	// The template functions needed for dnsmasq config
	funcMap := template.FuncMap{
		"ipWithoutCidr": func(ipWithCidr string) string {
			parts := strings.Split(ipWithCidr, "/")
			if len(parts) > 0 {
				return parts[0]
			}
			return ipWithCidr
		},
	}
	if err := generateAndWriteFile("dnsmasq", dnsmasqTemplate, DnsmasqConfigFile, cfg, funcMap); err != nil {
		return err
	}

	// 3. Generate a minimal netplan config JUST for the static IP
	netplanTemplate := `network:
  version: 2
  renderer: networkd
  wifis:
    {{.Network.WirelessInterface}}:
      dhcp4: no
      addresses: [{{.Network.ApIpAddress}}]
`
	if err := generateAndWriteFile("netplan", netplanTemplate, HotspotConfigFile, cfg); err != nil {
		return err
	}

	log.Println("Successfully synced all hotspot configuration files.")
	return nil
}

// generateAndWriteFile is a helper to generate, check, and write config files.
func generateAndWriteFile(name, tmplStr, path string, cfg *config.Config, funcMap ...template.FuncMap) error {
	tmpl := template.New(name)
	if len(funcMap) > 0 {
		tmpl = tmpl.Funcs(funcMap[0])
	}
	parsedTmpl, err := tmpl.Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("failed to parse internal %s template: %w", name, err)
	}

	var buf bytes.Buffer
	if err := parsedTmpl.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("failed to execute %s template: %w", name, err)
	}
	expectedContent := buf.Bytes()

	currentContent, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read current %s config: %w", name, err)
	}

	if !bytes.Equal(currentContent, expectedContent) {
		log.Printf("%s configuration is out of sync. Updating...", name)
		if err := os.WriteFile(path, expectedContent, 0644); err != nil {
			return fmt.Errorf("failed to write updated %s config: %w", name, err)
		}
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
	// When forcing hotspot, we remove the client configs and restart services.
	// The pifigo service, on next start, will re-sync the correct hotspot configs.
	if err := os.Remove(activeClientConfig); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Could not remove active client config: %v", err)
	}
	
	// Re-apply netplan. With the client config gone, it will use the static IP config.
	applyCmd := ExecCommand("netplan", "apply")
	if output, err := applyCmd.CombinedOutput(); err != nil {
		log.Printf("ERROR: Failed to apply hotspot config: %v\nOutput: %s", err, string(output))
		return err
	}
	// Restart the services that depend on the static IP.
	restartCmd := ExecCommand("sh", "-c", "systemctl restart hostapd dnsmasq")
	if output, err := restartCmd.CombinedOutput(); err != nil {
		log.Printf("ERROR: Failed to restart hotspot services: %v\nOutput: %s", err, string(output))
		return err
	}
	return nil
}
