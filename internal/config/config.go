package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2" // Corrected import
	uuid "github.com/google/uuid"
)

// --- REMOVED HARDCODED CONSTANTS FOR FILE PATHS ---
// These paths will now be passed into LoadConfig via AppPaths
const (
	HostapdConfPath    = "/etc/hostapd/hostapd.conf"
	DnsmasqConfPath    = "/etc/dnsmasq.conf"
	DhcpcdConfPath     = "/etc/dhcpcd.conf"
	SysctlConfPath     = "/etc/sysctl.conf"
	DefaultHostapdPath = "/etc/default/hostapd"
	WpaSupplicantPath  = "/etc/wpa_supplicant/wpa_supplicant.conf"
)

// --- AppPaths Struct to hold runtime-provided paths ---
type AppPaths struct {
	ConfigFilePath string
	AssetsDirPath  string
	LangDirPath    string
	DeviceIdPath   string
}

// UiConfig contains UI styling settings.
type UiConfig struct {
	PageTitle           string `toml:"page_title"`
	HeadingText         string `toml:"heading_text"`
	BodyFont            string `toml:"body_font"`
	BackgroundColor     string `toml:"background_color"`
	TextColor           string `toml:"text_color"`
	ContainerColor      string `toml:"container_color"`
	HeadingColor        string `toml:"heading_color"`
	CustomImageURL      string `toml:"custom_image_url"`
	CustomTemplate      string `toml:"custom_template"`
}

// LanguageStrings contains all translatable text for pifigo.
type LanguageStrings struct {
	PageTitle             string `toml:"page_title"`
	HeadingText           string `toml:"heading_text"`
	ConnectButtonText     string `toml:"connect_button_text"`
	RefreshButtonText     string `toml:"refresh_button_text"`
	AvailableNetworksLabel string `toml:"available_networks_label"`
	ManualSSIDLabel       string `toml:"manual_ssid_label"`
	PasswordLabel         string `toml:"password_label"`
	ManualSSIDPlaceholder string `toml:"manual_ssid_placeholder"`
	PasswordPlaceholder   string `toml:"password_placeholder"`
	NoNetworksMessage     string `toml:"no_networks_message"`
	InitialMessage        string `toml:"initial_message"`
	ConnectingMessage     string `toml:"connecting_message"`
	SuccessMessageTemplate string `toml:"success_message_template"`
	ErrorMessagePrefix    string `toml:"error_message_prefix"`
	DeviceIdLabel         string `toml:"device_id_label"`
	ClaimCodeLabel        string `toml:"claim_code_label"`
	PostConnectInstructions string `toml:"post_connect_instructions"`
	LocalNodeSetupLinkText string `toml:"local_node_setup_link_text"`
	CopiedToClipboardMessage string `toml:"copied_to_clipboard_message"`
	// No wallet-specific strings here, as pifigo doesn't know about wallets.
}

// NetworkConfig contains network-specific settings for pifigo's AP mode and client config.
type NetworkConfig struct {
	ApSsid        string `toml:"ap_ssid"`
	ApPassword    string `toml:"ap_password"`
	ApChannel     uint8  `toml:"ap_channel"`
	WifiCountry   string `toml:"wifi_country"`
	DeviceHostname string `toml:"device_hostname"` // Hostname for mDNS on user's network
}

// RuntimeConfig for dynamically detected settings
type RuntimeConfig struct {
    NetworkManagerType string `toml:"network_manager_type"` // e.g., "NetworkManager", "dhcpcd", "systemd-networkd"
}

// --- NEW: LanguageConfig struct for the [language] section ---
type LanguageConfig struct {
    DefaultLang string `toml:"default_lang"` // Default language code, e.g., "en"
}


// GlobalConfig is the top-level configuration struct for pifigo.
type GlobalConfig struct {
	Ui               UiConfig        `toml:"ui"`
	Network          NetworkConfig   `toml:"network"`
	LanguageSettings LanguageConfig  `toml:"language"` // Corrected field name and TOML tag
    Runtime          RuntimeConfig   `toml:"runtime_config"`
}

// --- Default Values (for when fields are missing in TOML) ---

func (c *UiConfig) SetDefaults() {
	if c.PageTitle == "" { c.PageTitle = "PiFigo Setup" }
	if c.HeadingText == "" { c.HeadingText = "Connect Your Device to WiFi" }
	if c.BodyFont == "" { c.BodyFont = "Arial" }
	if c.BackgroundColor == "" { c.BackgroundColor = "#f0f2f5" }
	if c.TextColor == "" { c.TextColor = "#333" }
	if c.ContainerColor == "" { c.ContainerColor = "#ffffff" }
	if c.HeadingColor == "" { c.HeadingColor = "#007bff" }
}

func (s *LanguageStrings) SetDefaults() {
	if s.PageTitle == "" { s.PageTitle = "PiFigo Setup" }
	if s.HeadingText == "" { s.HeadingText = "Connect Your Device to WiFi" }
	if s.ConnectButtonText == "" { s.ConnectButtonText = "Connect" }
	if s.RefreshButtonText == "" { s.RefreshButtonText = "Refresh Networks" }
	if s.AvailableNetworksLabel == "" { s.AvailableNetworksLabel = "Available Networks:" }
	if s.ManualSSIDLabel == "" { s.ManualSSIDLabel = "Or Enter SSID Manually (if not listed):" }
	if s.PasswordLabel == "" { s.PasswordLabel = "Password:" }
	if s.ManualSSIDPlaceholder == "" { s.ManualSSIDPlaceholder = "Enter network name" }
	if s.PasswordPlaceholder == "" { s.PasswordPlaceholder = "Enter password" }
	if s.NoNetworksMessage == "" { s.NoNetworksMessage = "No networks found. Try refreshing or enter manually." }
	if s.InitialMessage == "" { s.InitialMessage = "Please connect your device to your Wi-Fi network." }
	if s.ConnectingMessage == "" { s.ConnectingMessage = "Attempting to connect... Device will reboot." }
	if s.SuccessMessageTemplate == "" { s.SuccessMessageTemplate = "Success! Your Device is connecting to '%s'. It will reboot shortly. Please reconnect your device to your main Wi-Fi network. Then, navigate to %s to continue setup." }
	if s.ErrorMessagePrefix == "" { s.ErrorMessagePrefix = "Error: " }
	if s.DeviceIdLabel == "" { s.DeviceIdLabel = "Your Device ID:" }
	if s.ClaimCodeLabel == "" { s.ClaimCodeLabel = "Claim Code:" }
	if s.PostConnectInstructions == "" { s.PostConnectInstructions = "Once your device is online, you can use its Device ID and Claim Code for further setup." }
	if s.LocalNodeSetupLinkText == "" { s.LocalNodeSetupLinkText = "Click here to access your device's local page." }
	if s.CopiedToClipboardMessage == "" { s.CopiedToClipboardMessage = "Copied '%s' to clipboard!" }
}

func (n *NetworkConfig) SetDefaults() {
	if n.ApSsid == "" { n.ApSsid = "PiFigoSetup" }
	if n.ApPassword == "" { n.ApPassword = "pifigo_pass" }
	if n.ApChannel == 0 { n.ApChannel = 7 }
	if n.WifiCountry == "" { n.WifiCountry = "US" }
	if n.DeviceHostname == "" { n.DeviceHostname = "pifigo-device" }
}

func (r *RuntimeConfig) SetDefaults() {
    if r.NetworkManagerType == "" { r.NetworkManagerType = "unknown" }
}

func (l *LanguageConfig) SetDefaults() { // Defaults for LanguageConfig
    if l.DefaultLang == "" { l.DefaultLang = "en" }
}


// LoadConfig loads GlobalConfig from a specified config file and LanguageStrings.
// It requires an AppPaths struct to know where to find its own config/data files.
func LoadConfig(paths AppPaths) (GlobalConfig, LanguageStrings) {
	var cfg GlobalConfig
	var langStrings LanguageStrings

	// Initialize with defaults first
	cfg.Ui.SetDefaults()
	cfg.Network.SetDefaults()
    cfg.LanguageSettings.SetDefaults() // Use the new field name
    cfg.Runtime.SetDefaults() 

	// Load GlobalConfig from file
	configBytes, err := ioutil.ReadFile(paths.ConfigFilePath)
	if err != nil {
		log.Printf("Warning: Could not read config file at %s: %v. Using all defaults.", paths.ConfigFilePath, err)
	} else {
		err = toml.Unmarshal(configBytes, &cfg)
		if err != nil {
			log.Printf("Error: Could not parse config file at %s: %v. Using defaults.", paths.ConfigFilePath, err)
		} else {
			// Apply defaults for fields not present in TOML
			cfg.Ui.SetDefaults()
			cfg.Network.SetDefaults()
            cfg.LanguageSettings.SetDefaults() // Use the new field name
            cfg.Runtime.SetDefaults()
			log.Printf("Loaded config from %s.", paths.ConfigFilePath)
		}
	}

	// Adjust custom_image_url if present
	if cfg.Ui.CustomImageURL != "" {
		cfg.Ui.CustomImageURL = "/assets/" + cfg.Ui.CustomImageURL
	}

	// Load LanguageStrings from file (now using cfg.LanguageSettings.DefaultLang)
	langFilePath := filepath.Join(paths.LangDirPath, fmt.Sprintf("%s.toml", cfg.LanguageSettings.DefaultLang)) // Use cfg.LanguageSettings.DefaultLang
	langBytes, err := ioutil.ReadFile(langFilePath)
	if err != nil {
		log.Printf("Warning: Could not read language file at %s: %v. Using default English strings.", langFilePath, err)
		langStrings.SetDefaults()
	} else {
		err = toml.Unmarshal(langBytes, &langStrings)
		if err != nil {
			log.Printf("Error: Could not parse language file at %s: %v. Using default English strings.", langFilePath, err)
			langStrings.SetDefaults()
		} else {
			langStrings.SetDefaults()
			log.Printf("Loaded language strings from %s.", langFilePath)
		}
	}

	return cfg, langStrings
}

// EnsureDeviceId loads or generates a unique device ID for pifigo.
// It now takes the deviceIdPath from AppPaths.
func EnsureDeviceId(deviceIdPath string) (string, error) {
	if _, err := os.Stat(deviceIdPath); err == nil {
		content, err := ioutil.ReadFile(deviceIdPath)
		if err != nil {
			return "", fmt.Errorf("failed to read device ID from %s: %v", deviceIdPath, err)
		}
		id := strings.TrimSpace(string(content))
		if id != "" {
			log.Printf("Loaded existing device ID: %s", id)
			return id, nil
		}
	}

	// Generate new ID if not found or empty
	newID := uuid.New().String()
	log.Printf("Generating new device ID: %s", newID)

	err := ioutil.WriteFile(deviceIdPath, []byte(newID), 0640) // rw-r-----
	if err != nil {
		return "", fmt.Errorf("failed to write device ID to %s: %v", deviceIdPath, err)
	}

	return newID, nil
}