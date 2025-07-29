package server

import (
	"log"
	"net/http"
	"pifigo/internal/config"
)

// Server holds all dependencies for the web server, including the channel
// to signal the boot manager.
type Server struct {
	AppConfig  *config.Config
	StopSignal chan<- bool // The channel is write-only from the server's perspective.
}

// NewServer creates and returns a new Server instance.
func NewServer(cfg *config.Config, stopSignal chan<- bool) *Server {
	return &Server{
		AppConfig:  cfg,
		StopSignal: stopSignal,
	}
}

// Start registers all routes and starts the web server.
func (s *Server) Start() {
	// Serve static files (index.html, etc.) from the configured web_root.
	fs := http.FileServer(http.Dir(s.AppConfig.Paths.WebRoot))
	http.Handle("/", fs)

	// Register API endpoints to their handler methods.
	http.HandleFunc("/api/data", s.serveDataAPI)
	http.HandleFunc("/api/ssids", s.handleScanSSIDs)
	http.HandleFunc("/connect", s.handleConnect)

	// --- NEW ROUTES FOR SAVED CONNECTIONS ---
	http.HandleFunc("/api/saved_networks", s.handleListSavedNetworks)
	http.HandleFunc("/reconnect", s.handleReconnect)

	// Start the server.
	log.Printf("Starting pifigo web server on http://0.0.0.0:80")
	log.Printf("Serving web assets from '%s'", s.AppConfig.Paths.WebRoot)
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatalf("FATAL: ListenAndServe failed: %v", err)
	}
}
