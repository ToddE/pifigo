package netconfig

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	// "os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ToddE/pifigo/internal/common"
	"github.com/ToddE/pifigo/internal/config"
)

// The primary interface for network operations, which branches based on the detected manager.
type NetworkManager interface {
	ConfigureAPMode() error
	WriteClientWifiConf(ssid, password, country string) error
	ScanWifiNetworks() ([]string, error)
}

// GetNetworkManager returns the appropriate NetworkManager implementation based on config.
// It now requires the detected Wi-Fi interface name to be passed.
func GetNetworkManager(managerType string, netConfig config.NetworkConfig, wifiInterfaceName string) (NetworkManager, error) {
	if wifiInterfaceName == "" {
		return nil, fmt.Errorf("Wi-Fi interface name cannot be empty for network manager initialization")
	}
	switch managerType {
	case "NetworkManager":
		log.Println("NetConfig: Initializing NetworkManager implementation.")
		return &NetworkManagerImpl{netConfig: netConfig, wifiInterfaceName: wifiInterfaceName}, nil
	case "dhcpcd":
		log.Println("NetConfig: Initializing dhcpcd implementation.")
		return &DhcpcdImpl{netConfig: netConfig, wifiInterfaceName: wifiInterfaceName}, nil
	case "systemd-networkd":
		log.Println("NetConfig: Initializing systemd-networkd implementation.")
		return &SystemdNetworkdImpl{netConfig: netConfig, wifiInterfaceName: wifiInterfaceName}, nil
	default:
		return nil, fmt.Errorf("unsupported network manager type: %s", managerType)
	}
}

// --- Common AP Config File Management (shared by implementations) ---

// writeHostapdDnsmasqFiles writes the standard config files for hostapd and dnsmasq.
// This function now takes the detected wifiInterfaceName as an argument.
func writeHostapdDnsmasqFiles(netConfig config.NetworkConfig, wifiInterfaceName string) error {
	hostapdConfPath := config.HostapdConfPath
	dnsmasqConfPath := config.DnsmasqConfPath
	dhcpcdConfPath := config.DhcpcdConfPath
	sysctlConfPath := config.SysctlConfPath
	defaultHostapdPath := config.DefaultHostapdPath

	// dhcpcd.conf (Overwritten for static AP IP)
	dhcpcdContent := fmt.Sprintf(
		`interface %s
    static ip_address=192.168.4.1/24
    nohook wpa_supplicant
`, wifiInterfaceName) // Use detected interface name
	if err := ioutil.WriteFile(dhcpcdConfPath, []byte(dhcpcdContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %v", dhcpcdConfPath, err)
	}
	log.Printf("NetConfig: Wrote AP config to %s.", dhcpcdConfPath)

	// hostapd.conf
	hostapdContent := fmt.Sprintf(
		`country_code=%s
interface=%s
hw_mode=g
channel=%d
ssid=%s
wpa=2
wpa_passphrase=%s
wpa_key_mgmt=WPA-PSK
rsn_pairwise=CCMP
auth_algs=1
macaddr_acl=0
ignore_broadcast_ssid=0
`,
		netConfig.WifiCountry, wifiInterfaceName, // Use detected interface name
		netConfig.ApChannel, netConfig.ApSsid, netConfig.ApPassword,
	)
	if err := ioutil.WriteFile(hostapdConfPath, []byte(hostapdContent), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %v", hostapdConfPath, err)
	}
	log.Printf("NetConfig: Wrote %s.", hostapdConfPath)

	// Update DAEMON_CONF in /etc/default/hostapd
	currentDefaultHostapd, _ := ioutil.ReadFile(defaultHostapdPath)
	if !strings.Contains(string(currentDefaultHostapd), fmt.Sprintf("DAEMON_CONF=\"%s\"", hostapdConfPath)) {
		newDefaultHostapd := strings.ReplaceAll(string(currentDefaultHostapd), "#DAEMON_CONF=\"\"", fmt.Sprintf("DAEMON_CONF=\"%s\"", hostapdConfPath))
		newDefaultHostapd = strings.ReplaceAll(newDefaultHostapd, "DAEMON_CONF=\"\"", fmt.Sprintf("DAEMON_CONF=\"%s\"", hostapdConfPath))
		if err := ioutil.WriteFile(defaultHostapdPath, []byte(newDefaultHostapd), 0644); err != nil {
			return fmt.Errorf("failed to update %s: %v", defaultHostapdPath, err)
		}
		log.Printf("NetConfig: Updated %s with DAEMON_CONF path.", defaultHostapdPath)
	}

	// dnsmasq.conf
	dnsmasqContent := fmt.Sprintf(
		`interface=%s
dhcp-range=192.168.4.2,192.168.4.20,255.255.255.0,24h
domain=%s.local
address=/%s.local/192.168.4.1
address=/#/192.168.4.1
`,
		wifiInterfaceName, // Use detected interface name
		netConfig.DeviceHostname, netConfig.DeviceHostname,
	)
	if err := ioutil.WriteFile(dnsmasqConfPath, []byte(dnsmasqContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %v", dnsmasqConfPath, err)
	}
	log.Printf("NetConfig: Wrote %s.", dnsmasqConfPath)

	// sysctl.conf (IP Forwarding)
	currentSysctl, _ := ioutil.ReadFile(sysctlConfPath)
	if !strings.Contains(string(currentSysctl), "net.ipv4.ip_forward=1") {
		newSysctl := strings.ReplaceAll(string(currentSysctl), "#net.ipv4.ip_forward=1", "net.ipv4.ip_forward=1")
		if err := ioutil.WriteFile(sysctlConfPath, []byte(newSysctl), 0644); err != nil {
			return fmt.Errorf("failed to update %s: %v", sysctlConfPath, err)
		}
		if _, err := common.ExecCommand("sudo", "sysctl", "-p"); err != nil {
			return fmt.Errorf("failed to apply sysctl changes: %v", err)
		}
		log.Printf("NetConfig: Enabled IP forwarding in %s.", sysctlConfPath)
	}

	// Clean up wpa_supplicant.conf as it's not needed for AP mode and conflicts
	if err := ioutil.WriteFile(config.WpaSupplicantPath, []byte(""), 0600); err != nil {
		log.Printf("NetConfig: Warning: Failed to clear %s: %v", config.WpaSupplicantPath, err)
	}
	return nil
}

// stopApServices stops hostapd and dnsmasq. It also cleans up the config files created for AP mode.
func stopApServices() error {
	log.Println("NetConfig: Stopping and disabling hostapd and dnsmasq...")
	_, err := common.ExecCommand("sudo", "systemctl", "stop", "hostapd")
	if err != nil { log.Printf("NetConfig: Warning: Failed to stop hostapd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "disable", "hostapd")
	if err != nil { log.Printf("NetConfig: Warning: Failed to disable hostapd: %v", err) }
	
	_, err = common.ExecCommand("sudo", "systemctl", "stop", "dnsmasq")
	if err != nil { log.Printf("NetConfig: Warning: Failed to stop dnsmasq: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "disable", "dnsmasq")
	if err != nil { log.Printf("NetConfig: Warning: Failed to disable dnsmasq: %v", err) }

	// Clean up config files created for AP mode
	if err := ioutil.WriteFile(config.HostapdConfPath, []byte(""), 0600); err != nil { log.Printf("NetConfig: Warning: Failed to clear %s: %v", config.HostapdConfPath, err) }
	if err := ioutil.WriteFile(config.DnsmasqConfPath, []byte(""), 0644); err != nil { log.Printf("NetConfig: Warning: Failed to clear %s: %v", config.DnsmasqConfPath, err) }
	if err := ioutil.WriteFile(config.DhcpcdConfPath, []byte(""), 0644); err != nil { log.Printf("NetConfig: Warning: Failed to clear %s: %v", config.DhcpcdConfPath, err) }
	return nil
}

// --- NEW: Wi-Fi Interface Detection ---
// DetectWifiInterface tries to find the primary Wi-Fi interface name (e.g., wlan0, wlp88s0).
func DetectWifiInterface() (string, error) { // Correctly exported
	log.Println("NetConfig: Detecting primary Wi-Fi interface...")

	// Try nmcli first (most reliable on NetworkManager systems)
	output, err := common.ExecCommand("sudo", "nmcli", "--terse", "--fields", "DEVICE,TYPE", "device", "status")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 && parts[1] == "wifi" {
				log.Printf("NetConfig: Detected Wi-Fi interface via nmcli: %s", parts[0])
				return parts[0], nil
			}
		}
	} else {
		log.Printf("NetConfig: Warning: nmcli device status failed, trying other methods: %v", err)
	}

	// Fallback to /sys/class/net/wireless (more generic Linux approach)
	files, err := ioutil.ReadDir("/sys/class/net/")
	if err == nil {
		for _, file := range files {
			interfaceName := file.Name()
			if strings.HasPrefix(interfaceName, "wl") || strings.HasPrefix(interfaceName, "wlan") {
				// Check if it's a real wireless interface (not virtual like wlan0mon)
				if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s/wireless", interfaceName)); err == nil {
					log.Printf("NetConfig: Detected Wi-Fi interface via /sys/class/net/wireless: %s", interfaceName)
					return interfaceName, nil
				}
			}
		}
	} else {
		log.Printf("NetConfig: Warning: Could not read /sys/class/net/: %v", err)
	}

	// Final fallback: try iwconfig (if installed, older method)
	output, err = common.ExecCommand("sudo", "iwconfig")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Wireless") && !strings.Contains(line, "no wireless extensions") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					interfaceName := fields[0]
					log.Printf("NetConfig: Detected Wi-Fi interface via iwconfig: %s", interfaceName)
					return interfaceName, nil
				}
			}
		}
	} else {
		log.Printf("NetConfig: Warning: iwconfig failed, this might not be installed: %v", err)
	}

	return "", fmt.Errorf("no Wi-Fi interface found")
}

// --- Shared Scan Function for nmcli ---
func scanWifiWithNmcli(wifiInterfaceName string) ([]string, error) { // CORRECTED: Moved definition earlier
	log.Printf("NetConfig: Scanning for Wi-Fi networks via nmcli on %s...", wifiInterfaceName)
	
	_, err := common.ExecCommand("sudo", "nmcli", "radio", "wifi", "on")
	if err != nil { log.Printf("NetConfig: Warning: Failed to ensure nmcli Wi-Fi radio is on: %v", err) }

	_, err = common.ExecCommand("sudo", "nmcli", "device", "wifi", "rescan", "ifname", wifiInterfaceName) // Rescan specific interface
	if err != nil { log.Printf("NetConfig: Warning: Failed to trigger nmcli wifi rescan: %v", err) }
	time.Sleep(5 * time.Second) // Give time for scan to complete

	output, err := common.ExecCommand("sudo", "nmcli", "--fields", "SSID,BSSID,MODE", "device", "wifi", "list", "--rescan", "no", "ifname", wifiInterfaceName) // List specific interface
	if err != nil { return nil, fmt.Errorf("failed to list wifi via nmcli -g: %v", err) }
	
	networks := []string{}
	lineRegex := regexp.MustCompile(`^(.+?):([0-9A-Fa-f:]{17}):Ap$`) // Matches SSID:BSSID:Ap

	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		matches := lineRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			ssid := matches[1]
			ssid = strings.TrimPrefix(ssid, "\"")
			ssid = strings.TrimSuffix(ssid, "\"")
			networks = append(networks, ssid)
		}
	}
	
	sort.Strings(networks)
	uniqueNetworks := make([]string, 0, len(networks))
	seen := make(map[string]bool)
	for _, net := range networks {
		if _, ok := seen[net]; !ok {
			uniqueNetworks = append(uniqueNetworks, net)
			seen[net] = true
		}
	}
	log.Printf("NetConfig: Found %d unique Wi-Fi networks via nmcli on %s.", len(uniqueNetworks), wifiInterfaceName)
	return uniqueNetworks, nil
}


// --- NetworkManager Implementation ---
type NetworkManagerImpl struct {
	netConfig config.NetworkConfig
	wifiInterfaceName string // Store the detected interface name
}

// ConfigureAPMode for NetworkManager-managed systems.
func (nm *NetworkManagerImpl) ConfigureAPMode() error {
	log.Println("NetConfig: [NetworkManagerImpl] Configuring AP mode.")

	// 1. Temporarily disable NetworkManager
	log.Println("NetConfig: [NetworkManagerImpl] Stopping and disabling NetworkManager for AP mode...")
	_, err := common.ExecCommand("sudo", "systemctl", "stop", "NetworkManager.service")
	if err != nil { log.Printf("NetConfig: [NetworkManagerImpl] Warning: Failed to stop NetworkManager: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "disable", "NetworkManager.service")
	if err != nil { log.Printf("NetConfig: [NetworkManagerImpl] Warning: Failed to disable NetworkManager: %v", err) }
	
	// 2. Ensure wlan0 is raw for hostapd
	log.Printf("NetConfig: [NetworkManagerImpl] Ensuring %s is down and unmanaged by NetworkManager...", nm.wifiInterfaceName)
	_, _ = common.ExecCommand("sudo", "nmcli", "device", "set", nm.wifiInterfaceName, "managed", "no") 
	_, _ = common.ExecCommand("sudo", "ip", "link", "set", nm.wifiInterfaceName, "down")
	_, err = common.ExecCommand("sudo", "ip", "link", "set", nm.wifiInterfaceName, "up")
	if err != nil { return fmt.Errorf("[NetworkManagerImpl] failed to bring %s up for AP mode: %v", nm.wifiInterfaceName, err) }

	// 3. Write and start hostapd/dnsmasq files (shared function)
	if err := writeHostapdDnsmasqFiles(nm.netConfig, nm.wifiInterfaceName); err != nil {
		return fmt.Errorf("[NetworkManagerImpl] failed to write hostapd/dnsmasq files: %v", err)
	}

	// 4. Start hostapd/dnsmasq services
	log.Println("NetConfig: [NetworkManagerImpl] Starting hostapd and dnsmasq...")
	_, err = common.ExecCommand("sudo", "systemctl", "start", "hostapd")
	if err != nil { return fmt.Errorf("[NetworkManagerImpl] failed to start hostapd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "start", "dnsmasq")
	if err != nil { return fmt.Errorf("[NetworkManagerImpl] failed to start dnsmasq: %v", err) }
	
	log.Println("NetConfig: [NetworkManagerImpl] AP mode configured and activated.")
	return nil
}

// WriteClientWifiConf for NetworkManager-managed systems.
func (nm *NetworkManagerImpl) WriteClientWifiConf(ssid, password, country string) error {
	log.Printf("NetConfig: [NetworkManagerImpl] Configuring client Wi-Fi via NetworkManager for SSID: %s", ssid)

	// 1. Ensure NetworkManager is enabled and started
	log.Println("NetConfig: [NetworkManagerImpl] Ensuring NetworkManager is enabled and started...")
	_, err := common.ExecCommand("sudo", "systemctl", "enable", "NetworkManager.service")
	if err != nil { return fmt.Errorf("[NetworkManagerImpl] failed to enable NetworkManager: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "start", "NetworkManager.service")
	if err != nil { return fmt.Errorf("[NetworkManagerImpl] failed to start NetworkManager: %v", err) }
	
	// 2. Ensure Wi-Fi interface is managed by NetworkManager and up
	log.Printf("NetConfig: [NetworkManagerImpl] Ensuring %s is managed by NetworkManager and up...", nm.wifiInterfaceName)
	_, err = common.ExecCommand("sudo", "nmcli", "device", "set", nm.wifiInterfaceName, "managed", "yes")
	if err != nil { return fmt.Errorf("[NetworkManagerImpl] failed to set %s as managed by nmcli: %v", nm.wifiInterfaceName, err) }
	_, err = common.ExecCommand("sudo", "ip", "link", "set", nm.wifiInterfaceName, "up")
	if err != nil { log.Printf("NetConfig: [NetworkManagerImpl] Warning: Failed to bring %s up: %v", nm.wifiInterfaceName, err) }

	// 3. Delete any existing connection for this SSID (to prevent conflicts)
	conName := fmt.Sprintf("pifigo-%s", ssid) 
	log.Printf("NetConfig: [NetworkManagerImpl] Attempting to delete existing connection '%s' if present...", conName)
	// nmcli connection delete returns 1 if connection does not exist, so ignore error.
	_, _ = common.ExecCommand("sudo", "nmcli", "connection", "delete", conName) 

	// 4. Add the new Wi-Fi connection
	log.Printf("NetConfig: [NetworkManagerImpl] Adding new Wi-Fi connection '%s'...", conName)
	_, err = common.ExecCommand("sudo", "nmcli", "connection", "add", 
		"type", "wifi", 
		"con-name", conName, 
		"ifname", nm.wifiInterfaceName, 
		"ssid", ssid, 
		"wifi-sec.key-mgmt", "wpa-psk", 
		"wifi-sec.psk", password,
		"wifi.mode", "infra", 
		"connection.autoconnect", "yes") 
	if err != nil {
		return fmt.Errorf("[NetworkManagerImpl] failed to add NetworkManager Wi-Fi connection: %v", err)
	}

	// 5. Bring up the connection (activate it)
	log.Printf("NetConfig: [NetworkManagerImpl] Activating Wi-Fi connection '%s'...", conName)
	_, err = common.ExecCommand("sudo", "nmcli", "connection", "up", conName)
	if err != nil {
		return fmt.Errorf("[NetworkManagerImpl] failed to activate NetworkManager Wi-Fi connection: %v", err)
	}

	// 6. Clean up AP mode config files (universal function)
	if err := stopApServices(); err != nil {
		log.Printf("NetConfig: [NetworkManagerImpl] Warning: Failed to clean up AP services: %v", err)
	}

	log.Println("NetConfig: [NetworkManagerImpl] Client Wi-Fi configured and activated.")
	return nil
}

// ScanWifiNetworks for NetworkManager-managed systems.
func (nm *NetworkManagerImpl) ScanWifiNetworks() ([]string, error) {
	// Call the shared nmcli scan function
	return scanWifiWithNmcli(nm.wifiInterfaceName)
}

// --- dhcpcd Implementation ---
type DhcpcdImpl struct {
	netConfig config.NetworkConfig
	wifiInterfaceName string // Store the detected interface name
}

// ConfigureAPMode for dhcpcd-managed systems.
func (d *DhcpcdImpl) ConfigureAPMode() error {
	log.Println("NetConfig: [DhcpcdImpl] Configuring AP mode.")

	// 1. Temporarily disable dhcpcd
	log.Println("NetConfig: [DhcpcdImpl] Stopping and disabling dhcpcd for AP mode...")
	_, err := common.ExecCommand("sudo", "systemctl", "stop", "dhcpcd.service")
	if err != nil { log.Printf("NetConfig: [DhcpcdImpl] Warning: Failed to stop dhcpcd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "disable", "dhcpcd.service")
	if err != nil { log.Printf("NetConfig: [DhcpcdImpl] Warning: Failed to disable dhcpcd: %v", err) }
	
	// 2. Ensure wlan0 is raw for hostapd
	log.Printf("NetConfig: [DhcpcdImpl] Ensuring %s is down and up...", d.wifiInterfaceName)
	_, _ = common.ExecCommand("sudo", "ip", "link", "set", d.wifiInterfaceName, "down")
	_, err = common.ExecCommand("sudo", "ip", "link", "set", d.wifiInterfaceName, "up")
	if err != nil { return fmt.Errorf("[DhcpcdImpl] failed to bring %s up for AP mode: %v", d.wifiInterfaceName, err) }

	// 3. Write and start hostapd/dnsmasq files (shared function)
	if err := writeHostapdDnsmasqFiles(d.netConfig, d.wifiInterfaceName); err != nil {
		return fmt.Errorf("[DhcpcdImpl] failed to write hostapd/dnsmasq files: %v", err)
	}

	// 4. Start hostapd/dnsmasq services
	log.Println("NetConfig: [DhcpcdImpl] Starting hostapd and dnsmasq...")
	_, err = common.ExecCommand("sudo", "systemctl", "start", "hostapd")
	if err != nil { return fmt.Errorf("[DhcpcdImpl] failed to start hostapd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "start", "dnsmasq")
	if err != nil { return fmt.Errorf("[DhcpcdImpl] failed to start dnsmasq: %v", err) }
	
	log.Println("NetConfig: [DhcpcdImpl] AP mode configured and activated.")
	return nil
}

// WriteClientWifiConf for dhcpcd-managed systems.
func (d *DhcpcdImpl) WriteClientWifiConf(ssid, password, country string) error {
	log.Printf("NetConfig: [DhcpcdImpl] Configuring client Wi-Fi via dhcpcd/wpa_supplicant for SSID: %s", ssid)

	// 1. Ensure dhcpcd is enabled and started
	log.Println("NetConfig: [DhcpcdImpl] Ensuring dhcpcd is enabled and started...")
	_, err := common.ExecCommand("sudo", "systemctl", "enable", "dhcpcd.service")
	if err != nil { return fmt.Errorf("[DhcpcdImpl] failed to enable dhcpcd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "start", "dhcpcd.service")
	if err != nil { return fmt.Errorf("[DhcpcdImpl] failed to start dhcpcd: %v", err) }

	// 2. Write the wpa_supplicant.conf
	wpaSupplicantPath := config.WpaSupplicantPath
	content := fmt.Sprintf(
		`ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1
country=%s

network={
    ssid="%s"
    psk="%s"
}`, country, ssid, password)

	if err := ioutil.WriteFile(wpaSupplicantPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("[DhcpcdImpl] failed to write %s: %v", wpaSupplicantPath, err)
	}
	log.Printf("NetConfig: [DhcpcdImpl] Wrote client Wi-Fi config to %s.", wpaSupplicantPath)

	// 3. Request new DHCP lease
	log.Println("NetConfig: [DhcpcdImpl] Restarting dhcpcd service to acquire new lease...")
	_, err = common.ExecCommand("sudo", "systemctl", "restart", "dhcpcd.service")
	if err != nil { return fmt.Errorf("[DhcpcdImpl] failed to restart dhcpcd.service: %v", err) }

	// 4. Clean up AP mode config files (universal function)
	if err := stopApServices(); err != nil {
		log.Printf("NetConfig: [DhcpcdImpl] Warning: Failed to clean up AP services: %v", err)
	}

	log.Println("NetConfig: [DhcpcdImpl] Client Wi-Fi configured and activated.")
	return nil
}

// ScanWifiNetworks for dhcpcd-managed systems (uses iwlist).
func (d *DhcpcdImpl) ScanWifiNetworks() ([]string, error) {
	log.Println("NetConfig: [DhcpcdImpl] Scanning for Wi-Fi networks via iwlist...")
	
	_, err := common.ExecCommand("sudo", "ifconfig", d.wifiInterfaceName, "up")
	if err != nil { log.Printf("NetConfig: [DhcpcdImpl] Warning: Failed to ensure %s is up for scan: %v", d.wifiInterfaceName, err) }

	output, err := common.ExecCommand("sudo", "iwlist", d.wifiInterfaceName, "scan")
	if err != nil { return nil, fmt.Errorf("[DhcpcdImpl] failed to execute iwlist: %v", err) }

	var networks []string
	lines := strings.Split(output, "\n")
	re := regexp.MustCompile(`ESSID:"([^"]+)"`) // Correct regex for iwlist output

	for _, line := range lines {
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			networks = append(networks, matches[1])
		}
	}
	sort.Strings(networks)
	uniqueNetworks := make([]string, 0, len(networks))
	seen := make(map[string]bool)
	for _, net := range networks {
		if _, ok := seen[net]; !ok {
			uniqueNetworks = append(uniqueNetworks, net)
			seen[net] = true
		}
	}
	log.Printf("NetConfig: [DhcpcdImpl] Found %d unique Wi-Fi networks.", len(uniqueNetworks))
	return uniqueNetworks, nil
}


// --- systemd-networkd Implementation (Placeholder) ---
type SystemdNetworkdImpl struct {
	netConfig config.NetworkConfig
	wifiInterfaceName string // Store the detected interface name
}

func (s *SystemdNetworkdImpl) ConfigureAPMode() error {
	log.Println("NetConfig: [SystemdNetworkdImpl] Configuring AP mode. (Implementation needed)")
	// 1. Temporarily disable systemd-networkd
	log.Println("NetConfig: [SystemdNetworkdImpl] Stopping and disabling systemd-networkd for AP mode...")
	_, err := common.ExecCommand("sudo", "systemctl", "stop", "systemd-networkd.service")
	if err != nil { log.Printf("NetConfig: [SystemdNetworkdImpl] Warning: Failed to stop systemd-networkd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "disable", "systemd-networkd.service")
	if err != nil { log.Printf("NetConfig: [SystemdNetworkdImpl] Warning: Failed to disable systemd-networkd: %v", err) }

	// 2. Ensure wlan0 is raw for hostapd
	log.Printf("NetConfig: [SystemdNetworkdImpl] Ensuring %s is down and up...", s.wifiInterfaceName)
	_, _ = common.ExecCommand("sudo", "ip", "link", "set", s.wifiInterfaceName, "down")
	_, err = common.ExecCommand("sudo", "ip", "link", "set", s.wifiInterfaceName, "up")
	if err != nil { return fmt.Errorf("[SystemdNetworkdImpl] failed to bring %s up for AP mode: %v", s.wifiInterfaceName, err) }
	
	// 3. Write and start hostapd/dnsmasq files (shared function)
	if err := writeHostapdDnsmasqFiles(s.netConfig, s.wifiInterfaceName); err != nil {
		return fmt.Errorf("[SystemdNetworkdImpl] failed to write hostapd/dnsmasq files: %v", err)
	}

	// 4. Start hostapd/dnsmasq services
	log.Println("NetConfig: [SystemdNetworkdImpl] Starting hostapd and dnsmasq...")
	_, err = common.ExecCommand("sudo", "systemctl", "start", "hostapd")
	if err != nil { return fmt.Errorf("[SystemdNetworkdImpl] failed to start hostapd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "start", "dnsmasq")
	if err != nil { return fmt.Errorf("[SystemdNetworkdImpl] failed to start dnsmasq: %v", err) }

	log.Println("NetConfig: [SystemdNetworkdImpl] AP mode configured and activated.")
	return nil
}

func (s *SystemdNetworkdImpl) WriteClientWifiConf(ssid, password, country string) error {
	log.Println("NetConfig: [SystemdNetworkdImpl] Configuring client Wi-Fi. (Implementation needed)")
	// 1. Ensure systemd-networkd is enabled and started
	log.Println("NetConfig: [SystemdNetworkdImpl] Ensuring systemd-networkd is enabled and started...")
	_, err := common.ExecCommand("sudo", "systemctl", "enable", "systemd-networkd.service")
	if err != nil { return fmt.Errorf("[SystemdNetworkdImpl] failed to enable systemd-networkd: %v", err) }
	_, err = common.ExecCommand("sudo", "systemctl", "start", "systemd-networkd.service")
	if err != nil { return fmt.Errorf("[SystemdNetworkdImpl] failed to start systemd-networkd: %v", err) }

	// 2. Write the .network file for wlan0
	// This is a basic example. Real .network files can be complex.
	networkFileContent := fmt.Sprintf(`[Match]
Name=%s

[Network]
DHCP=yes

[Wifi]
SSID=%s
Password=%s
# Country needs to be set globally or in wpa_supplicant.conf linked by networkd.
# For simplicity, relying on global system country or a separate wpa_supplicant.conf
# that networkd is configured to use.
`, s.wifiInterfaceName, ssid, password) // Use detected interface name
    networkDirPath := "/etc/systemd/network/"
    if err := os.MkdirAll(networkDirPath, 0755); err != nil { // Ensure directory exists
        return fmt.Errorf("[SystemdNetworkdImpl] failed to create %s: %v", networkDirPath, err)
    }
    networkFilePath := filepath.Join(networkDirPath, fmt.Sprintf("%s.network", s.wifiInterfaceName)) // Name based on interface
    if err := ioutil.WriteFile(networkFilePath, []byte(networkFileContent), 0644); err != nil {
        return fmt.Errorf("[SystemdNetworkdImpl] failed to write %s: %v", networkFilePath, err)
    }
    log.Printf("NetConfig: [SystemdNetworkdImpl] Wrote client Wi-Fi config to %s.", networkFilePath)

    // 3. Reload networkd configuration and restart service
    _, err = common.ExecCommand("sudo", "systemctl", "reload", "systemd-networkd.service")
    if err != nil { log.Printf("NetConfig: [SystemdNetworkdImpl] Warning: Failed to reload systemd-networkd: %v", err) }
    _, err = common.ExecCommand("sudo", "systemctl", "restart", "systemd-networkd.service")
    if err != nil { return fmt.Errorf("[SystemdNetworkdImpl] failed to restart systemd-networkd.service: %v", err) }

	// 4. Clean up AP mode config files (universal function)
	if err := stopApServices(); err != nil {
		log.Printf("NetConfig: [SystemdNetworkdImpl] Warning: Failed to clean up AP services: %v", err)
	}

	log.Println("NetConfig: [SystemdNetworkdImpl] Client Wi-Fi configured and activated.")
	return nil
}

func (s *SystemdNetworkdImpl) ScanWifiNetworks() ([]string, error) {
	log.Println("NetConfig: [SystemdNetworkdImpl] Scanning for Wi-Fi networks. (Implementation needed)")
	
    // Declare slice and map here, outside the if/else blocks to ensure scope
    var networks []string
    var uniqueNetworks []string
    var seen map[string]bool

    // Ensure interface is up for scanning
    _, err := common.ExecCommand("sudo", "ip", "link", "set", s.wifiInterfaceName, "up")
    if err != nil { log.Printf("NetConfig: [SystemdNetworkdImpl] Warning: Failed to bring %s up: %v", s.wifiInterfaceName, err) }

    // Try nmcli first (if installed and working)
    nmcliOutput, err := common.ExecCommand("sudo", "nmcli", "-g", "SSID,BSSID,MODE", "dev", "wifi", "list")
    if err == nil {
        log.Println("NetConfig: [SystemdNetworkdImpl] Using nmcli for scan.")
        networks = make([]string, 0) // Assign, not redeclare
        lineRegex := regexp.MustCompile(`^(.+?):([0-9A-Fa-f:]{17}):Ap$`) 
        for _, line := range strings.Split(strings.TrimSpace(nmcliOutput), "\n") {
            matches := lineRegex.FindStringSubmatch(line)
            if len(matches) > 1 {
                ssid := matches[1]
                ssid = strings.TrimPrefix(ssid, "\"")
                ssid = strings.TrimSuffix(ssid, "\"")
                networks = append(networks, ssid)
            }
        }
        sort.Strings(networks)
        uniqueNetworks = make([]string, 0, len(networks)) // Assign
        seen = make(map[string]bool) // Assign
        for _, net := range networks {
            if _, ok := seen[net]; !ok {
                uniqueNetworks = append(uniqueNetworks, net)
                seen[net] = true
            }
        }
        log.Printf("NetConfig: [SystemdNetworkdImpl] Found %d unique Wi-Fi networks.", len(uniqueNetworks))
        return uniqueNetworks, nil
    }

    // Fallback to iwlist if nmcli fails
    log.Println("NetConfig: [SystemdNetworkdImpl] nmcli failed. Attempting iwlist for scan.")
    iwlistOutput, err := common.ExecCommand("sudo", "iwlist", s.wifiInterfaceName, "scan")
    if err == nil {
        log.Println("NetConfig: [SystemdNetworkdImpl] Using iwlist for scan.")
        networks = make([]string, 0) // Assign, not redeclare
        lines := strings.Split(iwlistOutput, "\n")
        re := regexp.MustCompile(`ESSID:"([^"]+)"`)
        for _, line := range lines {
            if matches := re.FindStringSubmatch(line); len(matches) > 1 {
                networks = append(networks, matches[1])
            }
        }
        sort.Strings(networks)
        uniqueNetworks = make([]string, 0, len(networks)) // Assign
        seen = make(map[string]bool) // Assign
        for _, net := range networks {
            if _, ok := seen[net]; !ok {
                uniqueNetworks = append(uniqueNetworks, net)
                seen[net] = true
            }
        }
        log.Printf("NetConfig: [SystemdNetworkdImpl] Found %d unique Wi-Fi networks.", len(uniqueNetworks))
        return uniqueNetworks, nil
    }
    
	return nil, fmt.Errorf("SystemdNetworkdImpl Wi-Fi scan failed: neither nmcli nor iwlist worked: %v", err)
}