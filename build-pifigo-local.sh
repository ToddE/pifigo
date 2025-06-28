#!/bin/bash
# build-pifigo.sh - Builds the pifigo executable for various Raspberry Pi architectures.
# Now outputs compiled binaries to a 'releases/' directory.

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

APP_NAME="pifigo"
SOURCE_PATH="./cmd/${APP_NAME}/" # Path to the main package within this repo
OUTPUT_DIR="releases"             # NEW: Define the output directory name

# --- Manual Version Number ---
# Update this string whenever you build a new version.
VERSION_CLEAN="0.0.1" 

echo -e "${BLUE}--- Starting ${APP_NAME} Build (Version: ${VERSION_CLEAN}) ---${NC}"

# --- Ensure Go modules are tidy and up-to-date ---
echo -e "${CYAN}Running 'go mod tidy' to synchronize go.mod and go.sum...${NC}"
go mod tidy
echo -e "${GREEN}'go mod tidy' complete.${NC}"

# --- Create the output directory ---
echo -e "${CYAN}Creating output directory: ${OUTPUT_DIR}/${NC}"
mkdir -p "$OUTPUT_DIR"

# Function to build for a specific target
build_target() {
    local arch_suffix="$1" # e.g., _linux_armv6
    local goarm_val="$2"   # e.g., 6 or 7 (or empty for arm64)
    local goarch_val="$3"  # e.g., arm or arm64

    OUTPUT_NAME="${APP_NAME}_${VERSION_CLEAN}${arch_suffix}"
    OUTPUT_PATH="${OUTPUT_DIR}/${OUTPUT_NAME}" # NEW: Full path to output binary
    
    echo -e "${CYAN}  Building for linux/${goarch_val}/${goarm_val} -> ${OUTPUT_PATH}...${NC}"

    if [ -z "$goarm_val" ]; then # For arm64, GOARM is not set
        GOOS=linux GOARCH="$goarch_val" CGO_ENABLED=0 go build -o "$OUTPUT_PATH" "$SOURCE_PATH"
    else
        GOOS=linux GOARCH="$goarch_val" GOARM="$goarm_val" CGO_ENABLED=0 go build -o "$OUTPUT_PATH" "$SOURCE_PATH"
    fi
    echo -e "${GREEN}    Output: ${OUTPUT_PATH}${NC}"
    ls -lh "${OUTPUT_PATH}"
}

# --- Build for all targets ---

# 1. Build for ARMv6 (Widest 32-bit compatibility)
build_target "_linux_armv6" "6" "arm"

# 2. Build for ARMv7 (Optimized 32-bit for newer Pis)
build_target "_linux_armv7" "7" "arm"

# 3. Build for ARM64 (Native 64-bit)
build_target "_linux_arm64" "" "arm64"

# 4. Build for PC amd64
build_target "_linux_amd64" "" "amd64" # Suffix, GOARM="", GOARCH="amd64"

echo -e "${BLUE}All ${APP_NAME} builds complete!${NC}"
ls -lh "${OUTPUT_DIR}"
