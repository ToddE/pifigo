package locale

import (
	"os"
	"gopkg.in/yaml.v3"
)

// LanguageStrings holds all the UI text for a specific language.
// The YAML tags must match the keys in your locale files (e.g., en.yaml).
type LanguageStrings struct {
	PageTitle               string `yaml:"page_title"`
	HeadingText             string `yaml:"heading_text"`
	ConnectButtonText       string `yaml:"connect_button_text"`
	AvailableNetworksLabel  string `yaml:"available_networks_label"`
	ManualSsidLabel         string `yaml:"manual_ssid_label"`
	PasswordLabel           string `yaml:"password_label"`
	ManualSsidPlaceholder   string `yaml:"manual_ssid_placeholder"`
	PasswordPlaceholder     string `yaml:"password_placeholder"`
	InitialMessage          string `yaml:"initial_message"`
	DeviceIdLabel           string `yaml:"device_id_label"`
	ClaimCodeLabel          string `yaml:"claim_code_label"`
	PostConnectInstructions string `yaml:"post_connect_instructions"`

	// New fields for the Saved Connections feature
	SavedConnectionsLabel     string `yaml:"saved_connections_label"`
	ReconnectButtonText       string `yaml:"reconnect_button_text"`
	NoSavedConnectionsMessage string `yaml:"no_saved_connections_message"`
}

// LoadLanguageStrings loads the specified language file from a given path.
func LoadLanguageStrings(path string) (*LanguageStrings, error) {
	var strings LanguageStrings
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &strings)
	if err != nil {
		return nil, err
	}
	return &strings, nil
}
