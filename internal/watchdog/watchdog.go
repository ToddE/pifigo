package watchdog

import (
	"log"
	"net/http"
	"os"
	"time"

	"pifigo/internal/bootmanager"
	"pifigo/internal/config"
)

const (
	activeClientConfig = "/etc/netplan/99-pifigo-client.yaml"
)

// Start begins the watchdog process in a continuous loop.
// It only takes action if it's enabled in the config.
func Start(cfg *config.Config) {
	// Give the system a couple of minutes to settle after boot before starting checks.
	time.Sleep(2 * time.Minute)

	log.Println("Watchdog service started.")
	failureCount := 0

	for {
		// Wait for the configured interval before the next check.
		time.Sleep(time.Duration(cfg.Watchdog.CheckIntervalSeconds) * time.Second)

		// Before checking, verify that we are supposed to be in client mode.
		// If the client config file doesn't exist, it means we are correctly
		// in hotspot mode, so the watchdog should do nothing.
		if _, err := os.Stat(activeClientConfig); os.IsNotExist(err) {
			if failureCount > 0 {
				log.Println("Watchdog: Device is in hotspot mode. Resetting failure count.")
				failureCount = 0 // Reset counter if we're back in hotspot mode.
			}
			continue // Skip the check.
		}

		// We are in client mode, so perform the connectivity check.
		if !checkInternet(cfg.Watchdog.CheckURL) {
			failureCount++
			log.Printf("Watchdog: Connectivity check failed (%d/%d).", failureCount, cfg.Watchdog.FailureThreshold)
		} else {
			// If the connection is good, reset the counter.
			if failureCount > 0 {
				log.Println("Watchdog: Connectivity restored. Resetting failure count.")
			}
			failureCount = 0
		}

		// If the failure threshold has been reached, take action.
		if failureCount >= cfg.Watchdog.FailureThreshold {
			log.Printf("Watchdog: Failure threshold of %d reached. Forcing hotspot mode.", cfg.Watchdog.FailureThreshold)
			
			// Call the function directly to revert to hotspot mode.
			if err := bootmanager.ForceHotspotMode(); err != nil {
				log.Printf("Watchdog: ERROR - Failed to force hotspot mode: %v", err)
			}
			
			// Reset the counter after taking action to avoid continuous triggers.
			failureCount = 0
		}
	}
}

// checkInternet performs a simple HTTP HEAD request to verify connectivity.
func checkInternet(url string) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// A successful request is typically in the 2xx range.
	return resp.StatusCode >= 200 && resp.StatusCode <= 299
}
