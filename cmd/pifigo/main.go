package main

import (
	"embed"
	"fmt"
	// "html/template"
	"log"
	"net/http"
	// "net/url"
	"os"
	// "strings"
	"time"

	"github.com/ToddE/pifigo/internal/common"
	"github.com/ToddE/pifigo/internal/config"
	"github.com/ToddE/pifigo/internal/netconfig"
	"github.com/ToddE/pifigo/internal/ui"
)

// --- Embed the template and ALL assets folder ---
//go:embed templates/index.html
var pifigoIndexHTML string

//go:embed assets
var embeddedAssetsFS embed.FS

// Data passed to HTML template
type PifigoTemplateData struct {
	ui.TemplateData
	// No additional fields here for pifigo
}

// init() function for logging setup
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// main() for pifigo binary
func main() {
	log.Println("Pifigo: PiFigo Wi-Fi Setup Service starting...")

	// --- Get runtime paths from environment variables ---
	appPaths := config.AppPaths{
		ConfigFilePath: os.Getenv("PIFIGO_CONFIG_PATH"),
		AssetsDirPath:  os.Getenv("PIFIGO_ASSETS_PATH"),
		LangDirPath:    os.Getenv("PIFIGO_LANG_PATH"),
		DeviceIdPath:   os.Getenv("PIFIGO_DEVICE_ID_PATH"),
	}

	// Check if essential paths are provided
	if appPaths.ConfigFilePath == "" || appPaths.AssetsDirPath == "" || 
	   appPaths.LangDirPath == "" || appPaths.DeviceIdPath == "" {
		log.Fatalf("Pifigo: Missing essential environment variables for paths. " +
			"Ensure PIFIGO_CONFIG_PATH, PIFIGO_ASSETS_PATH, PIFIGO_LANG_PATH, PIFIGO_DEVICE_ID_PATH are set.")
	}
	// --- END NEW ---

	// Load configuration and language strings
	cfg, lang := config.LoadConfig(appPaths) // Pass appPaths to LoadConfig

	// Ensure device ID exists (generated if not, loaded if it does)
	deviceID, err := config.EnsureDeviceId(appPaths.DeviceIdPath) // Pass deviceIdPath
	if err != nil {
		log.Fatalf("Pifigo: Failed to ensure device ID: %v", err)
	}
	claimCode := common.GenerateClaimCode()

	// --- Detect Wi-Fi Interface Name ---
	wifiInterfaceName, err := netconfig.DetectWifiInterface()
	if err != nil {
		log.Fatalf("Pifigo: Failed to detect Wi-Fi interface: %v", err)
	}
	log.Printf("Pifigo: Detected Wi-Fi interface: %s", wifiInterfaceName)
	// --- END NEW ---

	// --- Initialize the NetworkManager based on config.Runtime.NetworkManagerType ---
	networkManager, err := netconfig.GetNetworkManager(cfg.Runtime.NetworkManagerType, cfg.Network, wifiInterfaceName) // Pass wifiInterfaceName
	if err != nil {
		log.Fatalf("Pifigo: Failed to initialize network manager for type '%s': %v", cfg.Runtime.NetworkManagerType, err)
	}
	// --- END NEW ---

	// Configure AP mode network files (hostapd, dnsmasq, etc.)
	err = networkManager.ConfigureAPMode()
	if err != nil {
		log.Fatalf("Pifigo: Failed to configure AP mode: %v", err)
	}

	// --- CORRECTED appState struct definition and initialization ---
	appState := &struct {
		GlobalConfig    config.GlobalConfig
		LanguageStrings config.LanguageStrings
		DeviceID        string
		ClaimCode       string
		NetworkManager  netconfig.NetworkManager
		AppPaths        config.AppPaths // ADD THIS FIELD
	}{
		GlobalConfig:    cfg,
		LanguageStrings: lang,
		DeviceID:        deviceID,
		ClaimCode:       claimCode,
		NetworkManager:  networkManager,
		AppPaths:        appPaths, // ASSIGN IT HERE
	}

	// Setup HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlePifigoIndex(w, r, appState)
	})
	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		handleConnect(w, r, appState)
	})
	
	// Serve embedded assets from the 'assets' subdirectory within embeddedAssetsFS
	http.Handle("/assets/", http.FileServer(http.FS(embeddedAssetsFS))) 

	log.Printf("Pifigo: Web server listening on :80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

// handlePifigoIndex serves the captive portal page.
func handlePifigoIndex(w http.ResponseWriter, r *http.Request, appState *struct {
	GlobalConfig    config.GlobalConfig
	LanguageStrings config.LanguageStrings
	DeviceID        string
	ClaimCode       string
	NetworkManager  netconfig.NetworkManager
	AppPaths        config.AppPaths // Access via this field
}) {
	networks, err := appState.NetworkManager.ScanWifiNetworks()
	if err != nil {
		log.Printf("Pifigo: Error scanning Wi-Fi: %v", err)
	}

	tmpl, err := ui.GetTemplate(appState.GlobalConfig.Ui.CustomTemplate, pifigoIndexHTML, appState.AppPaths) // Pass appState.AppPaths
	if err != nil {
		http.Error(w, fmt.Sprintf("Pifigo: Error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	data := ui.TemplateData{
		Networks:        networks,
		UIConfig:        appState.GlobalConfig.Ui,
		Lang:            appState.LanguageStrings,
		DeviceID:        appState.DeviceID,
		ClaimCode:       appState.ClaimCode,
		Message:         appState.LanguageStrings.InitialMessage,
		LocalNodeSetupLink: fmt.Sprintf("http://%s.local/", appState.GlobalConfig.Network.DeviceHostname),
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Pifigo: Error executing template: %v", err)
		http.Error(w, "Pifigo: Error rendering page", http.StatusInternalServerError)
	}
}

// handleConnect processes Wi-Fi connection form submission.
func handleConnect(w http.ResponseWriter, r *http.Request, appState *struct {
	GlobalConfig   config.GlobalConfig
	LanguageStrings config.LanguageStrings
	DeviceID       string
	ClaimCode      string
	NetworkManager netconfig.NetworkManager
	AppPaths       config.AppPaths // Access via this field
}) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	ssid := r.FormValue("ssid_manual")
	if ssid == "" {
		ssid = r.FormValue("ssid_select")
	}
	password := r.FormValue("password")

	if ssid == "" {
		networks, _ := appState.NetworkManager.ScanWifiNetworks()
		tmpl, _ := ui.GetTemplate(appState.GlobalConfig.Ui.CustomTemplate, pifigoIndexHTML, appState.AppPaths) // Pass appState.AppPaths
		data := ui.TemplateData{
			Networks:     networks,
			UIConfig:     appState.GlobalConfig.Ui,
			Lang:         appState.LanguageStrings,
			ErrorMessage: appState.LanguageStrings.ErrorMessagePrefix + "SSID cannot be empty.",
			DeviceID:     appState.DeviceID,
			ClaimCode:    appState.ClaimCode,
		}
		tmpl.Execute(w, data)
		return
	}

	log.Printf("Pifigo: Attempting to connect to SSID: %s", ssid)

	// Write wpa_supplicant.conf for client mode using the interface
	err := appState.NetworkManager.WriteClientWifiConf(ssid, password, appState.GlobalConfig.Network.WifiCountry)
	if err != nil {
		log.Printf("Pifigo: Error writing client Wi-Fi config: %v", err)
		networks, _ := appState.NetworkManager.ScanWifiNetworks()
		tmpl, _ := ui.GetTemplate(appState.GlobalConfig.Ui.CustomTemplate, pifigoIndexHTML, appState.AppPaths) // Pass appPaths
		data := ui.TemplateData{
			Networks:     networks,
			UIConfig:     appState.GlobalConfig.Ui,
			Lang:         appState.LanguageStrings,
			ErrorMessage: appState.LanguageStrings.ErrorMessagePrefix + fmt.Sprintf("Failed to write Wi-Fi config: %v", err),
			DeviceID:     appState.DeviceID,
			ClaimCode:    appState.ClaimCode,
		}
		tmpl.Execute(w, data)
		return
	}

	// Disable AP mode services (these are universal, not manager-specific)
	log.Println("Pifigo: Stopping and disabling hostapd and dnsmasq...")
	common.ExecCommand("sudo", "systemctl", "stop", "hostapd")
	common.ExecCommand("sudo", "systemctl", "disable", "hostapd")
	common.ExecCommand("sudo", "systemctl", "stop", "dnsmasq")
	common.ExecCommand("sudo", "systemctl", "disable", "dnsmasq")

	// Disable this pifigo service itself so it doesn't run again on next boot
	log.Println("Pifigo: Disabling pifigo service...")
	common.ExecCommand("sudo", "systemctl", "disable", "pifigo")

	// Trigger reboot
	log.Println("Pifigo: Scheduling reboot in 1 minute...")
	_, err = common.ExecCommand("sudo", "shutdown", "-r", "+1")
	if err != nil {
		log.Printf("Pifigo: Error scheduling reboot: %v", err)
	}

	// Prepare success message for the user before reboot
	tmpl, _ := ui.GetTemplate(appState.GlobalConfig.Ui.CustomTemplate, pifigoIndexHTML, appState.AppPaths) // Pass appPaths
	localNodeSetupLink := fmt.Sprintf("http://%s.local/", appState.GlobalConfig.Network.DeviceHostname)
	successMsg := fmt.Sprintf(appState.LanguageStrings.SuccessMessageTemplate, ssid, localNodeSetupLink)
	
	data := ui.TemplateData{
		UIConfig:        appState.GlobalConfig.Ui,
		Lang:            appState.LanguageStrings,
		SuccessMessage:  successMsg,
		DeviceID:        appState.DeviceID,
		ClaimCode:       appState.ClaimCode,
		LocalNodeSetupLink: localNodeSetupLink,
	}
	tmpl.Execute(w, data)

	// Keep the server running briefly to send response, then exit gracefully
	go func() {
		time.Sleep(3 * time.Second) // Give browser time to receive response
		log.Println("Pifigo: Service exiting after successful configuration and reboot scheduling.")
		os.Exit(0) // Exit the process
	}()
}