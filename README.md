![pifigo](/docs/image/pifigo.png) 
# pifigo: Wi-Fi Configuration for Headless Systems

**ATTN THOSE WHO CARE: Access the [DEV/Implementation Doc](/DEVELOPER.md)**

### WARNING!!!  
  **This solution is still in testing. Be careful when using or experimenting with this. It can make your device unaccessible.**

PiFigo is a utility designed to facilitate the initial network configuration for headless devices, such as Raspberry Pi or Orange Pi single-board computers. The application addresses the common challenge of connecting such devices to a wireless network without the need for a dedicated keyboard, monitor, or other peripherals. 

**Pifigo is originally being built to create an “appliance” based on Armbian running on an Orange Pi Zero 3 board.**

The system functions by transforming the device into a temporary Wi-Fi access point. This allows a user to establish a direct connection from a personal computer or mobile device and access a web-based portal. 

Through this interface, the user can select a local Wi-Fi network and provide the necessary credentials. Upon successful configuration, the device will automatically connect to the designated network.

## **Core Features**

- **Simplified Headless Setup:** The system obviates the need for direct peripheral access (keyboard, mouse, monitor) for initial network configuration.
- **Web-Based Interface:** All configuration is managed through a simple and responsive web portal, ensuring compatibility with most modern devices.
- **Automatic Reconnection:** Following a system reboot, the device will automatically attempt to re-establish a connection with the last successfully configured wireless network.
- **Environmental Portability:** If the device is moved to a new location where the previously configured network is unavailable, the setup hotspot will automatically reactivate, allowing for straightforward reconfiguration.

## **Installation Procedure**

The recommended method for installing PiFigo is via the pre-built Debian (.deb) software package.

### **Step 1: Download the Software Package**

1. Navigate to the official **PiFigo GitHub Releases page**. (Not yet available)
   
2. Locate the most recent release and download the .deb file that corresponds to your device's architecture (e.g., pifigo\_0.0.1\_arm64.deb for most 64-bit ARM-based systems).

### **Step 2: System Installation**

1. Transfer the downloaded .deb file to the target device. For devices already connected via a wired Ethernet connection, a utility such as scp may be utilized.
  
2. Establish a command-line session with the device (e.g., via SSH).
  
3. Execute the following sequence of commands to install the package and its required software dependencies.  
   
   - First, execute the package installer. This step may report dependency errors, which is an expected outcome.  Replace <version> with the proper version number.
        ```bash
        sudo dpkg -i pifigo_<version>_arm64.deb
        ```
    
   - Next, execute this command to resolve and install any missing dependencies automatically.  
        ```bash
        sudo apt-get install -f
        ```
4. To complete the installation and initiate the service, reboot the device.  
    ```bash
    sudo reboot
    ```

## **Operational Guide**

Upon completion of the installation and subsequent reboot, the device will be ready for network configuration.

1. **Connect to the Hotspot:** Using a personal computer or mobile device, scan for available Wi-Fi networks. Connect to the network named **"PiFigoSetup"** using the password **87654321**.
   
2. Access the Web Portal: Once a connection is established, open a web browser and navigate to the following address:  
  http://pifigo.local

3. **Configure the Wi-Fi Connection:**
  - The web portal will automatically display a list of detected wireless networks in the vicinity.
  - Select the desired network from the list.
  - Enter the corresponding password for that network.
  - Click the "Connect" button to submit the configuration.
  
4. **Process Completion:** The device will then disengage from hotspot mode and attempt to connect to the specified Wi-Fi network. The "PiFigoSetup" access point will no longer be broadcast. At this point, reconnect your computer or mobile device to your primary Wi-Fi network. The PiFigo device should now be accessible on the local network.

## Command-Line Interface (CLI) for Administration

The pifigo binary includes a set of command-line flags for troubleshooting and administration. These are intended to be used by an administrator connected to the device (e.g., via SSH over Ethernet).

| Flag                 | Description                                                               |
| :------------------- | :------------------------------------------------------------------------ |
| \--status            | Shows the current mode (Hotspot/Client) and checks internet connectivity. |
| \--list-saved        | Lists the SSIDs of all saved network profiles.                            |
| \--last-good         | Shows which saved network is the current default for the boot manager.    |
| \--set-good \<SSID\> | Manually sets the default fallback network to a specific saved profile.   |
| \--forget \<SSID\>   | Deletes a saved network profile.                                          |
| \--force-hotspot     | Forces the device into hotspot mode. Used by the watchdog or an admin.    |
| \--version           | Prints the application version.                                           |
| \-v, \--verbose      | Enables verbose logging on startup.                                       |
| \-h, \--help         | Displays the help message with all available flags.                       |

## Troubleshooting

- **"PiFigoSetup" Hotspot is Not Visible:** Ensure the device has been rebooted after the installation process was completed. Verify that the device's Wi-Fi hardware is enabled.

- **The Address http://pifigo.local is Unreachable:** Certain network environments may interfere with mDNS resolution. As an alternative, you may attempt to navigate directly to the device's static IP address: **http://192.168.4.1**.

- **Reconfiguring the Device for a New Location:** Power on the device in the new location. If it is unable to connect to the previously configured network, the "PiFigoSetup" hotspot is designed to reactivate automatically after a few minutes. This will allow you to repeat the configuration process for the new network.