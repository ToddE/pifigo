package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"pifigo/internal/config"
	"pifigo/internal/locale"
)

// execCommand is a package-level variable that holds the function for executing commands.
// This allows us to replace it with a mock during testing.
var execCommand = exec.Command

var (
	savedNetworksDir   = "/etc/pifigo/saved_networks"
	lastGoodSymlink    = "/etc/pifigo/last-good-wifi.yaml"
	activeClientConfig = "/etc/netplan/99-pifigo-client.yaml"
	netplanTemplate    = "/etc/pifigo/netplan.tpl"
)

// PageData is a composite struct that holds all data needed for API responses.
type PageData struct {
	Config  *config.Config
	Strings *locale.LanguageStrings
}

// serveDataAPI loads the full configuration and language strings and serves them as JSON.
func (s *Server) serveDataAPI(w http.ResponseWriter, r *http.Request) {
	langFilePath := filepath.Join(s.AppConfig.Paths.LocalesDir, s.AppConfig.Language+".yaml")
	langStrings, err := locale.LoadLanguageStrings(langFilePath)
	if err != nil {
		log.Printf("ERROR: Could not load language file '%s': %v", langFilePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	pageData := PageData{Config: s.AppConfig, Strings: langStrings}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pageData); err != nil {
		log.Printf("ERROR: Failed to encode JSON response: %v", err)
	}
}

// handleScanSSIDs scans for wireless networks and returns an HTML fragment for HTMX.
func (s *Server) handleScanSSIDs(w http.ResponseWriter, r *http.Request) {
	cmdStr := fmt.Sprintf("iw dev %s scan | grep 'SSID:' | sed 's/\\s*SSID: //'", s.AppConfig.Network.WirelessInterface)
	cmd := execCommand("sh", "-c", cmdStr)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("ERROR: Failed to scan for Wi-Fi networks: %v", err)
		fmt.Fprint(w, `<p class="text-red-500 p-4">Error: Could not scan for networks.</p>`)
		return
	}
	var ssids []string
	for _, ssid := range strings.Split(string(out), "\n") {
		if trimmed := strings.TrimSpace(ssid); trimmed != "" {
			ssids = append(ssids, trimmed)
		}
	}
	listTemplate := `{{range .}}<div class="ssid-item" onclick="selectSSID('{{.}}')">{{.}}</div>{{end}}`
	tmpl, _ := template.New("ssids").Parse(listTemplate)
	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, ssids)
}

// handleConnect receives credentials, saves the profile, and applies the connection.
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	ssid := r.FormValue("ssid")
	password := r.FormValue("password")
	if ssid == "" { http.Error(w, "SSID cannot be empty.", http.StatusBadRequest); return }
	log.Printf("Received request to connect to SSID: %s", ssid)
	tmpl, err := template.ParseFiles(netplanTemplate); if err != nil { log.Printf("ERROR: Failed to parse netplan template: %v", err); http.Error(w, "Internal Server Error", 500); return }
	data := struct{ SSID, Password, WirelessInterface string }{ssid, password, s.AppConfig.Network.WirelessInterface}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil { log.Printf("ERROR: Failed to execute netplan template: %v", err); http.Error(w, "Internal Server Error", 500); return }
	netplanContent := buf.Bytes()
	if err := os.MkdirAll(savedNetworksDir, 0755); err != nil { log.Printf("ERROR: Could not create saved_networks directory: %v", err); http.Error(w, "Internal Server Error", 500); return }
	profilePath := filepath.Join(savedNetworksDir, ssid+".yaml")
	if err := os.WriteFile(profilePath, netplanContent, 0644); err != nil { log.Printf("ERROR: Failed to write network profile: %v", err); http.Error(w, "Internal Server Error", 500); return }
	log.Printf("Saved new network profile to %s", profilePath)
	_ = os.Remove(lastGoodSymlink)
	if err := os.Symlink(profilePath, lastGoodSymlink); err != nil { log.Printf("ERROR: Failed to update symlink: %v", err) } else { log.Printf("Updated last-good symlink to point to %s", profilePath) }
	if err := os.WriteFile(activeClientConfig, netplanContent, 0644); err != nil { log.Printf("ERROR: Failed to write active netplan config: %v", err); http.Error(w, "Internal Server Error", 500); return }
	select { case s.StopSignal <- true: log.Println("Sent stop signal to boot manager."); default: log.Println("Could not send stop signal to boot manager (it may have already exited).") }
	
	cmd := execCommand("sh", "-c", "systemctl stop hostapd dnsmasq pifigo && netplan apply")
	if output, err := cmd.CombinedOutput(); err != nil { log.Printf("ERROR: Failed to apply netplan configuration: %v\nOutput: %s", err, string(output)); http.Error(w, "Failed to apply network settings.", 500); return }
	
	fmt.Fprint(w, `<p class="text-green-600 font-semibold">Success! The device is now attempting to connect to your Wi-Fi network.</p>`)
}

// handleListSavedNetworks reads the saved network profiles and returns an HTML fragment.
func (s *Server) handleListSavedNetworks(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(savedNetworksDir)
	langFilePath := filepath.Join(s.AppConfig.Paths.LocalesDir, s.AppConfig.Language+".yaml")
	langStrings, _ := locale.LoadLanguageStrings(langFilePath)
	if err != nil || len(files) == 0 {
		if langStrings != nil {
			fmt.Fprintf(w, `<p class="text-stone-500 p-4 text-center">%s</p>`, langStrings.NoSavedConnectionsMessage)
		}
		return
	}
	var savedSSIDs []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			savedSSIDs = append(savedSSIDs, strings.TrimSuffix(file.Name(), ".yaml"))
		}
	}
	reconnectText := "Reconnect"
	if langStrings != nil {
		reconnectText = langStrings.ReconnectButtonText
	}
	listTemplate := `{{range .SSIDs}}
        <div class="flex justify-between items-center p-2 rounded-lg hover:bg-stone-100">
            <span class="font-medium">{{.}}</span>
            <button hx-post="/reconnect" hx-vals='{"ssid": "{{.}}"}' hx-target="#response-div" hx-swap="innerHTML" hx-indicator="#spinner" class="px-3 py-1 text-sm bg-stone-200 text-stone-700 font-semibold rounded-md hover:bg-stone-300 transition-colors">
                {{$.ReconnectText}}
            </button>
        </div>
        {{end}}`
	tmpl, _ := template.New("saved").Parse(listTemplate)
	data := struct { SSIDs []string; ReconnectText string }{ savedSSIDs, reconnectText }
	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, data)
}

// handleReconnect takes a saved SSID, applies its config, and connects.
func (s *Server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	ssid := r.FormValue("ssid")
	if ssid == "" { http.Error(w, "SSID cannot be empty.", http.StatusBadRequest); return }
	log.Printf("Received reconnect request for SSID: %s", ssid)
	profilePath := filepath.Join(savedNetworksDir, ssid+".yaml")
	netplanContent, err := os.ReadFile(profilePath)
	if err != nil { log.Printf("ERROR: Could not read saved profile '%s': %v", profilePath, err); http.Error(w, "Could not find saved network profile.", http.StatusNotFound); return }
	_ = os.Remove(lastGoodSymlink)
	if err := os.Symlink(profilePath, lastGoodSymlink); err != nil { log.Printf("ERROR: Failed to update symlink: %v", err) }
	if err := os.WriteFile(activeClientConfig, netplanContent, 0644); err != nil { log.Printf("ERROR: Failed to write active netplan config: %v", err); http.Error(w, "Internal Server Error", 500); return }
	select { case s.StopSignal <- true: log.Println("Sent stop signal to boot manager."); default: log.Println("Could not send stop signal to boot manager (it may have already exited).") }
	
	cmd := execCommand("sh", "-c", "systemctl stop hostapd dnsmasq pifigo && netplan apply")
	if output, err := cmd.CombinedOutput(); err != nil { log.Printf("ERROR: Failed to apply netplan configuration: %v\nOutput: %s", err, string(output)); http.Error(w, "Failed to apply network settings.", 500); return }

	fmt.Fprintf(w, `<p class="text-green-600 font-semibold">Success! Attempting to reconnect to %s.</p>`, ssid)
}
