package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"pifigo/internal/bootmanager"
	"pifigo/internal/cli"
	"pifigo/internal/config"
	"pifigo/internal/watchdog"
	"pifigo/server"
)

var version = "1.0.0"

func main() {
	// --- Define and Parse Command-Line Flags ---
	// (This section is unchanged)
	forceHotspot := flag.Bool("force-hotspot", false, "Force the device into hotspot mode and exit.")
	showVersion := flag.Bool("version", false, "Show the application version and exit.")
	showStatus := flag.Bool("status", false, "Show the current network status of the device and exit.")
	verbose := flag.Bool("v", false, "Enable verbose logging for server startup.")
	showLastGood := flag.Bool("last-good", false, "Show the SSID of the last-known-good network connection.")
	listSaved := flag.Bool("list-saved", false, "List all saved network profiles.")
	setGood := flag.String("set-good", "", "Set the last-known-good network to the specified SSID.")
	forgetNetwork := flag.String("forget", "", "Forget (delete) a saved network profile by SSID.")
	flag.Parse()

	// --- Dispatch Logic for Flags ---
	// (This section is unchanged)
	if *showVersion { fmt.Printf("pifigo version %s\n", version); os.Exit(0) }
	if *forceHotspot {
		if err := bootmanager.ForceHotspotMode(); err != nil { log.Fatalf("Failed to force hotspot mode: %v", err) }
		log.Println("Successfully reverted to hotspot mode."); os.Exit(0)
	}
	if *showStatus { if err := cli.ShowStatus(); err != nil { log.Fatalf("Failed to get status: %v", err) }; os.Exit(0) }
	if *showLastGood { if err := cli.ShowLastGood(); err != nil { log.Fatalf("Failed to get last good network: %v", err) }; os.Exit(0) }
	if *listSaved { if err := cli.ListSavedNetworks(); err != nil { log.Fatalf("Failed to list saved networks: %v", err) }; os.Exit(0) }
	if *setGood != "" { if err := cli.SetLastGood(*setGood); err != nil { log.Fatalf("Failed to set last good network: %v", err) }; os.Exit(0) }
	if *forgetNetwork != "" { if err := cli.ForgetNetwork(*forgetNetwork); err != nil { log.Fatalf("Failed to forget network: %v", err) }; os.Exit(0) }

	// --- Default Action: Start the Server and Services ---
	
	if *verbose { log.Println("Verbose logging enabled.") }
	log.Println("No admin flags provided. Starting pifigo services...")
	
	appConfig, err := config.LoadConfig("/etc/pifigo/config.yaml")
	if err != nil {
		log.Fatalf("FATAL: Could not load configuration from /etc/pifigo/config.yaml: %v", err)
	}

	// --- NEW: Sync the hotspot configuration on every start ---
	if err := bootmanager.SyncHotspotConfig(appConfig); err != nil {
		log.Printf("WARNING: Could not sync hotspot configuration: %v", err)
		// This is not a fatal error; the service can continue with the old config.
	}

	// Start the boot manager in a background goroutine.
	stopSignal := make(chan bool, 1)
	go bootmanager.Start(appConfig, stopSignal)

	// Start the watchdog if it's enabled in the config.
	if appConfig.Watchdog.Enabled {
		go watchdog.Start(appConfig)
	} else {
		log.Println("Watchdog is disabled in the configuration.")
	}

	// Create and start the web server in the main thread.
	srv := server.NewServer(appConfig, stopSignal)
	srv.Start()
}
