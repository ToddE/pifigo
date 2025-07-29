package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

// Config defines the structure of the main configuration file (/etc/pifigo/config.yaml).
type Config struct {
	// BootManager holds settings for the timed hotspot on boot.
	BootManager struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
	} `yaml:"boot_manager"`

	// Watchdog holds settings for the internet connectivity checker.
	Watchdog struct {
		Enabled              bool   `yaml:"enabled"`
		CheckIntervalSeconds int    `yaml:"check_interval_seconds"`
		FailureThreshold     int    `yaml:"failure_threshold"`
		CheckURL             string `yaml:"check_url"`
	} `yaml:"watchdog"`

	// Paths specifies the file system locations for web assets and locales.
	Paths struct {
		WebRoot    string `yaml:"web_root"`
		LocalesDir string `yaml:"locales_dir"`
	} `yaml:"paths"`

	// UI holds settings for customizing the web interface's appearance.
	UI struct {
		PageTitle      string `yaml:"page_title"`
		HeadingText    string `yaml:"heading_text"`
		CustomImageURL string `yaml:"custom_image_url"`
	} `yaml:"ui"`

	// Network contains settings for the Wi-Fi hotspot and device hostname.
	Network struct {
		ApSSID            string   `yaml:"ap_ssid"`
		ApPassword        string   `yaml:"ap_password"`
		ApChannel         int      `yaml:"ap_channel"`
		ApIpAddress       string   `yaml:"ap_ip_address"`
		WifiCountry       string   `yaml:"wifi_country"`
		DeviceHostname    string   `yaml:"device_hostname"`
		WirelessInterface string   `yaml:"wireless_interface"`
		ConnectionMode    string   `yaml:"connection_mode"`
		StaticIP          string   `yaml:"static_ip"`
		Gateway           string   `yaml:"gateway"`
		DNSServers        []string `yaml:"dns_servers"`
	} `yaml:"network"`

	// Language sets the default language for the web interface.
	Language string `yaml:"language"`
}

// LoadConfig reads and parses the YAML configuration file from a given path.
func LoadConfig(path string) (*Config, error) {
	var cfg Config

	// Read the file from the provided path.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal the YAML data into the Config struct.
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
