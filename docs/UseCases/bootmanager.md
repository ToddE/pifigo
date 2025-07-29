# Use Case: Manage Network State on Boot

## 1. Actors

    Primary Actor: pifigo System (The application itself)

    Supporting Actor: User

## 2. Goal

To ensure the device either gets configured by a User or automatically connects to a previously saved network within a defined time window after booting up.

## 3. Trigger

The device boots, and the systemd service for pifigo is started.

## 4. Preconditions

- The pifigo service is installed and enabled.
- A timeout_seconds value is defined in /etc/pifigo/config.yaml.
- The device starts in "Hotspot Mode" by default.

## 5. Scenarios

### A. Main Success Scenario (User Configures WiFi)

1. The pifigo system starts on boot.
2. The bootmanager component starts its countdown timer (e.g., for 600 seconds).
3. Simultaneously, the server component starts the web server, broadcasting the configuration hotspot.
4. Before the timer expires, the User connects their phone/laptop to the pifigo hotspot.
5. The User accesses the web portal and submits valid credentials for their local Wi-Fi network.
6. The server component successfully generates the new netplan configuration.
7. The server component sends a "success" signal to the bootmanager component.
8.  The bootmanager receives the signal, immediately cancels its countdown timer, and exits its process.
9.  The server component proceeds to switch the device from Hotspot Mode to Client Mode.

**Postcondition:** The device is successfully connected to the User's Wi-Fi network, and the boot manager's fallback logic is cancelled for the current boot cycle.

### B. Alternative Scenario (Timeout Reached)

1. The pifigo system starts on boot.
2. The bootmanager component starts its countdown timer.
3. The server component starts the web server in hotspot mode.
4. The User does not interact with the web portal
5. The bootmanager's timer expires.
6. The bootmanager checks for the existence of a previously saved /etc/pifigo/last-good-wifi.yaml file.
7. The bootmanager finds the file and initiates the network switch.
8. It stops the hotspot services (hostapd, dnsmasq) and applies the "last-good" netplan configuration.
9. The bootmanager exits its process.

**Postcondition:** The device automatically connects to the last known good Wi-Fi network without any user interaction.

#### Alternate Path to Basic Path B #6 (Timeout with No Saved Network)

6. The bootmanager checks for /etc/pifigo/last-good-wifi.yaml but does not find it (this is typical on the very first boot).

7. The bootmanager takes no further action and exits its process.

**Postcondition: **The device remains in Hotspot Mode indefinitely, waiting for a User to perform the initial configuration.