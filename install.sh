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
CYAN='\033[1;36m'
# if using color with effects, use color first and then the effect. The color codes above reset the effect to none.
BOLD='\033[1m' # bold 
ITAL='\033[3m' # italics
ULINE='\033[4m' # underline
XOUT='\033[9m' # crossed out
REV='\033[7m' # reversed
NC='\033[0m' # No Color (resets to default)
# vanity ANSI logo
PIFIGO="${YELLOW}${BOLD}pifi${BLUE}${BOLD}go${YELLOW}${BOLD}\u21C5${NC}"

# # testing formating and read enter
# echo -e "${BOLD}Press ENTER to continue installation of${NC} ${PIFIGO}"
# read -s -n 1 key
# if [[ $key = "" ]]; then 
#     echo 'You pressed enter! . . . continuing'
#     continue
# else
#     echo "You pressed '$key'"
#     exit 1
# fi

# --- Application Version ---
## ROADMAP: have github release builder update this
VERSION="0.0.1" 

# --- Configuration Variables --- USER SHOULDN'T have to change any of these
APP_NAME="pifigo"
PROJECT_ROOT=$(pwd) 
RELEASE_DIR=$(pwd)/releases

APP_PKGS_array=("hostapd" "dnsmasq" "iptables-persistent" "avahi-daemon" "iproute2") # ARRAY of packages/dependencies needed by apt

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



### FUNCTIONS

## Formatting/Interactive Functions
# run_indented: runs command and indents output
run_indented() {
  local indent=${INDENT:-"    "}
  { "$@" 2> >(sed "s/^/$indent/g" >&2); } | sed "s/^/$indent/g"
}


# prompt_for_enter: waits for ENTER key to proceed 
prompt_for_enter() {
    local prompt="${1:-Press the ENTER key to continue...}"
    echo -e "$prompt"
    # Loop forever until the user presses the correct key.
    while true; do
        # Read a single, silent character.
        read -s -n 1 key
        # Check if the key was the Enter key (which results in an empty string).
        if [[ $key == "" ]]; then
            echo "✔ ENTER pressed. Continuing."
            # Exit the loop successfully.
            break
        else
            # Inform the user they pressed the wrong key and prompt again.
            # The -e allows interpretation of \n (newline).
            echo -e "\n${RED}${BOLD}Wrong key pressed: '$key'.\nPlease press ENTER to continue or CTRL-C to exit."
        fi
    done
}

check_rootuser(){
    if [[ $EUID -ne 0 ]]; then
        echo -e "${RED}This script must be run as root. Please use ${NC}${BOLD}sudo${RED}.${NC}"
        exit 1
    fi
}

# install_packages: installs packages provided in an array
# @example with list of files provided in-line
#   install_packages "iptables" "vim"
# @example with variable array
#   packages_to_add=("htop" "vim" "net-tools")
#   install_packages "${packages_to_add[@]}"
install_packages(){
    # validate that there are packages to install
    if [ -z "$@" ]; then
        echo "${YELLOW}${BOLD}INFO:${NC} No packages specified for installation."
        prompt_for_enter "${YELLOW}${BOLD}Press ENTER to proceed. . . "
        return 0 # Return with an error code
    fi

        local packages_to_install=${@}

    echo -e "${CYAN}Updating system packages. . ."
    run_indented apt-get update -y
    
    echo -e "\n${CYAN}Installing dependencies:${NC} $packages_to_install . . ."
    run_indented apt upgrade -y

    ## ROADMAP: only install what is needed (move this to after determining network manager service)
    run_indented apt install -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" $packages_to_install || true
}

# create_dir: creates directories
create_dir(){ # always use QUOTES ("") when passing variables to this function (example:  "$APP_CONFIG_DIR"/{dir1,dir2} "$APP_BACKUP_DIR")
    # 1. Guard clause: check if any arguments were passed
    if [ -z "$@" ]; then
        echo "Usage: create_dir <dir1> [dir2] ..." >&2
        return 1
    fi

    # 2. Loop through all the arguments provided
    for dir in "$@"; do
        echo "Ensuring directory exists: '$dir'"
        run_indented mkdir -pv "$dir"
    done
}

# mod_dir: change mode of array list of directories 
# always use QUOTES ("") when passing variables to this function
mod_dir() {
    # check if we have enough arguments (at least a mode and one directory)
    if [ "$#" -lt 2 ]; then
        echo "ERROR: Incorrect usage of mod_dir." >$2
        echo "Usage: mod_dir <mode> <dir1> [dir2] ..." >&2
        return 1
    fi

    # Assign the first argument to mode and validate it
    local mode="$1"
    if ! [[ "$mode" =~ ^[0-7]{3}$ ]]; then
        echo "ERROR: Invalid mode '$mode'. The first argument must be a three-digit octal permission mode (e.g., 700, 755)." >&2
        return 1
    fi

    # Assign all OTHER arguments (from the 2nd one onwards) to a new array.
    local dirs_to_modify=("${@:2}")

    # Loop through the new array of directories.
    for dir in "${dirs_to_modify[@]}"; do
        echo -e "${CYAN}Setting mode${NC} '$mode' ${CYAN}on${NC} ${dir}"
        # Use -v (verbose) to see the changes.
        run_indented chmod -v "$mode" "$dir"
        run_indented ls -lh "$dir"
        echo -e "\n"
    done
}

# enable_network_service: # Enables a given network service manager.
# Argument $1: The name of the service (e.g., "NetworkManager")
enable_network_service() {
    local service_name="$1" # Using local variable for function

    if [[ -z "$service_name" ]]; then
        echo "Error: No service name provided to enable_network_service function." >&2
        return 1 # Exit function with an error status
    fi

    echo "Enabling $service_name for next boot..."
    systemctl unmask "${service_name}.service" || true
    systemctl unmask "${service_name}.socket" || true
    systemctl enable "${service_name}.service"
}

#--------------------------------------------------------------------------
# @description Creates a timestamped backup of a given file, directory, or command output.
#
# @example
#   backup --path /etc/hosts -o hosts                          # Backs up to default dir
#   backup --path /etc/hosts -o hosts -d /mnt/nas/backups       # Backs up to a custom dir
#   backup --exec "dmesg" -o "dmesg-log"                        # Backs up command to default dir
#--------------------------------------------------------------------------
backup() {
    # --- 1. Initialize variables ---
    local source_path=""
    local source_command=""
    local output_basename=""
    local destination_dir="" # NEW: Variable for the destination directory

    # --- 2. The Argument Parsing Loop ---
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --path|-p)
                source_path="$2"; shift 2 ;;
            --exec|-e)
                source_command="$2"; shift 2 ;;
            --output|-o)
                output_basename="$2"; shift 2 ;;
            # NEW: Flag for specifying the destination directory
            --destination|-d)
                destination_dir="$2"; shift 2 ;;
            --help|-h)
                echo "Usage: backup [options]"
                echo "  -p, --path <path>          Path to the file or directory to back up."
                echo "  -e, --exec <command>         Command whose output will be backed up."
                echo "  -o, --output <name>      (Optional) Basename for the backup file."
                echo "  -d, --destination <dir>  (Optional) Directory to store the backup."
                echo "                           (Default: \$HOME/backups)"
                return 0 ;;
            *)
                echo "ERROR: Unknown option: $1" >&2; return 1 ;;
        esac
    done

    # --- 3. Validate and Set Defaults ---
    # (Source validation logic is unchanged)
    if [ -z "$source_path" ] && [ -z "$source_command" ]; then
        echo "ERROR: You must provide either --path or --exec." >&2; return 1
    fi
    if [ -n "$source_path" ] && [ -n "$source_command" ]; then
        echo "ERROR: You cannot use --path and --exec at the same time." >&2; return 1
    fi

    # Set default destination if not provided
    destination_dir="${destination_dir:-$HOME/backups}"

    # Infer output name if it was not provided
    if [ -z "$output_basename" ]; then
        if [ -n "$source_path" ]; then
            output_basename=$(basename "$source_path")
        elif [ -n "$source_command" ]; then
            output_basename=$(echo "$source_command" | awk '{print $1}')
        fi
        echo -e "${YELLOW}INFO: --output name not provided. Inferred name: '$output_basename'${NC}"
    fi

    # --- 4. The Business Logic ---
    # Ensure the destination directory exists before we try to use it.
    # The -p flag creates parent directories as needed.
    if ! mkdir -p "$destination_dir"; then
        echo "ERROR: Could not create destination directory: $destination_dir" >&2
        return 1
    fi

    local timestamp=$(date +"%Y%m%d_%H%M%S")
  
    local backup_path="${destination_dir}/${output_basename}.${timestamp}.bak"

    echo -e "${CYAN}Creating backup: ${NC}$backup_path"

    if [ -n "$source_path" ]; then
        if [ -e "$source_path" ]; then
            run_indented cp -rp "$source_path" "$backup_path"
        else
            echo -e "${YELLOW}Warning: Source path not found, skipping backup:${NC} $source_path"
        fi
    elif [ -n "$source_command" ]; then
        if ! command -v "$(echo "$source_command" | awk '{print $1}')" &> /dev/null; then
             echo -e "${YELLOW}Warning: Command not found, skipping backup:${NC} $source_command"
             return 1
        fi
        run_indented eval "$source_command" > "$backup_path"
    fi
}

### ROADMAP: REMOVE THESE BACKUP functions after updating script
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


### ---- END FUNCTIONS ----

## MAIN Installer

#### DO NOT USE THIS INSTALLER YET!!!!
prompt_for_enter "${BOLD} THIS INSTALLER IS NOT YET READY. Press ENTER to exit installation of ${PIFIGO}${NC}"
exit 1
#####


# --- Check for root privileges ---
check_rootuser

## Pre-check 
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
if [ ! -f "$RELEASE_DIR/$SOURCE_BINARY_NAME" ]; then
    echo -e "${RED}Error: Required binary '$SOURCE_BINARY_NAME' not found in '$RELEASE_DIR'.${NC}" >&2
    echo -e "${RED}Please check that the value of the VERSION variable is set properly in this './install.sh' script ${NC}" >&2
    echo -e "${RED}If you have built $APP_NAME locally: ensure you have run './build-pifigo.sh' in the $APP_NAME project root to compile binaries for this target.${NC}" >&2
    exit 1
fi
echo -e "${GREEN}Detected architecture: ${BOLD}$(uname -m)${GREEN}. Will install ${NC}${BOLD}'$SOURCE_BINARY_NAME'${NC}\n\n"


# Start installation
prompt_for_enter "${BOLD} Press ENTER to continue installation of ${PIFIGO}${NC}. . .\n"

sleep 2
echo -e "${BLUE}${BOLD}--- Starting $APP_NAME Installation ---${NC}"
sleep 2

# --- 1. DETECT NETWORK MANAGER in use ---
# ROADMAP: Make this a function that outputs the DETECTED Network Manager
# This section configures the *system's default network manager* for the next boot.
# It does NOT stop actively running services during installation to avoid disruption.
echo -e "${CYAN}Ensuring system's network manager is correctly configured for next boot...${NC}"
## ROADMAP - run this first to see which packages we need to install if we don't need all of them
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


# --- 2. Backup Network Configurations
# ROADMAP - use the new backup function, create an array and go through array to backup
backup_file "/etc/systemd/system/NetworkManager.service" 
backup_file "/etc/systemd/system/NetworkManager.socket"
backup_file "/etc/systemd/system/dhcpcd.service"
backup_file "/etc/systemd/system/dhcpcd.socket"
backup_file "/etc/systemd/system/systemd-networkd.service"
backup_file "/etc/systemd/system/systemd-networkd.socket"

# --- 2. Mask ALL *other* potential network managers and their sockets to prevent future conflicts
# We will then explicitly unmask and enable the chosen one.
echo -e "${CYAN}Masking all common network managers and their sockets to prevent future conflicts...${NC}"

# Mask all
systemctl mask NetworkManager.service || true; systemctl mask NetworkManager.socket || true
systemctl mask dhcpcd.service || true; systemctl mask dhcpcd.socket || true
systemctl mask systemd-networkd.service || true; systemctl mask systemd-networkd.socket || true

# Explicitly unmask and enable the chosen network manager
echo -e "${CYAN}Unmasking and enabling the detected manager for next boot:${NC} ${DETECTED_MANAGER_FOR_SYSTEM}...${NC}"

case "$DETECTED_MANAGER_FOR_SYSTEM" | tr -d '\n\r' in  # Strip newlines if present from detection
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

echo -e "${GREEN}Primary network manager for system set to:${NC}${BOLD} ${DETECTED_MANAGER_FOR_SYSTEM}${NC}"

# --- 1. Update System & Install Core Dependencies ---

# --- Pre-seed answers for iptables-persistent ---
# This answers "Save current IPv4 rules?" and "Save current IPv6 rules?" with "true"
# This should be run before 'apt install'
echo "iptables-persistent iptables-persistent/autosave_v4 boolean true" | debconf-set-selections
echo "iptables-persistent iptables-persistent/autosave_v6 boolean true" | debconf-set-selections

# ROADMAP - only install absolutely necesseary packages
install_packages "${APP_PKGS_array[@]}"



# --- 2. Create Application-Specific Configuration and Data Directories (and BACKUP dir) ---
echo -e "${CYAN}Creating application directories...${NC}"
create_dir "$APP_CONFIG_DIR"/{assets,lang} "$APP_DEVICE_DATA_DIR" "$APP_BACKUP_DIR"
mod_dir 700 "$APP_DEVICE_DATA_DIR" "$APP_BACKUP_DIR"




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
cp "$RELEASE_DIR/$SOURCE_BINARY_NAME" "$APP_BINARY_DEST" 
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