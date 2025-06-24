#!/bin/bash
# install.sh - Installs and configures the pifigo application for Wi-Fi setup.
# This version proactively handles network managers and provides robust config updates.
# It now creates backups of original system configuration files.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- ANSI Color Codes ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color (resets to default)

# --- Application Version ---
VERSION="0.0.1" 

# --- Configuration Variables ---
APP_NAME="pifigo"
PROJECT_ROOT=$(pwd) 

APP_CONFIG_DIR="/etc/$APP_NAME"             # /etc/pifigo (for config.toml, lang/*)
APP_ASSETS_DEST_DIR="$APP_CONFIG_DIR/assets" # /etc/pifigo/assets (for optional external assets)
APP_LANG_DIR="$APP_CONFIG_DIR/lang"         # /etc/pifigo/lang
APP_DEVICE_DATA_DIR="/var/lib/$APP_NAME"    # /var/lib/pifigo (for device_id persistence)
APP_BACKUP_DIR="$APP_DEVICE_DATA_DIR/.backup" # NEW: Backup directory

APP_BINARY_DEST="/usr/local/bin/$APP_NAME"

APP_SYSTEMD_SERVICE_PATH="/etc/systemd/system/$APP_NAME.service"

# Source directory for assets (where go:embed finds them during build)
APP_ASSETS_SOURCE_DIR="$PROJECT_ROOT/cmd/$APP_NAME/assets" 

# --- Extract Wi-Fi AP info from config.toml for the final message ---
AP_SSID=$(grep 'ap_ssid =' "$PROJECT_ROOT/config.toml" | cut -d'"' -f2)
AP_PASSWORD=$(grep 'ap_password =' "$PROJECT_ROOT/config.toml" | cut -d'"' -f2)
DEVICE_HOSTNAME=$(grep 'device_hostname =' "$PROJECT_ROOT/config.toml" | cut -d'"' -f2)

# --- Determine the correct binary name to install based on target Pi's architecture ---
echo -e "${CYAN}Detecting target architecture to select correct $APP_NAME binary...${NC}"
ARCH_SUFFIX=""
case "$(uname -m)" in
    "armv6l") ARCH_SUFFIX="_linux_armv6" ;; # Raspberry Pi 1, Zero W
    "armv7l") ARCH_SUFFIX="_linux_armv7" ;; # Raspberry Pi 2/3/4, Zero 2 W (32-bit OS)
    "aarch64") ARCH_SUFFIX="_linux_arm64" ;; # Raspberry Pi 3/4, Zero 2 W (64-bit OS)
    *) echo -e "${RED}Error: Unknown ARM architecture $(uname -m). Cannot install $APP_NAME.${NC}" >&2; exit 1 ;;
esac
SOURCE_BINARY_NAME="${APP_NAME}_${VERSION}${ARCH_SUFFIX}" # e.g., pifigo_0.0.1_linux_armv7

# Verify the compiled binary exists
if [ ! -f "$PROJECT_ROOT/$SOURCE_BINARY_NAME" ]; then
    echo -e "${RED}Error: Required binary '$SOURCE_BINARY_NAME' not found in '$PROJECT_ROOT'.${NC}" >&2
    echo -e "${RED}Please ensure you have run './build-pifigo.sh' in the $APP_NAME project root to compile binaries for this target.${NC}" >&2
    exit 1
fi
echo -e "${GREEN}Detected architecture: $(uname -m). Will install '$SOURCE_BINARY_NAME'.${NC}"


# --- Check for root privileges ---
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}This script must be run as root. Please use sudo.${NC}"
   exit 1
fi

echo -e "${BLUE}--- Starting $APP_NAME Installation ---${NC}"

# --- 1. Update System & Install Core Dependencies ---
echo -e "${CYAN}Updating system packages and installing dependencies: hostapd, dnsmasq, iptables-persistent, avahi-daemon, iproute2...${NC}"
apt update && apt upgrade -y
# Crucial for headless: Automate answers to debconf prompts and config file handling
# -y: Assume yes to prompts
# -o Dpkg::Options::="--force-confdef": Use default config if new/changed, don't prompt
# -o Dpkg::Options::="--force-confold": Keep old config if new/changed, don't prompt
apt install -y \
    -o Dpkg::Options::="--force-confdef" \
    -o Dpkg::Options::="--force-confold" \
    hostapd dnsmasq iptables-persistent avahi-daemon iproute2 network-manager || true # NetworkManager might be there already

# --- 2. Create Application-Specific Configuration and Data Directories (and BACKUP dir) ---
echo -e "${CYAN}Creating application directories...${NC}"
mkdir -p "$APP_CONFIG_DIR"/{assets,lang}
mkdir -p "$APP_DEVICE_DATA_DIR"
mkdir -p "$APP_BACKUP_DIR" # NEW: Create backup directory
chmod 700 "$APP_DEVICE_DATA_DIR" 
chmod 700 "$APP_BACKUP_DIR" # Ensure backup dir is also secure


# --- Function to backup a file before modification ---
backup_file() {
    local file_path="$1"
    local backup_name=$(basename "$file_path")
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_path="${APP_BACKUP_DIR}/${backup_name}.${timestamp}.bak"

    if [ -f "$file_path" ]; then
        echo -e "${CYAN}Backing up $file_path to $backup_path${NC}"
        cp -p "$file_path" "$backup_path"
    else
        echo -e "${YELLOW}Note: $file_path not found, no backup created.${NC}"
    fi
}
# --- Function to backup iptables rules ---
backup_iptables() {
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_path="${APP_BACKUP_DIR}/iptables_rules.v4.${timestamp}.bak"
    echo -e "${CYAN}Backing up current iptables rules to $backup_path${NC}"
    iptables-save > "$backup_path"
}
# --- Function to backup sudoers ---
backup_sudoers() {
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_path="${APP_BACKUP_DIR}/sudoers.${timestamp}.bak"
    echo -e "${CYAN}Backing up /etc/sudoers to $backup_path${NC}"
    cp -p /etc/sudoers "$backup_path"
}


# --- NETWORK MANAGER DETECTION AND CONFIGURATION (Dynamic) ---
# This section configures the *system's default network manager* for the next boot.
# It does NOT stop actively running services during installation to avoid disruption.
echo -e "${CYAN}Ensuring system's network manager is correctly configured for next boot...${NC}"

# Detect currently running/enabled network managers
DETECTED_MANAGER_FOR_SYSTEM="unknown"
if systemctl is-active --quiet NetworkManager.service || systemctl is-enabled --quiet NetworkManager.service; then
    DETECTED_MANAGER_FOR_SYSTEM="NetworkManager"
elif systemctl is-active --quiet systemd-networkd.service || systemctl is-enabled --quiet systemd-networkd.service; then
    DETECTED_MANAGER_FOR_SYSTEM="systemd-networkd"
elif systemctl is-active --quiet dhcpcd.service || systemctl is-enabled --quiet dhcpcd.service; then
    DETECTED_MANAGER_FOR_SYSTEM="dhcpcd"
fi

echo -e "${YELLOW}Primary network manager detected for system: ${DETECTED_MANAGER_FOR_SYSTEM}${NC}"

# Mask ALL *other* potential network managers and their sockets to prevent future conflicts
# We will then explicitly unmask and enable the chosen one.
echo -e "${CYAN}Masking all common network managers and their sockets to prevent future conflicts...${NC}"
backup_file "/etc/systemd/system/NetworkManager.service" 
backup_file "/etc/systemd/system/NetworkManager.socket"
backup_file "/etc/systemd/system/dhcpcd.service"
backup_file "/etc/systemd/system/dhcpcd.socket"
backup_file "/etc/systemd/system/systemd-networkd.service"
backup_file "/etc/systemd/system/systemd-networkd.socket"

# Mask all
systemctl mask NetworkManager.service || true; systemctl mask NetworkManager.socket || true
systemctl mask dhcpcd.service || true; systemctl mask dhcpcd.socket || true
systemctl mask systemd-networkd.service || true; systemctl mask systemd-networkd.socket || true

# Explicitly unmask and enable the chosen network manager
echo -e "${CYAN}Unmasking and enabling the detected manager for next boot: ${DETECTED_MANAGER_FOR_SYSTEM}...${NC}"
case "$DETECTED_MANAGER_FOR_SYSTEM" | tr -d '\n\r' in # Strip newlines if present from detection
    "NetworkManager")
        systemctl unmask NetworkManager.service || true
        systemctl unmask NetworkManager.socket || true
        systemctl enable NetworkManager.service
        # systemctl start NetworkManager.service # No need to start here, systemd will do it on boot
        ;;
    "systemd-networkd")
        systemctl unmask systemd-networkd.service || true
        systemctl unmask systemd-networkd.socket || true
        systemctl enable systemd-networkd.service
        # systemctl start systemd-networkd.service # No need to start here
        ;;
    "dhcpcd")
        systemctl unmask dhcpcd.service || true
        systemctl unmask dhcpcd.socket || true
        systemctl enable dhcpcd.service
        # systemctl start dhcpcd.service # No need to start here
        ;;
    *) # If no manager was detected, or a problem, fallback to NetworkManager as the robust default
        echo -e "${YELLOW}No recognized network manager detected. Defaulting to NetworkManager (installing if needed and enabling for next boot)...${NC}"
        systemctl unmask NetworkManager.service || true
        systemctl unmask NetworkManager.socket || true
        systemctl enable NetworkManager.service
        # systemctl start NetworkManager.service # No need to start here
        DETECTED_MANAGER_FOR_SYSTEM="NetworkManager" # Set for config file write
        ;;
esac

echo -e "${GREEN}Primary network manager for system set to: ${DETECTED_MANAGER_FOR_SYSTEM}${NC}"


# --- 3. Create Application-Specific Configuration and Data Directories ---
echo -e "${CYAN}Creating application directories...${NC}"
mkdir -p "$APP_CONFIG_DIR"/{assets,lang}
mkdir -p "$APP_DEVICE_DATA_DIR"
chmod 700 "$APP_DEVICE_DATA_DIR" 

# --- 4. Stop the current pifigo service (if running) before copying new binary ---
echo -e "${CYAN}Stopping $APP_NAME service if it's currently running...${NC}"
if systemctl is-active --quiet "$APP_NAME".service; then
    systemctl stop "$APP_NAME".service
    echo -e "${GREEN}$APP_NAME service stopped.${NC}"
else
    echo -e "${GREEN}$APP_NAME service not running, no need to stop.${NC}"
fi

# --- 5. Copy Compiled Go Binary ---
echo -e "${CYAN}Copying compiled Go binary '$SOURCE_BINARY_NAME' to '$APP_BINARY_DEST'...${NC}"
cp "$PROJECT_ROOT/$SOURCE_BINARY_NAME" "$APP_BINARY_DEST" 
chmod +x "$APP_BINARY_DEST"

# --- 6. Copy App-Specific Configuration Files and Assets ---
echo -e "${CYAN}Copying app-specific config and asset files...${NC}"
backup_file "$APP_CONFIG_DIR/config.toml" 
cp "$PROJECT_ROOT/config.toml" "$APP_CONFIG_DIR/config.toml" # Copy the base config.toml
cp -r "$PROJECT_ROOT/lang/." "$APP_LANG_DIR/"
cp -r "$APP_ASSETS_SOURCE_DIR/." "$APP_ASSETS_DEST_DIR/" 

# --- Write detected network manager type to config.toml (Robustly) ---
CONFIG_FILE_TO_UPDATE="$APP_CONFIG_DIR/config.toml"
RUNTIME_SECTION_HEADER="[runtime]"
RUNTIME_KEY_VALUE="network_manager_type = \"$DETECTED_MANAGER_FOR_SYSTEM\""

echo -e "${CYAN}Updating detected network manager type in $CONFIG_FILE_TO_UPDATE...${NC}"

# 1. Check if the [runtime] section exists
if ! grep -q "^\[runtime\]" "$CONFIG_FILE_TO_UPDATE"; then # Use literal grep for header
    # [runtime] section does NOT exist, append it and the key
    echo -e "\n${RUNTIME_SECTION_HEADER}\n${RUNTIME_KEY_VALUE}" >> "$CONFIG_FILE_TO_UPDATE"
    echo -e "${GREEN}Added new [runtime] section with '$DETECTED_MANAGER_FOR_SYSTEM'.${NC}"
else
    # [runtime] section exists. Now check if network_manager_type key exists within it.
    # Check if network_manager_type key exists (anywhere in file)
    if grep -qE '^network_manager_type\s*=' "$CONFIG_FILE_TO_UPDATE"; then
        CURRENT_CONFIG_MANAGER_TYPE=$(grep -E '^network_manager_type\s*=' "$CONFIG_FILE_TO_UPDATE" | cut -d'=' -f2 | tr -d '[:space:]"')
        
        if [ "$CURRENT_CONFIG_MANAGER_TYPE" != "\"$DETECTED_MANAGER_FOR_SYSTEM\"" ]; then # Compare with quoted string
            echo -e "${YELLOW}Warning: Mismatch detected! Configured network_manager_type in $CONFIG_FILE_TO_UPDATE ('$CURRENT_CONFIG_MANAGER_TYPE') does not match detected ('$DETECTED_MANAGER_FOR_SYSTEM').${NC}"
            echo -e "${CYAN}Updating $CONFIG_FILE_TO_UPDATE to use detected type.${NC}"
            sed -i "/^network_manager_type\s*=/c\\$RUNTIME_KEY_VALUE" "$CONFIG_FILE_TO_UPDATE"
        else
            echo -e "${GREEN}Configured network_manager_type already matches detected type ('$DETECTED_MANAGER_FOR_SYSTEM'). No change needed.${NC}"
        fi
    else
        # [runtime] section exists, but network_manager_type key does not. Append it to the section.
        echo -e "${GREEN}Adding 'network_manager_type' key to existing [runtime] section in $CONFIG_FILE_TO_UPDATE.${NC}"
        # Use sed to append directly after the [runtime] header
        sed -i "/^\[runtime\]/a\\$RUNTIME_KEY_VALUE" "$CONFIG_FILE_TO_UPDATE"
    fi
fi


# --- 7. Configure systemd Service for pifigo ---
echo -e "${CYAN}Configuring systemd service for $APP_NAME...${NC}"
backup_file "$APP_SYSTEMD_SERVICE_PATH" # Backup pifigo's systemd unit file
cat <<EOF > "$APP_SYSTEMD_SERVICE_PATH"
[Unit]
Description=Headless Wi-Fi Setup Service
After=network-pre.target # Ensure basic network interfaces are up
Wants=network-pre.target
# This service needs to start before NetworkManager (or other managers) fully takes over wlan0
# as it might temporarily stop the manager for AP setup.
Before=NetworkManager.service dhcpcd.service systemd-networkd.service

[Service]
Type=simple
ExecStart=$APP_BINARY_DEST
WorkingDirectory=$APP_CONFIG_DIR
StandardOutput=inherit
StandardError=inherit
Restart=on-failure # Restart if it crashes during initial AP setup
RestartSec=5
# Removed: User=root (already implied by systemd default if not specified and binary runs as root)
# Removed: Group=root 

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$APP_NAME".service # Enable pifigo for initial boot

# --- INITIAL SERVICE STATE FOR AP MODE ---
# Ensure services that conflict with AP are stopped and disabled by default.
# `unmask` first for services that might be masked by default (common for hostapd/dnsmasq)
echo -e "${CYAN}Ensuring clean state for AP mode (stopping conflicting services)...${NC}"
backup_file "/etc/hostapd/hostapd.conf" # Backup hostapd/dnsmasq config files
backup_file "/etc/dnsmasq.conf"
backup_file "/etc/dhcpcd.conf"
backup_file "/etc/default/hostapd"
backup_file "/etc/sysctl.conf"
backup_file "/etc/wpa_supplicant/wpa_supplicant.conf"
backup_iptables # Backup current iptables rules
backup_sudoers # Backup sudoers file

echo -e "${BLUE}Unmasking hostapd and dnsmasq services (if masked)...${NC}"
systemctl unmask hostapd.service || true
systemctl unmask dnsmasq.service || true

echo -e "${BLUE}Stopping hostapd and dnsmasq services...${NC}"
# --- REVISED: ONLY STOP, DO NOT DISABLE, hostapd and dnsmasq here ---
systemctl stop --now --no-block hostapd.service || true
systemctl stop --now --no-block dnsmasq.service || true

echo -e "${BLUE}Stopping primary network managers...${NC}"
# --- REVISED: ONLY STOP, DO NOT DISABLE, the primary network managers here ---
# Their enabled/masked state is handled earlier. pifigo Go code will manage
# disabling/enabling when it configures the AP or client.
systemctl stop --now --no-block NetworkManager.service || true 
systemctl stop --now --no-block dhcpcd.service || true
systemctl stop --now --no-block systemd-networkd.service || true
# --- END REVISED ---

# --- 9. Configure Sudoers for pifigo ---
echo -e "${CYAN}Configuring sudoers for $APP_NAME application...${NC}"
backup_sudoers # Sudoers backup done earlier.
echo "pi ALL=NOPASSWD: /usr/sbin/ifconfig, /usr/bin/systemctl *, /sbin/shutdown, /sbin/reboot, /usr/sbin/iwlist, /usr/bin/nmcli, /usr/sbin/ip" | EDITOR='tee -a' visudo

echo -e "${BLUE}--- $APP_NAME Installation Complete! ---${NC}"
echo -e "${GREEN}The system will now reboot. On reboot, it will start the Wi-Fi AP setup service.${NC}"
echo -e "${GREEN}Connect to '$AP_SSID' Wi-Fi (password: $AP_PASSWORD) and navigate to http://$DEVICE_HOSTNAME.local/.${NC}"
echo -e "${YELLOW}Rebooting in 5 seconds...${NC}"
sleep 5
reboot