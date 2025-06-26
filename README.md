## pifigo - Headless Wi-Fi Setup for Embedded Linux Devices

![pifigo Logo Placeholder](cmd/pifigo/assets/logo.png) **pifigo** is a lightweight, Go-powered application designed to simplify the initial Wi-Fi configuration of headless Linux-based embedded devices (like Raspberry Pi, Orange Pi, or other single-board computers) via a web browser. It sets up a temporary Access Point (AP) and a captive portal, allowing users to connect their device to their home Wi-Fi network easily, without needing a monitor, keyboard, or mouse.

# THIS IS A WORK IN PROGRESS

## Table of Contents
- [THIS IS A WORK IN PROGRESS](#this-is-a-work-in-progress)
  - [Table of Contents](#table-of-contents)
  - [1. Features](#1-features)
  - [2. Why pifigo?](#2-why-pifigo)
  - [3. Requirements](#3-requirements)
  - [4. TLDR QUICK START GUIDE](#4-tldr-quick-start-guide)
    - [After your device reboots (wait 30-60 seconds)](#after-your-device-reboots-wait-30-60-seconds)
  - [5. Configuring pifigo](#5-configuring-pifigo)
    - [Main Config (`config.toml`)](#main-config-configtoml)
    - [Localization (`lang/`)](#localization-lang)
    - [Custom Assets (`assets/`)](#custom-assets-assets)
  - [6. Management](#6-management)
    - [pifigo Service Control](#pifigo-service-control)
    - [Forcing AP Mode / Resetting Wi-Fi](#forcing-ap-mode--resetting-wi-fi)
    - [Updating pifigo](#updating-pifigo)
    - [Uninstalling pifigo](#uninstalling-pifigo)
  - [7. Troubleshooting](#7-troubleshooting)
  - [8. Contributing](#8-contributing)
  - [9. License](#9-license)

---

## 1. Features

* **Headless Wi-Fi Provisioning:** Connects your device to a Wi-Fi network without requiring a display or input devices.
  
* **Temporary Access Point (AP):** Creates its own Wi-Fi network on first boot or fallback.

* **Captive Portal:** Automatically redirects connected devices to the setup web interface. Or connect to the device with the mDNS name (e.g., http://pifigo.local)

* **Automatic Hostname Setup:** Configures mDNS (`.local`) hostname for easy access.

* **"Universal" Network Manager Support:** Automatically detects the system's active network manager (`NetworkManager`, `dhcpcd`, or `systemd-networkd`) and adapts its Wi-Fi configuration strategy accordingly.

* **Persistent Device ID & Claim Code:** Generates and displays unique identifiers for your device for subsequent application setup.

* **Cross-Platform Binaries:** Pre-compiled Go binaries available for various ARM architectures (ARMv6, ARMv7, ARM64).

* **Customizable UI:** Easily change styling, text, and logos via simple TOML configuration files.

* **COMING SOON: Robust & Recoverable:** Includes automatic fallback to AP mode if internet connection is lost.

## 2. Why pifigo?

**pifigo** is designed to enable appliance-style SBC-based solutions (e.g., Raspberry Pi, Orange Pi, etc.) with a reliable and streamlined way to get the device online, acting as the crucial first step for any IoT or embedded Linux project that needs Wi-Fi connectivity and subsequent configuration.

**This project is not production ready** and in the test phase. Use at your own risk. 

## 3. Requirements

* **Hardware:** A Linux-based embedded device with a Wi-Fi adapter (e.g., Raspberry Pi Zero W, Raspberry Pi Zero 2 W, Raspberry Pi 3/4, Orange Pi Zero 3, etc.).
* **Operating System:** A minimal Linux distribution (e.g., Raspberry Pi OS Lite, Armbian Minimal).
    * **:warning:** OS must have `systemd` as its init system.
    * **:warning:** Wi-Fi adapter drivers must be installed and functional.
* **ALL Required Packages (will be installed by `install.sh`):** `hostapd`, `dnsmasq`, `iptables-persistent`, `avahi-daemon`, `iproute2`, `network-manager`.

## 4. TLDR QUICK START GUIDE
**TLDR** *Download and extract the package and get it on your Pi device and then run the installer as root.*

1. Download and extract the [latest release](https://github.com/ToddE/pifigo/releases) on your target device (Raspberry Pi)
   
2. Review the config.toml file and make any changes to the default Access Point or passwords. (see [Configuring Pifigo](#5-configuring-pifigo))

3. Run the installer with sudo privileges:
  
  ```bash
  sudo ./install.sh
  ```
The installer will automatically detect your architecture, install the correct binary and configure the device before rebooting.

<!-- 
## 5. Detailed Installation Overview

This guide assumes you have a freshly flashed minimal Linux OS on your device and can connect to it via SSH (e.g., via USB gadget mode or temporary Ethernet).

### Step 1: Prepare Your Device OS
*While this information is placed here for convenience, you should follow the directions associated with your Pi or other device.*

1.  **Flash OS:** Use your preferred imager (e.g., Raspberry Pi Imager) to flash **Raspberry Pi OS Lite** (32-bit or 64-bit, depending on your device's capabilities), **Armbian Minimal** or a similar minimal Linux distribution to your microSD card.
2.  **:warning: During Imaging Process be sure to set and note the following**
    * **Set hostname:** e.g., `my-device` (this will be `my-device.local` via mDNS).
    * **Enable SSH:** Check "Enable SSH" and select "Use password authentication".
    * **Set username and password:** e.g., `pi` and `raspberry` (or your preferred secure credentials).
    * **Configure wireless LAN:**
        * **If your device has Ethernet (e.g., Pi 3/4):** Leave this blank and *use Ethernet for initial SSH access* if you can. 
        * **If your device is Wi-Fi-only (e.g., Pi Zero W/2W, some Orange Pis):** **You MUST configure your home Wi-Fi here** to gain initial SSH access. **pifigo** will then detect your network manager and reconfigure Wi-Fi for its setup AP after installation.
    * **Set locale settings:** Choose your correct **Wi-Fi country** (e.g., `US`, `GB`, `DE`). This is absolutely essential for Wi-Fi to function correctly and legally.
3.  **Boot Device:** Insert the SD card into your device and power it on.
4.  **Initial Access:** Connect to your device via SSH (e.g., using USB Gadget Mode for Pi Zeros: `ssh pi@raspberrypi.local` or `ssh pi@192.168.7.2`). If you configured Wi-Fi, you can SSH to the IP your router assigns (`ssh [username]@[hostname/ip].local` or find its IP on your router).

### Step 2: Obtain pifigo Binaries & Installer 
1. Download and extract the [latest release](https://github.com/ToddE/pifigo/releases)
2. Extract/unzip the files on your target device. For example
   ```bash
   scp -r 
3. 

4.  **Transfer files to your device:**

    #### Example: Adjust binary name based on your choice and version
    ```bash
    BINARY_NAME="pifigo_1.0.0_linux_armv7" # or _linux_arm64, etc.
    ```

    #### Create a temporary staging directory on the device 
    NOTE: replace `[username]` with the username and `[hostname/ip]` you use to login to your device
    ```bash
    ssh [username]@[hostname/ip] "mkdir -p /tmp/pifigo_staging"
    ```

    #### Transfer the chosen binary and the installer script, config, lang, and assets
    ```bash
    scp "./releases/$BINARY_NAME" [username]@[hostname/ip]:"/tmp/pifigo_staging/pifigo"
    scp "./install.sh" [username]@[hostname/ip]:"/tmp/pifigo_staging/install.sh"
    scp "./config.toml" [username]@[hostname/ip]:"/tmp/pifigo_staging/config.toml"
    scp -r "./lang" [username]@[hostname/ip]:"/tmp/pifigo_staging/"
    scp -r "./cmd/pifigo/assets" [username]@[hostname/ip]:"/tmp/pifigo_staging/assets" # Copy the assets source dir
    ```

### Step 3: Run the Installer Script

On your **device's SSH terminal**:

1.  **Navigate to the staged installer directory:**
    ```bash
    cd /tmp/pifigo_staging/
    ```
2.  **Run the installer script:**
    ```bash
    sudo ./install.sh
    ```
    The script will:
    * Update system packages and install necessary dependencies.
    * Detect and configure the primary network manager (`NetworkManager`, `dhcpcd`, or `systemd-networkd`).
    * Copy **pifigo** binary, config files, and assets to their final system locations (`/usr/local/bin/pifigo`, `/etc/pifigo/`, `/var/lib/pifigo/`).
    * Configure **pifigo** as a `systemd` service.
    * Set up `sudoers` permissions.
    * Finally, trigger a reboot.

## 5. Initial Setup Flow (User Guide)
!-->

### After your device reboots (wait 30-60 seconds)

1.  **Connect to pifigo's AP:**
    * On your mobile phone or PC, open your Wi-Fi settings.
    * Look for a new Wi-Fi network named **`PiFigoSetup`** (this is the default, check your `config.toml` for `ap_ssid` if changed).
    * Connect to it using the password you set in `config.toml` (`87654321` by default for `ap_password`).
2.  **Access the Captive Portal:**
    * Your device should automatically pop up a browser window for a "Wi-Fi Login" or "Sign in to network" page.
    * If not, open your web browser and navigate to `http://pifigo.local/` (this is the default hostname, check your `config.toml` for `device_hostname` if changed).
3.  **Perform Wi-Fi Configuration:**
    * The web page will display:
        * Your device's unique **Device ID** and **Claim Code**. **Note these down!** They are crucial for subsequent application setup.
        * A list of nearby Wi-Fi networks.
    * Select your home Wi-Fi network from the list (or manually enter its SSID).
    * Enter your Wi-Fi password.
    * Click "Connect."
    * The page will show a "Success!" message and tell the device will reboot.
4.  **Reconnect to Your Home Wi-Fi:**
    * Your mobile/PC will lose connection to `PiFigoSetup`.
    * Reconnect your mobile/PC back to *your primary home Wi-Fi network*.
5.  **Device is Now Online:**
    * Your device (Pi) should now be connected to your home Wi-Fi.
    * You can then access its local web-based service interface (like `randao-node-manager` if installed) by navigating your browser to `http://<your_device_hostname>.local/` (e.g., `http://pifigo.local/`).

## 5. Configuring pifigo

**pifigo**'s behavior and appearance can be customized via its `config.toml` file and by providing custom assets.

### Main Config (`config.toml`)

The primary configuration file is located at `/etc/pifigo/config.toml` on your device.

```toml
# config.toml - Configuration for pifigo (Wi-Fi Setup Service)

# --- UI CUSTOMIZATION SETTINGS ---
[ui]
page_title = "PiFigo Setup"          # Main title for browser tab/window
heading_text = "Connect Your Device to WiFi" # Main heading on the page
body_font = "Arial"                   # Font family for the page body
background_color = "#f0f2f5"          # Background color of the entire page
text_color = "#333"                   # Default text color
container_color = "#ffffff"           # Background color of the main content box
heading_color = "#007bff"             # Color for <h1> headings

# Custom image for logo/branding. Path is relative to /etc/pifigo/assets/
# Example: If you place 'my_logo.svg' in /etc/pifigo/assets/, set this to "my_logo.svg"
custom_image_url = "randao_logo.png" # Default is the embedded logo

# Optional: Path to a completely custom HTML template file to override the default.
# Path is relative to /etc/pifigo/assets/. If empty, the embedded default template is used.
# custom_template = "my_custom_template.html"


# --- NETWORK SETTINGS ---
[network]
ap_ssid = "PiFigoSetup"                 # SSID of the temporary Wi-Fi Access Point
ap_password = "pifigo_pass"             # Password for the temporary AP (min 8 chars). **CHANGE THIS DEFAULT IN PRODUCTION!**
ap_channel = 7                          # Wi-Fi channel for the Access Point (1-11 recommended for 2.4GHz)
wifi_country = "US"                     # IMPORTANT: Your Wi-Fi regulatory domain (e.g., US, GB, DE). Incorrect setting can cause Wi-Fi issues.
device_hostname = "pifigo-device"       # Hostname for mDNS (e.g., "[http://pifigo-device.local/](http://pifigo-device.local/)").

# --- LANGUAGE SETTING (specific to pifigo) ---
[language] # This section is for pifigo's UI language settings
default_lang = "en" # Default language code (e.g., "en", "fr", "es")

# --- RUNTIME SETTINGS (Automatically set by install.sh) ---
# This section is managed by the install.sh script and tells pifigo which network manager to use.
# DO NOT EDIT MANUALLY unless you know exactly what you are doing.
[runtime]
network_manager_type = "NetworkManager" # Example: This will be set to "NetworkManager", "dhcpcd", or "systemd-networkd"

```
### Localization (`lang/`)

Language strings are stored in TOML files within `/etc/pifigo/lang/`. The `language` setting in `/etc/pifigo/config.toml` (specifically `default_lang`) determines which file is loaded (e.g., `language = "fr"` loads `/etc/pifigo/lang/fr.toml`).

To add a new language, create a new TOML file (e.g., `es.toml`) in `/etc/pifigo/lang/` and update `config.toml`.

### Custom Assets (`assets/`)

You can override the default embedded logo (or provide your own custom CSS/JS files if you build a custom template) by placing them in `/etc/pifigo/assets/`. Ensure the `custom_image_url` (or `custom_template`) in `config.toml` points to your custom files.

## 6. Management

**pifigo** is managed via `systemd` commands.

### pifigo Service Control

* **Check Status:**
    ```bash
    sudo systemctl status pifigo.service
    ```
* **View Live Logs:**
    ```bash
    sudo journalctl -u pifigo.service -f
    ```
* **Stop Service (and force AP mode if running):**
    ```bash
    sudo systemctl stop pifigo.service
    # This will typically bring down the AP if it's active.
    # To restart AP, follow "Forcing AP Mode" below.
    ```
* **Disable Service (prevent starting on boot):**
    ```bash
    sudo systemctl disable pifigo.service
    ```
* **Enable Service (allow starting on boot):**
    ```bash
    sudo systemctl enable pifigo.service
    ```

### Forcing AP Mode / Resetting Wi-Fi

If your device is stuck offline, or you want to connect it to a new Wi-Fi network:

1.  **Ensure core network managers are stopped:**
    ```bash
    sudo systemctl stop NetworkManager.service
    sudo systemctl stop dhcpcd.service # If that's your manager
    sudo systemctl stop systemd-networkd.service # If that's your manager
    ```
2.  **Enable and Start pifigo:**
    ```bash
    sudo systemctl enable pifigo.service
    sudo systemctl start pifigo.service
    ```
3.  **Reboot:** (Recommended for a clean state)
    ```bash
    sudo reboot
    ```
    On reboot, **pifigo** will start in AP mode again.

### Updating pifigo

1.  **Download/Transfer New Binary:** On your development machine, build the new **pifigo** binary for your device's architecture (e.g., `pifigo_0.0.2_linux_armv7`). Use `scp` to copy it to a temporary location on your device (e.g., `/tmp/pifigo_new`).
2.  **Stop Existing Service:**
    ```bash
    sudo systemctl stop pifigo.service
    ```
3.  **Copy New Binary:**
    ```bash
    sudo cp /tmp/pifigo_new /usr/local/bin/pifigo
    ```
4.  **Restart Service:**
    ```bash
    sudo systemctl start pifigo.service
    ```
5.  **Clean up:** `rm /tmp/pifigo_new`

### Uninstalling pifigo

The `uninstall.sh` script (located in the **pifigo** repository) will remove all **pifigo** components and attempt to restore your system's network configuration to its state before **pifigo** was installed.

1.  **Transfer `uninstall.sh`:** Copy the `uninstall.sh` script from your development machine's **pifigo** repository to your device (e.g., `/tmp/uninstall_pifigo.sh`).
2.  **Run Uninstall Script:**
    ```bash
    sudo /tmp/uninstall_pifigo.sh
    ```
    The script will print its actions and reboot the device.

## 7. Troubleshooting

* **AP Not Appearing (`PiFigoSetup` not visible):**
    * Ensure your device is powered on.
    * Connect via USB Gadget Mode/Ethernet.
    * Check `sudo systemctl status pifigo.service`.
    * View logs: `sudo journalctl -u pifigo.service -f`. Look for errors related to `hostapd` or your Wi-Fi interface (e.g., `wlan0`, `wlpXsY`).
    * Verify `sudo systemctl status NetworkManager.service` (it should be stopped/inactive when pifigo is running its AP).
    * Confirm `hostapd` and `dnsmasq` are unmasked: `sudo systemctl status hostapd.service dnsmasq.service`. They should not be "masked".
    * Double-check `config.toml` for correct `network.wifi_country` code.
* **Captive Portal Not Redirecting:**
    * Ensure you are truly connected to `PiFigoSetup` Wi-Fi.
    * Try opening `http://192.168.4.1/` manually in your browser.
    * Check `sudo iptables -t nat -L -v -n` on the device for the redirection rules.
    * View `dnsmasq` logs: `sudo journalctl -u dnsmasq.service -f`.
* **Device Not Connecting to Home Wi-Fi:**
    * Check Wi-Fi password in captive portal (common error).
    * After reboot, connect via USB Gadget Mode/Ethernet.
    * Check `sudo systemctl status NetworkManager.service` (or `dhcpcd.service`/`systemd-networkd.service`, depending on what `install.sh` detected/configured). It should be `active`.
    * Check `sudo nmcli connection show` and `sudo nmcli device status wlan0` (or your interface name).
    * View `NetworkManager` logs: `sudo journalctl -u NetworkManager.service -f`.
    * Verify `config.toml`'s `network.wifi_country` is correct.
* **`pifigo-device.local` not resolving:**
    * Ensure `avahi-daemon` is installed and running (`sudo systemctl status avahi-daemon.service`).
    * Confirm your client device (phone/PC) supports mDNS/Bonjour.
    * Try finding the device's IP via your router's admin page or a network scanner tool (like "Fing" mobile app) and access `http://<IP_address>/` directly.

## 8. Contributing
We welcome contributions to **pifigo**! Please see the [CONTRIBUTING.md](https://github.com/ToddE/pifigo/blob/main/CONTRIBUTING.md) file (if you create one) for guidelines on how to contribute.

## 9. License

This project is licensed under the MIT License - see the [LICENSE.md](https://github.com/ToddE/pifigo/blob/main/LICENSE.md) file for details.