#!/bin/bash
# uninstall.sh - Uninstalls the pifigo application and attempts to restore
#                the system to its state before installation using backups.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration Variables (must match install.sh) ---
APP_NAME="pifigo"
APP_CONFIG_DIR="/etc/$APP_NAME"
APP_DEVICE_DATA_DIR="/var/lib/$APP_NAME"
APP_BACKUP_DIR="$APP_DEVICE_DATA_DIR/.backup" # NEW: Backup directory

APP_BINARY_DEST="/usr/local/bin/$APP_NAME"
APP_SYSTEMD_SERVICE_PATH="/etc/systemd/system/$APP_NAME.service"

# Network config files managed by pifigo
HOSTAPD_CONF_PATH="/etc/hostapd/hostapd.conf"
DNSMASQ_CONF_PATH="/etc/dnsmasq.conf"
DHCPCD_CONF_PATH="/etc/dhcpcd.conf"
SYSCTL_CONF_PATH="/etc/sysctl.conf"
DEFAULT_HOSTAPD_PATH="/etc/default/hostapd"
WPA_SUPPLICANT_PATH="/etc/wpa_supplicant/wpa_supplicant.conf"


# --- Check for root privileges ---
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root. Please use sudo."
   exit 1
fi

echo "--- Starting pifigo Uninstallation ---"

# --- 1. Stop and Disable pifigo Service ---
echo "Stopping and disabling $APP_NAME service..."
systemctl stop "$APP_NAME".service || true
systemctl disable "$APP_NAME".service || true
rm -f "$APP_SYSTEMD_SERVICE_PATH" || true
systemctl daemon-reload # Reload systemd to remove the service

# --- 2. Delete pifigo Files and Directories ---
echo "Deleting $APP_NAME application files and directories..."
rm -rf "$APP_CONFIG_DIR" || true
# Leave APP_DEVICE_DATA_DIR and APP_BACKUP_DIR for now, as they contain backups

# --- 3. Restore System Configuration Files from Backup ---
echo "Restoring system configuration files from backup..."
RESTORE_SUCCESS=true

# Function to restore the latest backup of a file
restore_latest_backup() {
    local original_path="$1"
    local backup_name=$(basename "$original_path")
    local latest_backup=$(ls -t "${APP_BACKUP_DIR}/${backup_name}"*.bak 2>/dev/null | head -n 1)

    if [ -n "$latest_backup" ]; then
        echo "Restoring $original_path from $latest_backup"
        cp -p "$latest_backup" "$original_path"
    else
        echo "No backup found for $original_path. Leaving as is (might be default or modified by others)."
        RESTORE_SUCCESS=false # Indicate a partial restore for this file
    fi
}

# Restore specific files
restore_latest_backup "$DHCPCD_CONF_PATH"
restore_latest_backup "$HOSTAPD_CONF_PATH"
restore_latest_backup "$DNSMASQ_CONF_PATH"
restore_latest_backup "$SYSCTL_CONF_PATH"
restore_latest_backup "$DEFAULT_HOSTAPD_PATH"
restore_latest_backup "$WPA_SUPPLICANT_PATH"
restore_latest_backup "/etc/systemd/system/NetworkManager.service" # Restore unit files if they were masked
restore_latest_backup "/etc/systemd/system/NetworkManager.socket"
restore_latest_backup "/etc/systemd/system/dhcpcd.service"
restore_latest_backup "/etc/systemd/system/dhcpcd.socket"
restore_latest_backup "/etc/systemd/system/systemd-networkd.service"
restore_latest_backup "/etc/systemd/system/systemd-networkd.socket"
restore_latest_backup "$APP_SYSTEMD_SERVICE_PATH" # Restore pifigo's unit (will be deleted after daemon-reload)

# Restore iptables rules
LATEST_IPTABLES_BACKUP=$(ls -t "${APP_BACKUP_DIR}/iptables_rules.v4."*.bak 2>/dev/null | head -n 1)
if [ -n "$LATEST_IPTABLES_BACKUP" ]; then
    echo "Restoring iptables rules from $LATEST_IPTABLES_BACKUP"
    iptables-restore < "$LATEST_IPTABLES_BACKUP"
else
    echo "No iptables backup found. Flushing current rules and saving empty set."
    iptables -F || true
    iptables -X || true
    iptables -t nat -F || true
    iptables -t nat -X || true
fi

# Restore sudoers file
LATEST_SUDOERS_BACKUP=$(ls -t "${APP_BACKUP_DIR}/sudoers."*.bak 2>/dev/null | head -n 1)
if [ -n "$LATEST_SUDOERS_BACKUP" ]; then
    echo "Restoring /etc/sudoers from $LATEST_SUDOERS_BACKUP"
    cp -p "$LATEST_SUDOERS_BACKUP" /etc/sudoers
else
    echo "No sudoers backup found. Please manually check /etc/sudoers for pifigo entries."
    # A safer approach for sudoers might be to remove specific lines added by install.sh (sed -i '/pifigo/d')
    # but for full restore, cp is used.
    RESTORE_SUCCESS=false
fi

# Remove the backup directory after restore
rm -rf "$APP_BACKUP_DIR" || true
# Remove the main device data directory after backups are gone
rm -rf "$APP_DEVICE_DATA_DIR" || true


# --- 4. Restore System's Primary Network Manager (NetworkManager) ---
echo "Restoring system's primary network manager to a clean state..."

# Unmask NetworkManager if it was masked (install.sh masks all initially)
systemctl unmask NetworkManager.service || true
systemctl unmask NetworkManager.socket || true

# Enable and start NetworkManager
systemctl enable NetworkManager.service || true
systemctl start NetworkManager.service || true

# Stop and disable other common managers (ensure NetworkManager takes full control)
systemctl stop dhcpcd.service || true
systemctl disable dhcpcd.service || true
systemctl mask dhcpcd.service || true # Mask again to ensure it stays off if not intended
systemctl mask dhcpcd.socket || true

systemctl stop systemd-networkd.service || true
systemctl disable systemd-networkd.service || true
systemctl mask systemd-networkd.service || true # Mask again
systemctl mask systemd-networkd.socket || true

# Also ensure hostapd and dnsmasq are disabled (they are for pifigo's AP)
systemctl stop hostapd || true
systemctl disable hostapd || true
systemctl mask hostapd.service || true # Mask to prevent conflicts
systemctl mask hostapd.socket || true # dnsmasq does not typically have a socket

systemctl stop dnsmasq || true
systemctl disable dnsmasq || true
systemctl mask dnsmasq.service || true # Mask to prevent conflicts


echo "NetworkManager should be active. System will reboot to apply full network state."

# --- 5. Final Reboot ---
echo "--- pifigo Uninstallation Complete! ---"
if $RESTORE_SUCCESS; then
    echo "System configuration files restored from backup."
else
    echo "Warning: Some configuration files could not be restored from backup. Manual check may be required."
fi
echo "The system will now reboot to apply changes."
echo "Your Pi should attempt to connect to Wi-Fi via NetworkManager on reboot."
echo "Rebooting in 5 seconds..."
sleep 5
reboot