# **PiFigo \- Headless WiFi Configuration Portal**

This document provides a technical overview of the pifigo project, its architecture, functionality, and the process for building and deploying it to a target device like an Orange Pi Zero 3\.

## **1\. Core Functionality & Architecture**

pifigo is a self-contained Go application designed to solve the "headless setup" problem for Linux-based single-board computers. It runs as a systemd service and uses netplan to manage the device's network state.

### **Operational Modes**

The service operates in two primary states:

1. **Hotspot Mode (Default/Fallback):**  
   * **Trigger:** Activates on first boot, after a bootmanager timeout with no user action, or when triggered by the watchdog.  
   * **Action:** Starts an access point using hostapd with DHCP services provided by dnsmasq. The device is accessible via a static IP (e.g., 192.168.4.1) and mDNS (http://pifigo.local via avahi-daemon).  
   * **Purpose:** To serve the web configuration portal.  
2. **Client Mode (Primary Goal):**  
   * **Trigger:** A user successfully submits credentials through the web portal.  
   * **Action:** The service stops the hotspot, generates a new netplan configuration file, and applies it, connecting the device to the user's chosen Wi-Fi network.  
   * **Purpose:** Normal, connected operation.

## **2\. Use Cases & Edge Case Handling**

pifigo is designed to be resilient and handle common failure scenarios automatically.

* **Initial Setup:** A user unboxes a new device, powers it on, connects to the "PiFigoSetup" Wi-Fi, and uses the web UI to connect it to their local network.  
* **Boot Manager Timeout:** If a user reboots the device and takes no action within the configured timeout (e.g., 3 minutes), the bootmanager goroutine will automatically attempt to connect to the last successfully used network. If no network has ever been configured, it remains in hotspot mode indefinitely.  
* **Watchdog Recovery:** If the device is in Client Mode but loses internet connectivity for a sustained period (configurable), the watchdog goroutine will assume the network is permanently unavailable (e.g., the device was moved) and will automatically revert the device to Hotspot Mode so it can be reconfigured.  
* **Saved Network Profiles:** The system saves every successful connection as a named profile. The web UI allows a user to quickly reconnect to any previously used network without re-entering the password. The bootmanager uses a symbolic link to track the "last good" profile for its fallback logic.

## **3\. Command-Line Interface (CLI) for Administration**

The pifigo binary includes a set of command-line flags for troubleshooting and administration. These are intended to be used by an administrator connected to the device (e.g., via SSH over Ethernet).

| Flag | Description |
| :---- | :---- |
| \--status | Shows the current mode (Hotspot/Client) and checks internet connectivity. |
| \--list-saved | Lists the SSIDs of all saved network profiles. |
| \--last-good | Shows which saved network is the current default for the boot manager. |
| \--set-good \<SSID\> | Manually sets the default fallback network to a specific saved profile. |
| \--forget \<SSID\> | Deletes a saved network profile. |
| \--force-hotspot | Forces the device into hotspot mode. Used by the watchdog or an admin. |
| \--version | Prints the application version. |
| \-v, \--verbose | Enables verbose logging on startup. |
| \-h, \--help | Displays the help message with all available flags. |

## **4\. Building and Deploying to an Orange Pi Zero 3**

This section outlines the process for compiling the project and deploying it.

### **Step 1: Compile the Go Binary**

From the project's root directory on your `amd64` Ubuntu development machine, cross-compile the application for the Orange Pi Zero 3's 64-bit ARM architecture.  
- Ensure dependencies are up-to-date  
`go mod tidy`

- Cross-compile for arm64  
```Bash
GOOS=linux GOARCH=arm64 go build -o pifigo .
```

### **Step 2: Stage Files for Packaging**

Prepare the `packaging/` directory with all the necessary components [(See "Debian Packaging Details" for more information)](#5-debian-packaging-details). 

Make sure that these files have the proper permissions that they'll need when copied/installed onto the target system. For example, files that will be placed in `/usr/local/bin` should be owned by root with `755` permissions.


### **Step 3: Build the Debian Package**

From the project's root directory, run the `dpkg-deb` command. 

- Build the package, naming it with version and architecture. 
example (change the version name ideally)

`dpkg-deb \--build packaging pifigo_0.0.1_arm64.deb`

### **Step 4: Copy and Install on the Orange Pi**

1. Copy the built .deb file to the target device using scp.  
   -  Replace replace <user> with your orange pi username and \<orange-pi-ip\> with the device's IP address  
   `scp pifigo_0.0.1_arm64.deb <user>@\<orange-pi-ip\>:~/`

2. SSH into the Orange Pi and install the package. This is a two-step process to correctly handle dependencies.  
   - SSH into the device
      ```bash
      ssh user@<orange-pi-ip>
      ```


   - Install the package with dpkg. This will likely show dependency errors.  
      ```bash
      sudo dpkg -i pifigo_0.0.1_arm64.deb
      ```

   - Use apt to automatically fix the missing dependencies.  
      ```bash
      sudo apt-get install -f
      ```
3. Take a look around and check journalctl for anything that seems like an obvious issue before rebooting. 
   ```bash
   journalctl -u pifigo.service
   ```

4. Reboot the device to apply the new network configuration cleanly.  
   ```bash
   sudo reboot
   ```
5. Cross your fingers. . . so far this hasn't worked but it is getting close (I think).
   
## **5\. Debian Packaging Details**

The packaging/ directory contains all files and scripts needed to build the .deb package.

* **DEBIAN/control**: The package's "identity card." It defines the package name, version, and, most importantly, the Depends list (netplan.io, hostapd, dnsmasq, etc.) that apt uses to install required software.  
* **DEBIAN/preinst**: A "smart installer" script that runs *before* installation. It checks for a known-conflicting default Armbian netplan file and safely replaces it with a specific configuration for the Ethernet port to prevent locking the user out.  
* **DEBIAN/postinst**: Runs *after* installation. It handles enabling the systemd service and deciding whether to start it immediately (if offline) or on the next boot (if already online).  
* **DEBIAN/prerm & postrm**: Scripts that ensure the service is stopped cleanly before removal and that all generated files are cleaned up afterward.  
* **File Structure**: The etc/, usr/, and var/ directories within packaging/ are exact replicas of the final installation paths on the target system.

## **6\. Project Structure & Testing**

The Go codebase is organized into modular packages to separate concerns.

* **main.go**: The main entry point. Handles CLI flag parsing and dispatches to the correct function or starts the services.  
* **server/**: Contains all the web server and API handler logic.
* **internal/**: Contains all the core application logic, kept private to the project.  
  * **config/**: Logic for parsing config.yaml.  
  * **locale/**: Logic for parsing language files.  
  * **bootmanager/**: Logic for the timed hotspot on boot.  
  * **watchdog/**: Logic for the internet connectivity monitor.  
  * **cli/**: Implementations for all the administrative CLI commands.  


To run the built-in unit tests, execute the following command from the project root:  
```bash
go test ./...  
```
