package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Use var instead of const to allow them to be modified during testing.
var (
	savedNetworksDir   = "/etc/pifigo/saved_networks"
	lastGoodSymlink    = "/etc/pifigo/last-good-wifi.yaml"
	activeClientConfig = "/etc/netplan/99-pifigo-client.yaml"
)

// ShowStatus checks and prints the current network state of the device.
func ShowStatus() error {
	if _, err := os.Lstat(activeClientConfig); err == nil {
		// If the client config exists, we are likely in client mode.
		// We can try to read the SSID from the last-good symlink for more info.
		ssid, err := os.Readlink(lastGoodSymlink)
		if err != nil {
			fmt.Println("Status: Client Mode (SSID unknown)")
		} else {
			// Use filepath.Base to get just the filename from the symlink target.
			baseName := filepath.Base(ssid)
			// Use strings.TrimSuffix to remove the .yaml extension.
			fmt.Printf("Status: Client Mode (Last configured for: %s)\n", strings.TrimSuffix(baseName, ".yaml"))
		}
		// Perform a quick internet check.
		fmt.Println("Checking internet connectivity...")
		if checkInternet() {
			fmt.Println("Result: Internet connection is active.")
		} else {
			fmt.Println("Result: No internet connection detected.")
		}
	} else {
		// If the client config does not exist, we are in hotspot mode.
		fmt.Println("Status: Hotspot Mode")
	}
	return nil
}

// ShowLastGood reads the symlink and prints the SSID it points to.
func ShowLastGood() error {
	target, err := os.Readlink(lastGoodSymlink)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No last-known-good network is set.")
			return nil
		}
		return fmt.Errorf("could not read symlink: %w", err)
	}
	// Extract the filename without the extension to show just the SSID.
	ssid := strings.TrimSuffix(filepath.Base(target), ".yaml")
	fmt.Printf("Last Good Network: %s\n", ssid)
	return nil
}

// ListSavedNetworks reads the contents of the saved networks directory.
func ListSavedNetworks() error {
	files, err := ioutil.ReadDir(savedNetworksDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No networks have been saved yet.")
			return nil
		}
		return fmt.Errorf("could not read saved networks directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No networks have been saved yet.")
		return nil
	}

	fmt.Println("Saved Networks:")
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			ssid := strings.TrimSuffix(file.Name(), ".yaml")
			fmt.Printf("- %s\n", ssid)
		}
	}
	return nil
}

// SetLastGood updates the symlink to point to a different saved network profile.
func SetLastGood(ssid string) error {
	profilePath := filepath.Join(savedNetworksDir, ssid+".yaml")

	// Check if the target profile actually exists.
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return fmt.Errorf("network profile for SSID '%s' does not exist", ssid)
	}

	// Remove the old symlink if it exists.
	_ = os.Remove(lastGoodSymlink)

	// Create the new symlink.
	if err := os.Symlink(profilePath, lastGoodSymlink); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	fmt.Printf("Successfully set last-known-good network to: %s\n", ssid)
	return nil
}

// ForgetNetwork deletes a saved network profile.
func ForgetNetwork(ssid string) error {
	profilePath := filepath.Join(savedNetworksDir, ssid+".yaml")

	// Check if we are about to delete the currently linked "last-good" network.
	currentTarget, err := os.Readlink(lastGoodSymlink)
	if err == nil && currentTarget == profilePath {
		fmt.Println("Warning: This is the current last-known-good network. Removing symlink.")
		if err := os.Remove(lastGoodSymlink); err != nil {
			return fmt.Errorf("could not remove symlink: %w", err)
		}
	}

	// Delete the profile file.
	if err := os.Remove(profilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("network profile for SSID '%s' does not exist", ssid)
		}
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	fmt.Printf("Successfully forgot network: %s\n", ssid)
	return nil
}

// checkInternet performs a simple DNS lookup to verify connectivity.
func checkInternet() bool {
	// Use a short timeout to avoid long waits.
	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	_, err := resolver.LookupHost(context.Background(), "www.google.com")
	return err == nil
}
