#!/bin/bash
# install.sh - Installs and configures the pifigo application for Wi-Fi setup.
# This version proactively handles network managers and provides robust config updates.
# It now creates backups of original system configuration files.

# # --- Start of Diagnostic Block ---
# echo "--- SCRIPT-INTERNAL DIAGNOSTICS ---"
# echo "BASH executable being used: $BASH"
# echo "BASH_VERSION being used: $BASH_VERSION"
# # Let's test the $'' syntax directly inside the script
# DIAG_GREEN=$'\e[32m'
# DIAG_NC=$'\e[0m'
# printf "${DIAG_GREEN}If this line is green, the interpreter is working correctly.${DIAG_NC}\n"
# echo "--- END OF DIAGNOSTICS ---"
# echo # Blank line for spacing

# Exit immediately if a command exits with a non-zero status.
set -e




# --- Application Version ---
## ROADMAP: have github release builder update this
VERSION="0.0.1" 

# --- Configuration Variables --- USER SHOULDN'T have to change any of these
APP_NAME="pifigo"

PROJECT_ROOT=$(pwd) 
RELEASE_DIR=$(pwd)/releases

APP_CONFIG_DIR="/etc/${APP_NAME}"             # /etc/pifigo (for config.toml, lang/*)
APP_ASSETS_DEST_DIR="${APP_CONFIG_DIR}/assets" # /etc/pifigo/assets (for optional external assets)
APP_LANG_DIR="${APP_CONFIG_DIR}/lang"         # /etc/pifigo/lang
APP_DEVICE_DATA_DIR="/var/lib/${APP_NAME}"    # /var/lib/pifigo (for device_id persistence)
APP_BACKUP_DIR="${APP_DEVICE_DATA_DIR}/.backup" # NEW: Backup directory

# Package Dependencies
APP_PKGS_array=("hostapd" "dnsmasq" "iptables-persistent" "avahi-daemon" "iproute2") # ARRAY of packages/dependencies needed by apt

APP_BINARY_DEST="/usr/local/bin/${APP_NAME}"

APP_SYSTEMD_SERVICE_PATH="/etc/systemd/system/${APP_NAME}.service"



## Initialize Log file
# --- Determine the correct user's home directory ---
if [[ -n "$SUDO_USER" ]]; then
    # Script is being run with sudo
    USER_HOME=$(getent passwd "$SUDO_USER" | cut -d: -f6)
else
    # Script is being run directly as root or a normal user
    USER_HOME="$HOME"
fi

# --- Define log directory using the determined home ---
LOG_DIR="${USER_HOME}/.pifigo"

# Create the directory. It will be owned by root if we used sudo.
mkdir -p "$LOG_DIR"

# --- Fix Permissions (Crucial!) ---
# If sudo was used, the log directory is owned by root.
# We must give ownership back to the original user so they can access it.
if [[ -n "$SUDO_USER" ]]; then
    chown -R "$SUDO_USER":"$SUDO_GID" "$LOG_DIR"
fi

# --- Define the Log File ---
export LOG_FILE="${LOG_DIR}/install-$(date +'%Y%m%d_%H%M%S').log"

# Create the log file or exit if you can't.
touch "$LOG_FILE" || { echo "FATAL: Cannot write to log file: $LOG_FILE"; exit 1; }

# Fix permissions on the new log file as well
if [[ -n "$SUDO_USER" ]]; then
    chown "$SUDO_USER":"$SUDO_GID" "$LOG_FILE"
fi
## END LOG INITIALIZATION


# Source directory for assets (where go:embed finds them during build)
APP_ASSETS_SOURCE_DIR="${PROJECT_ROOT}/cmd/${APP_NAME}/assets" 

# --- Extract Wi-Fi AP info from config.toml for the final message ---
AP_SSID=$(grep 'ap_ssid =' "${PROJECT_ROOT}/config.toml" | cut -d'"' -f2)
AP_PASSWORD=$(grep 'ap_password =' "${PROJECT_ROOT}/config.toml" | cut -d'"' -f2)
DEVICE_HOSTNAME=$(grep 'device_hostname =' "${PROJECT_ROOT}/config.toml" | cut -d'"' -f2)

#### Automated Variables

# System Architecture
if [ -n "$1" ]; then    # put here for testing purposes
    ARCH="$1"           #
else                    #
    ARCH=$(uname -m)
fi                      # end test leave line above



## FORMATTING VARIABLES
# --- ANSI Color Codes ---
RED=$'\e[31m'
GREEN=$'\e[32m'
BLUE=$'\e[94m'
YELLOW=$'\e[33m'
GOLD=$'\e[38;5;214m'
ORANGE=$'\e[38;5;208m'
MAGENTA=$'\e[35m'
CYAN=$'\e[36m'

# if using color with effects, use color first and then the effect. The color codes above reset the effect to none.
BOLD=$'\e[1m' # bold 
ITAL=$'\e[3m' # italics
ULINE=$'\e[4m' # underline
XOUT=$'\e[9m' # crossed out
REV=$'\e[7m' # reversed
NC=$'\e[0m' # No Color (resets to default)
# vanity ANSI logo

PIFIGO="${GOLD}${BOLD}pifi${BLUE}${BOLD}go${GOLD}${BOLD}⇅${NC}"

# Set color standards for script
INFO=${BLUE}
BOLD_INFO=${BOLD}${INFO}
WARN=${ORANGE}
BOLD_WARN=${BOLD}${WARN}
ERROR=${RED}
BOLD_ERROR=${BOLD}${ERROR}
SUCCESS=${GREEN}
BOLD_SUCCESS=${BOLD}${SUCCESS}


### FUNCTIONS



## Formatting  Functions
# run_indented: runs command and indents output
run_indented() {
  local indent=${INDENT:-"    "}
  { "$@" 2> >(sed "s/^/${indent}/g" >&2); } | sed "s/^/${indent}/g"
}

# stat_msg: A unified status message function.
# Usage: stat_msg <type> [arguments...]
# Types: error, warning, success, info
# formatting strings:
# __ plain text format
# _^ INFO color (e.g, CYAN)
#
# message types:
#   info    information messages 
#   warn    warning messages
#   error   error messages
#   success success messages
#   plain   plain formatted message
# 
# --- EXAMPLES OF HOW TO USE IT ---
# SET YOUR COLOR VARIABLES
# --- ANSI Color Codes ---
    # RED=$'\e[31m'
    # GREEN=$'\e[32m'
    # ORANGE=$'\e[38;5;208m'
    # CYAN=$'\e[36m'
    # # if using color with effects, use color first and then the effect. The color codes above reset the effect to none.
    # BOLD=$'\e[1m' # bold 
    # ITAL=$'\e[3m' # italics
    # ULINE=$'\e[4m' # underline
    # XOUT=$'\e[9m' # crossed out
    # REV=$'\e[7m' # reversed
    # NC=$'\e[0m' # No Color (resets to default)

    # # Set color standards for script
    # INFO=${CYAN}
    # BOLD_INFO=${INFO}${BOLD}
    # WARN=${ORANGE}
    # BOLD_WARN=${WARN}${BOLD}
    # ERROR=${RED}
    # BOLD_ERROR=${ERROR}${BOLD}
    # SUCCESS=${GREEN}
    # BOLD_SUCCESS=${SUCCESS}${BOLD}

## EXAMPLE USAGE
# # Get some data for our examples
# current_user=$(whoami)
# current_time=$(date +'%r')

# stat_msg success "Login detected."
# stat_msg info "User" "_^${current_user}" "__logged in at" "_^${current_time}"
# stat_msg warning "Your disk space is almost full."
# stat_msg error "Cannot find config file" "_^/etc/pifigo.conf" "__Aborting."
stat_msg() {

    # logging of all stat_msg's
    if [[ -n "$LOG_FILE" ]]; then
        # Create a clean, timestamped version of the message for the log
        # We strip the formatting prefixes like __ and _^ for a clean log
        local clean_message
        clean_message=$(echo "$*" | sed -E 's/(__|_M)//g')
        # Append the clean message to the log file
        printf "%s [%s]: %s\n" "$(date +'%Y-%m-%d %T')" "${1^^}" "$clean_message" >> "$LOG_FILE"
    fi

    printf "\r\e[2K" 
    # --- Validate that we have at least one argument (the type) ---
    if (( $# < 1 )); then
        printf "${BOLD_ERROR}[✖] Error:${NC}${ERROR} stat_msg: FATAL: Message type not provided.${NC}\n" >&2
        return 1
    fi
    local msg_type="$1"; shift

    # --- Set the color and label prefix based on the message type ---
    local prefix
    case "$msg_type" in
        err|error)
        prefix="${BOLD}${ERROR}[✖] Error:${NC}${ERROR}" # <-- Icon added
        ;;
        warn|warning)
        prefix="${BOLD_WARN}[!] Warning:${NC}${WARN}" # <-- Icon added
        ;;
        ok|success)
        prefix="${BOLD_SUCCESS}[✔] Success:${NC}${SUCCESS}" # <-- Icon added
        ;;
        info)
        prefix="${BOLD_INFO}[i] Info:${NC}${INFO}" # <-- Icon added
        ;;
        txt|text|plain)
        prefix="${NC}${BOLD}---${NC}${INFO}" # No icon for plain, looks clean
        ;;
        *)
        # Default case for invalid types
        printf "${BOLD_ERROR}[✖] Error:${NC}${ERROR} stat_msg: FATAL: Invalid message type '%s'.${NC}\n" "$msg_type" >&2
        return 1
        ;;
    esac

    # --- Process and print the rest of the message arguments ---
    printf "%s " "$prefix"
    if (( $# == 0 )); then
        printf "${NC}\n"
        return 0
    fi
    for arg in "$@"; do
        case "$arg" in
        __*) printf "%s" "${NC}${arg#__}";;
        _^*) printf "%s" "${INFO}${arg#_^}";;
        _+*) printf "%s" "${BOLD_INFO}${arg#_^}";;
        *) printf "%s" "${arg}";;
        esac
        printf " "
    done
    printf "${NC}\n"
}

## Logging Function
# usage:    log_exec echo "Some stuff to put in the log"
#           log_exec sudo apt-get -y install htop
log_exec() {
    # Announce what we are about to do using our standard message format.
    stat_msg info "Executing command:" "__$*"

    # Run the command and pipe its output (stdout and stderr)
    # into our line-by-line processing loop.
    "$@" 2>&1 | while IFS= read -r line; do
        # For each line of output, prepend a timestamp, a new tag,
        # an indent, and then append it to the log.
        printf "%s [CMD_OUT]   %s\n" "$(date +'%Y-%m-%d %T')" "$line" >> "$LOG_FILE"
    done
    return ${PIPESTATUS[0]}
}

# prompt_for_enter: waits for ENTER key to proceed 
prompt_for_enter() {
    local prompt="${1:-Press the ENTER key to continue...}"
    stat_msg warn "${prompt}"
    # Loop forever until the user presses the correct key.
    while true; do
        # Read a single, silent character.
        read -s -n 1 key
        # Check if the key was the Enter key (which results in an empty string).
        if [[ ${key} == "" ]]; then
            stat_msg success "✔ ENTER pressed. Continuing."
            # Exit the loop successfully.
            break
        else
            # Inform the user they pressed the wrong key and prompt again.
            # The -e allows interpretation of \n (newline).
            stat_msg warn "\nWrong key pressed:" "__${key}" "Please press ENTER to continue or" "_^CTRL-C$" "to exit."
        fi
    done
}


## NOTE: These spinner functions depend on the stat_msg function
## spinner functions:
# Functions to display an animated loading spinner/dots
# Usage: start_spinner "Your message here"
#        (run your long command)
#        stop_spinner $? "<success message>" "<failure message>"
_SPINNER_PID=0
_SPINNER_MESSAGE=""
_start_spinner_base() { #performs animation and core spinner behavior - called by start_spinner
    local color="$1"; shift
    local message="$*"

    if [[ "${_SPINNER_PID}" -ne 0 ]]; then return 1; fi
    _SPINNER_MESSAGE="$message"
    (
        local -a chars=("|" "/" "-" "\\")
        while true; do for char in "${chars[@]}"; do printf "${color}[%s] %s${NC}\r" "$char" "$message"; sleep 0.2; done; done
    ) &
    _SPINNER_PID=$!
    trap 'stop_spinner 1 "Task interrupted."' SIGINT SIGTERM
}

start_spinner() {
    local color message
    # Check if the first argument is a known message type
    case "$1" in
        info|warn|error) # Add any other types here
            color_var_name="${1^^}" # e.g., info -> INFO, warning -> WARN
            color="${!color_var_name:-$INFO}" # Use the color variable (e.g. $BLUE) or default
            shift # Remove the type from the arguments
            ;;
        *)
            # The first argument is not a type, so default to 'info' color
            color="$INFO"
            ;;
    esac
    message="$*"
    _start_spinner_base "$color" "$message"
}


## usage:"
## stop_spinner $? "<success message>" "<failure message>"
stop_spinner() {
    local exit_code=${1:-0}
    local success_message="${2:-}"
    local fail_message="${3:-}"

    if [[ "${_SPINNER_PID}" -eq 0 ]]; then return; fi

    # Add `|| true` to prevent set -e from exiting the script here.
    kill "${_SPINNER_PID}" &>/dev/null || true
    timeout 0.5 wait "${_SPINNER_PID}" &>/dev/null || true
    
    _SPINNER_PID=0
    trap - SIGINT SIGTERM

    # Now that the script doesn't exit prematurely, these lines will be reached.
    if [[ "$exit_code" -eq 0 ]]; then
        stat_msg success "${success_message:-${_SPINNER_MESSAGE} ... Done.}"
    else
        stat_msg error "${fail_message:-${_SPINNER_MESSAGE} ... Failed. (Exit code: ${exit_code})}"
    fi
}



### Functional Functions for install



# Sets BINARY_NAME
get_binaryName(){
        local arch_suffix=""
        case ${ARCH} in
            "armv6l")
                arch_suffix="_linux_armv6" ;; # Raspberry Pi 1, Zero W
            "armv7l")
                arch_suffix="_linux_armv7" ;; # Raspberry Pi 2/3/4, Zero 2 W [32-bit OS]
            "aarch64")
                arch_suffix="_linux_arm64" ;; # Raspberry Pi 3/4, Zero 2 W [64-bit OS]
            *)
                # stat_msg error "Unsupported Architecture: ${ARCH}"
                exit 1 ;;
        esac

    echo "${APP_NAME}_${VERSION}${arch_suffix}" # e.g., pifigo_0.0.1_linux_armv7
}

# install_packages: installs packages provided in an array
# @example with list of files provided in-line
#   install_packages "iptables" "vim"
# @example with variable array
#   packages_to_add=("htop" "vim" "net-tools")
#   install_packages "${packages_to_add[@]}"
install_packages(){
    start_spinner info "Updating system packages."
        apt-get -q=3 update -y > /dev/null 2>&1 #|| true # commenting out true b/c we want to know if this failed
    stop_spinner $? "System packages updated." "Package updates failed."

    # validate that there are packages to install
    if [ -z "$@" ]; then
        stat_msg warn "No packages specified for installation. This might be okay if intended..."
        prompt_for_enter
        return 0 # Return with an error code
    fi

    local packages_to_install=${@}

    start_spinner info "Installing dependencies:${NC} ${packages_to_install}"
        log_exec apt install -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" ${packages_to_install} || true
    stop_spinner $? "Dependencies Installed." "Dependencies failed to install."
}

# create_dir: creates directories
create_dir(){ # always use QUOTES ("") when passing variables to this function (example:  "${APP_CONFIG_DIR}"/{dir1,dir2} "${APP_BACKUP_DIR}")
    # 1. Guard clause: check if any arguments were passed
    if [ -z "$1" ]; then
        echo "Usage: create_dir <dir1> [dir2] ..." >&2
        return 1
    fi

    # 2. Loop through all the arguments provided
    for dir in "$@"; do
        # stat_msg info "Ensuring directory exists: '${dir}'"
        log_exec mkdir -pv "${dir}"
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
    if ! [[ "${mode}" =~ ^[0-7]{3}$ ]]; then
        stat_msg error "Invalid mode" "__${mode}" "The first value must be a three-digit octal permission mode (e.g., 700, 755)." >&2
        return 1
    fi

    # Assign all OTHER arguments (from the 2nd one onwards) to a new array.
    local dirs_to_modify=("${@:2}")

    # Loop through the new array of directories.
    for dir in "${dirs_to_modify[@]}"; do
        stat_msg info "Setting mode" "__${mode}" "on" "__${dir}"
        # Use -v (verbose) to see the changes.
        log_exec chmod -v "${mode}" "${dir}"
    done
}

# enable_network_service: # Enables a given network service manager.
# Argument $1: The name of the service (e.g., "NetworkManager")
enable_network_service() {
    local service_name="$1" # Using local variable for function

    if [[ -z "${service_name}" ]]; then
        echo "Error: No service name provided to enable_network_service function." >&2
        return 1 # Exit function with an error status
    fi

    echo "Enabling ${service_name} for next boot..."
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
# It's assumed that stat_msg, log_exec, start_spinner, and stop_spinner
# are all defined above this function in your script.

# A helper function for displaying usage instructions.
backup_usage() {
    stat_msg plain "Usage: backup [options]"
    stat_msg plain "  -p, --path <path>         Path to the file or directory to back up."
    stat_msg plain "  -e, --exec <command>      Command whose output will be backed up."
    stat_msg plain "  -o, --output <name>       (Optional) Basename for the backup file."
    stat_msg plain "  -d, --destination <dir>   (Optional) Directory to store the backup (Default: \${HOME}/.backup)"
}

# The fully refactored backup function
backup() {
    # --- 1. Initialize variables ---
    local source_path=""
    local source_command=""
    local output_basename=""
    local destination_dir=""
    local exit_code=0 # Used to track the final exit code of the operation

    # --- 2. The Argument Parsing Loop ---
    if [[ "$#" -eq 0 ]]; then
        backup_usage; return 1;
    fi
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --path|-p)        source_path="$2"; shift 2 ;;
            --exec|-e)        source_command="$2"; shift 2 ;;
            --output|-o)      output_basename="$2"; shift 2 ;;
            --destination|-d) destination_dir="$2"; shift 2 ;;
            --help|-h)        usage; return 0 ;;
            *)
                stat_msg error "Unknown option:" "__$1"
                backup_usage
                return 1 ;;
        esac
    done

    # --- 3. Validate and Set Defaults ---
    if [ -z "${source_path}" ] && [ -z "${source_command}" ]; then
        stat_msg error "You must provide either" "__--path" "or" "__--exec."
        return 1
    fi
    if [ -n "${source_path}" ] && [ -n "${source_command}" ]; then
        stat_msg error "You cannot use --path and --exec at the same time."
        return 1
    fi

    destination_dir="${destination_dir:-${HOME}/.backup}"

    if [ -z "${output_basename}" ]; then
        if [ -n "${source_path}" ]; then
            output_basename=$(basename "${source_path}")
        elif [ -n "${source_command}" ]; then
            output_basename=$(echo "${source_command}" | awk '{print $1}')
        fi
        stat_msg warn "Output name not provided. Using inferred name:" "__${output_basename}"
    fi

    # --- 4. The Business Logic ---
    stat_msg info "Preparing backup..."
    if ! mkdir -p "${destination_dir}"; then
        stat_msg error "Could not create destination directory:" "__${destination_dir}"
        return 1
    fi

    local timestamp
    timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_path="${destination_dir}/${output_basename}.${timestamp}.bak"

    # Announce the task, then start the spinner
    stat_msg info "Creating backup to:" "__${backup_path}"

    if [ -n "${source_path}" ]; then
        if [ -e "${source_path}" ]; then
            log_exec cp -rp "${source_path}" "${backup_path}"
            exit_code=$? 
        else
            stat_msg warn "Source path not found, skipping backup:" "__${source_path}"
        fi
    elif [ -n "${source_command}" ]; then
        local cmd_name=$(echo "${source_command}" | awk '{print $1}')
        if ! command -v "$cmd_name" &> /dev/null; then
            stat_msg error "Command not found, cannot create backup:" "__${cmd_name}"
            exit_code=1
        else
            # Announce the start and create a temporary log for errors
            local error_log; error_log=$(mktemp)
            printf "%s [CMD] Starting execution: %s\n" "$(date +'%Y-%m-%d %T')" "${source_command}" >> "$LOG_FILE"

            # Execute the command, sending errors to the temp file
            eval "${source_command}" > "${backup_path}" 2> "$error_log"
            exit_code=$?

            # Check for and process any errors with indentation
            if [[ -s "$error_log" ]]; then # If the error log has content...
                printf "%s [CMD_ERR] The command produced the following errors:\n" "$(date +'%Y-%m-%d %T')" >> "$LOG_FILE"
                # Read the temp file and log each line with a timestamp and indent
                while IFS= read -r line; do
                    printf "%s [CMD_ERR]   %s\n" "$(date +'%Y-%m-%d %T')" "$line" >> "$LOG_FILE"
                done < "$error_log"
            fi
            rm -f "$error_log" # Clean up

            # Announce completion
            printf "%s [CMD] Finished execution with exit code %s.\n" "$(date +'%Y-%m-%d %T')" "$exit_code" >> "$LOG_FILE"
        fi
    fi

    return ${exit_code}
}



# ### ROADMAP: REMOVE THESE BACKUP functions after updating script
# # --- Function to backup a file before modification ---
# backup_file() {
#     local file_path="$1"
#     local backup_name=$(basename "${file_path}")
#     local timestamp=$(date +"%Y%m%d_%H%M%S")
#     local backup_path="${APP_BACKUP_DIR}/${backup_name}.${timestamp}.bak"

#     if [ -f "${file_path}" ]; then
#         echo -e "${INFO}Backing up ${file_path} to ${backup_path}${NC}"
#         cp -p "${file_path}" "${backup_path}"
#     else
#         echo -e "${WARN}Note: ${file_path} not found, no backup created.${NC}"
#     fi
# }
# # --- Function to backup iptables rules ---
# backup_iptables() {
#     local timestamp=$(date +"%Y%m%d_%H%M%S")
#     local backup_path="${APP_BACKUP_DIR}/iptables_rules.v4.${timestamp}.bak"
#     echo -e "${CYAN}Backing up current iptables rules to ${backup_path}${NC}"
#     iptables-save > "${backup_path}"
# }
# # --- Function to backup sudoers ---
# backup_sudoers() {
#     local timestamp=$(date +"%Y%m%d_%H%M%S")
#     local backup_path="${APP_BACKUP_DIR}/sudoers.${timestamp}.bak"
#     echo -e "${CYAN}Backing up /etc/sudoers to ${backup_path}${NC}"
#     cp -p /etc/sudoers "${backup_path}"
# }

detect_netmgr() {

    local exit_code=0 # Default to success
    if systemctl is-active --quiet NetworkManager.service || systemctl is-enabled --quiet NetworkManager.service; then
        echo "NetworkManager"
    elif systemctl is-active --quiet systemd-networkd.service || systemctl is-enabled --quiet systemd-networkd.service; then
        echo "systemd-networkd"
    elif systemctl is-active --quiet dhcpcd.service || systemctl is-enabled --quiet dhcpcd.service; then
        echo "dhcpcd"
    else
        echo "unknown"
        exit_code=1
    fi
    
    return ${exit_code}
}

unmask_netmgr() {
    local netmgr="$1"
    case "${netmgr}" in  # Strip newlines if present from detection
    "NetworkManager")
        log_exec systemctl unmask NetworkManager.service || true
        log_exec systemctl unmask NetworkManager.socket || true
        log_exec systemctl enable NetworkManager.service
        # log_exec systemctl start NetworkManager.service # No need to start here, systemd will do it on boot
        ;;
    "systemd-networkd")
        log_exec systemctl unmask systemd-networkd.service || true
        log_exec systemctl unmask systemd-networkd.socket || true
        log_exec systemctl enable systemd-networkd.service
        # log_exec systemctl start systemd-networkd.service # No need to start here
        ;;
    "dhcpcd")
        log_exec systemctl unmask dhcpcd.service || true
        log_exec systemctl unmask dhcpcd.socket || true
        log_exec systemctl enable dhcpcd.service
        # log_exec systemctl start dhcpcd.service # No need to start here
        ;;
    *) # If no manager was detected, or a problem, fallback to NetworkManager as the robust default
        stat_msg warn "No recognized network manager detected. Defaulting to NetworkManager (installing if needed and enabling for next boot)..."
        log_exec systemctl unmask NetworkManager.service || true
        log_exec systemctl unmask NetworkManager.socket || true
        log_exec systemctl enable NetworkManager.service
        # log_exec systemctl start NetworkManager.service # No need to start here
        SYSTEM_NETWORK_MGR="NetworkManager" # Set for config file write
        ;;
esac

}

### ---- END FUNCTIONS ----

#############################################
## pifigo Installer
#############################################



# #### DO NOT USE THIS INSTALLER YET!!!!
prompt_for_enter "${BOLD}THIS INSTALLER IS NOT YET READY. Press ENTER to exit installation of ${PIFIGO}${NC}"
exit 0
# #####

stat_msg ok "good job mf"


# --- Check for root privileges ---
stat_msg info "Checking that installer is being run with root privileges."

    if [[ ${EUID} -ne 0 ]]; then
        #echo -e "${ERROR}This script must be run as root. Please use ${NC}${BOLD}sudo${ERROR}.${NC}"
        stat_msg err "Run this script as root:" "__sudo $0"
        exit 1
    fi
stat_msg ok "Running as root." "Proceeding"


## Start installation
prompt_for_enter "Press ENTER to continue installation"


## --- Create Application-Specific Configuration and Data Directories (and BACKUP dir) ---
stat_msg info "Creating application directories..."
    create_dir "${APP_CONFIG_DIR}"/{assets,lang} "${APP_DEVICE_DATA_DIR}" "${APP_BACKUP_DIR}"
    mod_dir 700 "${APP_DEVICE_DATA_DIR}" "${APP_BACKUP_DIR}"


## --- Confirm installation
# we may want to put more information here about how this can muck up your system/etc.
stat_msg info "_+--- Starting" "__${PIFOGO}" "_+Installation ---"

prompt_for_enter

### Detect OS Architecture (arm6, arm7, arch64, etc)
# --- Determine the correct binary name to install based on target Pi's architecture ---
stat_msg info "Determining correct binary for this system..."
    BINARY_NAME=$(get_binaryName) || { stat_msg error "Unsupported OS Architecture: ${ARCH}"; exit 1; }
stat_msg success "Detected binary:" "__${BINARY_NAME}"


## Verify the compiled binary exists
stat_msg info "Validating release is in location:${NC} ${RELEASE_DIR}/${BINARY_NAME}"
if [ ! -f "${RELEASE_DIR}/${BINARY_NAME}" ]; then
    echo -e "${BOLD_ERROR}Error:${ERROR} Required binary ${NC}${BOLD}'${BINARY_NAME}'${BOLD_ERROR} not found in ${NC}${BOLD}'${RELEASE_DIR}'" >&2
    echo -e "${ERROR}Please check that the value of the ${NC}VERSION${ERROR} variable is set properly in this ${NC}./install.sh${ERROR} script ${NC}" >&2
    echo -e "${ERROR}If you have built ${NC}${APP_NAME}${ERROR} locally: ensure you have run ${NC}./build-pifigo.sh${ERROR} in the ${NC}${APP_NAME}${ERROR} project root to compile binaries for this target${NC}" >&2
    exit 1
fi
stat_msg ok "Release is in location:${NC} ${RELEASE_DIR}/${BINARY_NAME}"


## --- DETECT NETWORK MANAGER in use ---
stat_msg info "Determining network management system in use."
    SYSTEM_NETWORK_MGR=$(detect_netmgr) || { stat_msg error "Unable to detect network management system." ; exit 1; }
stat_msg info "Network Manager detected:" "__${SYSTEM_NETWORK_MGR}"


## --- Backup Network Configurations
# create an array of file names to backup
stat_msg info "Backing up existing network configurations."
start_spinner info "Starting backup to ${APP_BACKUP_DIR}"
declare -a BACKUP_FILE_array=(\
    "/etc/systemd/system/NetworkManager.service"\
    "/etc/systemd/system/NetworkManager.socket"\
    "/etc/systemd/system/dhcpcd.service"\
    "/etc/systemd/system/dhcpcd.socket"\
    "/etc/systemd/system/systemd-networkd.service"\
    "/etc/systemd/system/systemd-networkd.socket"\
    )
for file in "${BACKUP_FILE_array}"; do 
    backup -p "${file}" -d "${APP_BACKUP_DIR}"
done
stop_spinner $? "Network Management files backed up to${NC} ${APP_BACKUP_DIR}" "Network Management file backups failed."

#### do not test locally beyond this point
stat_msg info "STOPPING HERE" 
exit 0
sleep 30
#########

## --- Mask ALL *other* potential network managers and their sockets to prevent future conflicts
# We will then explicitly unmask and enable the chosen one.
stat_msg info "Masking all common network managers and their sockets to prevent future conflicts."
start_spinner "Masking. . . "
    # Mask all
    log_exec systemctl mask NetworkManager.service || true
    log_exec mask NetworkManager.socket || true
    log_exec mask dhcpcd.service || true
    log_exec systemctl mask dhcpcd.socket || true
    log_exec mask systemd-networkd.service || true
    log_exec systemctl mask systemd-networkd.socket || true
stop_spinner $? "Masking Completed." "Masking Failed."

## --- Explicitly unmask and enable the chosen network manager
stat_msg info "Unmasking and enabling the detected manager for next boot:" "__${SYSTEM_NETWORK_MGR}"
start_spinner "Unmasking ${SYSTEM_NETWORK_MGR}"
    unmask_netmgr ${SYSTEM_NETWORK_MGR}
stop_spinner $? "Enabled ${SYSTEM_NETWORK_MGR}" "Failed to enable ${SYSTEM_NETWORK_MGR}"


## --- Update System & Install Core Dependencies ---

    # --- Pre-seed answers for iptables-persistent ---
    # This answers "Save current IPv4 rules?" and "Save current IPv6 rules?" with "true"
    # This should be run before 'apt install' for iptables
    echo "iptables-persistent iptables-persistent/autosave_v4 boolean true" | debconf-set-selections
    echo "iptables-persistent iptables-persistent/autosave_v6 boolean true" | debconf-set-selections

# ROADMAP - only install absolutely necessary packages - but they may all be necessary
install_packages "${APP_PKGS_array[@]}"










# --- 4. Stop the current pifigo service (if running) before copying new binary ---
echo -e "${CYAN}Stopping ${APP_NAME} service if it's currently running...${NC}"
if systemctl is-active --quiet "${APP_NAME}".service; then
    systemctl stop "${APP_NAME}".service
    echo -e "${GREEN}${APP_NAME} service stopped.${NC}"
else
    echo -e "${GREEN}${APP_NAME} service not running, no need to stop.${NC}"
fi

# --- 5. Copy Compiled Go Binary ---
echo -e "${CYAN}Copying compiled Go binary '${BINARY_NAME}' to '${APP_BINARY_DEST}'...${NC}"
cp "${RELEASE_DIR}/${BINARY_NAME}" "${APP_BINARY_DEST}" 
chmod +x "${APP_BINARY_DEST}"

# --- 6. Copy App-Specific Configuration Files and Assets ---
echo -e "${CYAN}Copying app-specific config and asset files...${NC}"
backup_file "${APP_CONFIG_DIR}/config.toml" 
cp "${PROJECT_ROOT}/config.toml" "${APP_CONFIG_DIR}/config.toml" # Copy the base config.toml
cp -r "${PROJECT_ROOT}/lang/." "${APP_LANG_DIR}/"
cp -r "${APP_ASSETS_SOURCE_DIR}/." "${APP_ASSETS_DEST_DIR}/" 

# --- Write detected network manager type to config.toml (Robustly) ---
CONFIG_FILE_TO_UPDATE="${APP_CONFIG_DIR}/config.toml"
RUNTIME_SECTION_HEADER="[runtime]"
RUNTIME_KEY_VALUE="network_manager_type = \"${SYSTEM_NETWORK_MGR}\""

echo -e "${CYAN}Updating detected network manager type in ${CONFIG_FILE_TO_UPDATE}...${NC}"

# 1. Check if the [runtime] section exists
if ! grep -q "^\[runtime\]" "${CONFIG_FILE_TO_UPDATE}"; then # Use literal grep for header
    # [runtime] section does NOT exist, append it and the key
    echo -e "\n${RUNTIME_SECTION_HEADER}\n${RUNTIME_KEY_VALUE}" >> "${CONFIG_FILE_TO_UPDATE}"
    echo -e "${GREEN}Added new [runtime] section with '${SYSTEM_NETWORK_MGR}'.${NC}"
else
    # [runtime] section exists. Now check if network_manager_type key exists within it.
    # Check if network_manager_type key exists (anywhere in file)
    if grep -qE '^network_manager_type\s*=' "${CONFIG_FILE_TO_UPDATE}"; then
        CURRENT_CONFIG_MANAGER_TYPE=$(grep -E '^network_manager_type\s*=' "${CONFIG_FILE_TO_UPDATE}" | cut -d'=' -f2 | tr -d '[:space:]"')
        
        if [ "${CURRENT_CONFIG_MANAGER_TYPE}" != "\"${SYSTEM_NETWORK_MGR}\"" ]; then # Compare with quoted string
            echo -e "${YELLOW}Warning: Mismatch detected! Configured network_manager_type in ${CONFIG_FILE_TO_UPDATE} ('${CURRENT_CONFIG_MANAGER_TYPE}') does not match detected ('${SYSTEM_NETWORK_MGR}').${NC}"
            echo -e "${CYAN}Updating ${CONFIG_FILE_TO_UPDATE} to use detected type.${NC}"
            sed -i "/^network_manager_type\s*=/c\\${RUNTIME_KEY_VALUE}" "${CONFIG_FILE_TO_UPDATE}"
        else
            echo -e "${GREEN}Configured network_manager_type already matches detected type ('${SYSTEM_NETWORK_MGR}'). No change needed.${NC}"
        fi
    else
        # [runtime] section exists, but network_manager_type key does not. Append it to the section.
        echo -e "${GREEN}Adding 'network_manager_type' key to existing [runtime] section in ${CONFIG_FILE_TO_UPDATE}.${NC}"
        # Use sed to append directly after the [runtime] header
        sed -i "/^\[runtime\]/a\\${RUNTIME_KEY_VALUE}" "${CONFIG_FILE_TO_UPDATE}"
    fi
fi


# --- 7. Configure systemd Service for pifigo ---
echo -e "${CYAN}Configuring systemd service for ${APP_NAME}...${NC}"
backup_file "${APP_SYSTEMD_SERVICE_PATH}" # Backup pifigo's systemd unit file
cat <<EOF > "${APP_SYSTEMD_SERVICE_PATH}"
[Unit]
Description=Headless Wi-Fi Setup Service
After=network-pre.target # Ensure basic network interfaces are up
Wants=network-pre.target
# This service needs to start before NetworkManager (or other managers) fully takes over wlan0
# as it might temporarily stop the manager for AP setup.
Before=NetworkManager.service dhcpcd.service systemd-networkd.service

[Service]
Type=simple
ExecStart=${APP_BINARY_DEST}
WorkingDirectory=${APP_CONFIG_DIR}
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
systemctl enable "${APP_NAME}".service # Enable pifigo for initial boot

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
echo -e "${CYAN}Configuring sudoers for ${APP_NAME} application...${NC}"
backup_sudoers # Sudoers backup done earlier.
echo "pi ALL=NOPASSWD: /usr/sbin/ifconfig, /usr/bin/systemctl *, /sbin/shutdown, /sbin/reboot, /usr/sbin/iwlist, /usr/bin/nmcli, /usr/sbin/ip" | EDITOR='tee -a' visudo

echo -e "${BLUE}--- ${APP_NAME} Installation Complete! ---${NC}"
echo -e "${GREEN}The system will now reboot. On reboot, it will start the Wi-Fi AP setup service.${NC}"
echo -e "${GREEN}Connect to '${AP_SSID}' Wi-Fi (password: ${AP_PASSWORD}) and navigate to http://${DEVICE_HOSTNAME}.local/.${NC}"
echo -e "${YELLOW}Rebooting in 5 seconds...${NC}"
sleep 5
reboot