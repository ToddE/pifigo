## Creating a Hotspot with Local DHCP Server and mDNS on Armbian Minimal

Setting up a WiFi hotspot on a minimal Armbian installation involves configuring the wireless interface as an access point, running a local DHCP server to assign IP addresses, and enabling mDNS for local service discovery. Here's a general outline based on available information, though specific configurations may vary based on your Armbian version and hardware: 

### 1. Install necessary packages:
- `hostapd`: This package is used to configure and manage the wireless interface as an access point.
  
- `dnsmasq`: This package provides both DNS and DHCP services, allowing the hotspot to provide IP addresses and resolve local hostnames.

- `avahi-daemon`: (Maybe) enables mDNS/DNS-SD by implementing Apple's Zeroconf architecture (also known as "Rendezvous" or "Bonjour"). This allows your Armbian device to be discoverable on the network using its hostname followed by ".local" (e.g., <hostname>.local). 

You may need other packages depending on your specific setup, such as tools for bridging networks if you want to share an existing wired connection. 

```bash
sudo apt update
sudo apt install hostapd dnsmasq 
# there may be no need to install avahi if systemd-resolved is used
sudo apt install avahi-daemon libnss-mdns libnss-mymachines #if systemd-resolved is not in use
```

### 2. Configure hostapd:
You'll need a configuration file for `hostapd`, typically located at `/etc/hostapd/hostapd.conf`.

This file defines the basic hotspot settings, including the network name (SSID), password, and operating channel.

<ins>**Example:**</ins>  configuration lines:  `/etc/hostapd/hostapd.conf`

```yaml
interface=wlan0  # Replace with your wireless interface name
ssid=MyArmbianHotspot
wpa_passphrase=MySecurePassword
# ... other hostapd configuration options
```
another example:  `/etc/hostapd/hostapd.conf`
```yaml
interface=wlan0  # Replace wlan0 with your actual wireless interface name
driver=nl80211
ssid=MyWiFiNetwork # Replace with your desired SSID
hw_mode=g
channel=6
wmm_enabled=0
macaddr_acl=0
auth_algs=1
ignore_broadcast_ssid=0
wpa=2
wpa_key_mgmt=WPA-PSK
wpa_pairwise=TKIP
rsn_pairwise=CCMP
wpa_passphrase=MyWiFiPassword  # Replace with your desired password
# Static IP configuration (example)
# You'll need to configure your network interface with the static IP as well (see below)
# This part is handled by your network configuration, not hostapd directly
# address=192.168.1.1
# netmask=24
# gateway=192.168.1.254
# dns=8.8.8.8,8.8.4.4
```

Enable and start the `hostapd` service:

```bash
sudo systemctl daemon-reload # always do this after updating system files
sudo systemctl unmask hostapd.service # just in case it is masked
sudo systemctl enable hostapd.service
sudo systemctl start hostapd.service
```

**Note:** Some Armbian setups might have `hostapd` masked by default, so you might need to unmask it before enabling. 

### 3. Configure `dnsmasq `for DHCP:
Edit the 'dnsmasq' configuration file, typically located at '/etc/dnsmasq.conf'.

Get the EXACT name of your wireless lan interface (e.g., wlan0)
```bash
ip link show | cut -d" " -f2 | grep wl | tr -d ': '
```

Configure a DHCP range for the clients that connect to your hotspot. 

<ins>**Example:**</ins> configuration lines: '/etc/dnsmasq.conf`
```yaml
interface=wlan0  # Ensure this matches your wireless interface
dhcp-range=10.10.1.50,10.10.1.199,12h  # Assign IPs from 10.10.1.50 to 10.10.1.199 for 12 hours
 # ... other dnsmasq configuration options
 ```

Enable and start the `dnsmasq` service:
```bash
sudo systemctl daemon-reload # always do this after updating system files
sudo systemctl unmask dnsmasq.service # just in case
sudo systemctl enable dnsmasq.service
sudo systemctl start dnsmasq.service
```

### 4. Enable mDNS
mDNS allows devices on the local network to discover each other by name, without needing a traditional DNS server.

You can typically enable mDNS through `systemd-resolved` or `NetworkManager`. Armbian minimal (our target platform for appliances) ships with `systemd-resolved` because it is less resource intensive than `NetworkManager`.

#### Using systemd-resolved:
Edit `/etc/systemd/resolved.conf` and uncomment/set 

```yaml
MulticastDNS=yes.
```

Restart `systemd-resolved`:
```bash
sudo systemctl daemon-reload # always do this after updating system files
sudo systemctl unmask systemd.resolved.service #just in case
sudo systemctl enable systemd.resolved.service  
sudo systemctl restart systemd-resolved.service
```

#### Using NetworkManager:

Use `nmcli` to modify the connection and enable mDNS. <ins>**Example:**</ins>

```bash
sudo nmcli connection modify <connection_name> connection.mdns 2
```

Replace `<connection_name>` with the name of your wireless connection. 

### OPTIONAL: Consider Network Bridging

If you want to share an existing wired internet connection through your hotspot, you'll need to set up a network bridge.

This involves creating a bridge interface and adding your wired and wireless interfaces to it. 

### Important Notes:

**Wireless Interface Name:** Ensure you use the correct name for your wireless interface (e.g., wlan0) throughout the configuration. To identify the name of your wireless interface using `ifconfig -a` or `ip link show`.
Configure the wireless interface with a static IP address.
        Set up a DHCP server (e.g., using dnsmasq) to assign IP addresses to connecting clients.

**Adjust Firewall:** You may need to configure your firewall (e.g., using iptables) to allow traffic for DHCP and DNS services on the wireless interface. 

**NetworkManager vs. Manual Setup:** Armbian offers `armbian-config` for network setup, which can simplify the process. However, you can also perform the setup manually by editing configuration files. I personally have had no luck with the `armbian-config` approach.
 
 Manual Configuration:
        Identify the name of your wireless interface using ifconfig -a or ip link show.
        Configure the wireless interface with a static IP address.
        Set up a DHCP server (e.g., using dnsmasq) to assign IP addresses to connecting clients. 
    
**IP Addressing:** Carefully configure the IP addresses and subnets to avoid conflicts with your existing network infrastructure.

**Troubleshooting:** Check the logs of `hostapd` and `dnsmasq` using `journalctl` for any errors or warnings.
    
**Security:** Use strong passwords for your WiFi hotspot. 

----



## Raspian Lite 

By default, a minimal Raspberry Pi OS installation (like the Lite version, often used for IoT projects on devices like the Pi Zero) does not use NetworkManager. Instead, it relies on two primary components for network management: 

- `dhcpcd`: This is the default DHCP client used to obtain an IP address from your network's DHCP server.
- `wpa_supplicant`: This handles connecting to Wi-Fi networks, especially for securing wireless connections. 

### Configuring Network on a Minimal Pi Zero: 
For headless setup (without a display), you can pre-configure the Wi-Fi details by placing a wpa_supplicant.conf file in the boot partition of your SD card. The wpa_supplicant.conf file contains the Wi-Fi network information, allowing the Pi to connect automatically upon boot. 

### NetworkManager as an Option:
While NetworkManager is not the default, it is available as an option in later releases of Raspberry Pi OS. You can enable NetworkManager using raspi-config in the advanced menu.

---
## Setting up a WiFi Hotspot with Local DHCP and DNS on Raspbian Lite
To create a WiFi hotspot on your Raspberry Pi running Raspbian Lite, complete with a local DHCP server and DNS name resolution, you'll need to install and configure a few packages, mainly hostapd and dnsmasq. 
Here's a breakdown of the process:
### 1. Install Required Packages:

Install hostapd to create the WiFi access point and dnsmasq to manage DHCP and DNS services.
```bash
sudo apt update
sudo apt install hostapd dnsmasq
```


### 2. Configure a Static IP for the Hotspot Interface (e.g., wlan0):
Edit the network configuration file (likely `/etc/dhcpcd.conf`) to set a static IP for your wireless interface.

```bash
sudo vi /etc/dhcpcd.conf
```

or 

```bash
sudo nano /etc/dhcpd.conf
```

Add the following lines (adjusting the IP address as needed for your desired network):

```yaml
interface wlan0
static ip_address=192.168.4.1/24
nohook wpa_supplicant
```


### 3. Configure DHCP and DNS using Dnsmasq:
Create a new configuration file for dnsmasq.

```bash
sudo nano /etc/dnsmasq.d/hotspot.conf
```
(I prefer vi but whatever)


Add the following configuration to define the DHCP range and specify the Raspberry Pi as the DNS server:
```yaml
# Gateway + DNS server
dhcp-option=3,192.168.4.1
dhcp-option=6,192.168.4.1
# Let the Raspberry Pi resolve all DNS queries
address=/#/192.168.4.1
```
 

### 4. Configure the Access Point (Hostapd):
Edit the hostapd default configuration file.

```bash
sudo vi /etc/default/hostapd
```

Add the following configuration to define your network name (SSID), channel, and mode:

```yaml
# Set the interface used by the access point
INTERFACE="wlan0"
# Set the SSID
SSID="YourHotspotName"
# Set the operating mode (e.g., g for 2.4 GHz)
HW_MODE="g"
# Set the channel
CHANNEL="1"
# Enable WPA encryption
WPA="1"
WPA_PASSPHRASE="YourPassword"
WPA_DRIVER="nl80211" #validate that this is correct
```

### 5. Enable IP Forwarding (not necessary for pifigo project)
To allow devices connected to the hotspot to potentially access the internet (if your Raspberry Pi has internet connectivity), you'll need to enable IP forwarding.

```bash
sudo vi /etc/sysctl.conf
```

Uncomment the line net.ipv4.ip_forward=1 by removing the # at the beginning of the line. 

### 6. Add an Iptables Rule (for Internet Sharing):
If you're sharing an internet connection (e.g., from an Ethernet connection), add an iptables rule to forward traffic.
```bash
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```
**Note:** Replace eth0 with the appropriate network interface for your internet connection if needed. 

### 7. Restart Services:
Restart the necessary services to apply the changes.
```bash
sudo systemctl unmask hostapd
sudo systemctl enable hostapd
sudo systemctl start hostapd
sudo systemctl start dnsmasq
```
 

###Important Notes:
* **Offline Mode:** The configuration for dnsmasq provided above makes the Raspberry Pi resolve all DNS queries locally, meaning devices connected to the hotspot won't have internet access.
  
* **Internet Sharing:** To allow internet access for devices on the hotspot, you would typically need a separate network connection (like Ethernet) and configure network address translation (NAT) using iptables.

* **DNS Resolution:** If you need the Raspberry Pi to also act as a DNS server for devices on the hotspot to access the internet, you may need to configure additional DNS settings or use a service like Pi-hole.

* **Conflicts:** Be aware that having multiple DHCP servers on the network can cause conflicts. 