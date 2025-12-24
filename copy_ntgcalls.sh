#!/usr/bin/env bash
set -Eeuo pipefail


BASE_URL="https://raw.githubusercontent.com/ashokshau/ntgcalls/master/examples/go"
API_URL="https://api.github.com/repos/ashokshau/ntgcalls/contents/examples/go"
MODULE_PATH="ashokshau/tgmusic"

NTGCALLS_DIR="src/vc/ntgcalls"
UBOT_DIR="src/vc/ubot"
UBOT_TYPES_DIR="src/vc/ubot/types"


if ! command -v jq &>/dev/null; then
    echo "q not found. Install jq first."
    exit 1
fi

if ! command -v curl &>/dev/null; then
    echo "curl not found. Install curl first."
    exit 1
fi

log() {
    echo "▶ $1"
}

log "Cleaning old directories..."

rm -rf "$NTGCALLS_DIR" "$UBOT_DIR"
mkdir -p "$NTGCALLS_DIR" "$UBOT_TYPES_DIR"

update_dir() {
    local remote_dir="$1"
    local local_dir="$2"

    log "Updating $local_dir"

    local response
    response=$(curl -fsSL "$API_URL/$remote_dir") || {
        echo "Failed to fetch $remote_dir"
        return 1
    }

    echo "$response" | jq -r '.[] | select(.type=="file") | .name' | while read -r file; do
        log "Downloading $remote_dir/$file"
        curl -fsSL \
            "$BASE_URL/$remote_dir/$file" \
            -o "$local_dir/$file"
    done
}

update_dir "ntgcalls" "$NTGCALLS_DIR"
update_dir "ubot" "$UBOT_DIR"
update_dir "ubot/types" "$UBOT_TYPES_DIR"

log "Source update completed"


log "Fixing Go import paths..."

find "$UBOT_DIR" -maxdepth 1 -type f -name "*.go" -print0 | xargs -0 sed -i \
    -e "s|\"../ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g" \
    -e "s|\"gotgcalls/ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g" \
    -e "s|\"gotgcalls/ubot/types\"|\"${MODULE_PATH}/src/vc/ubot/types\"|g"

find "$UBOT_TYPES_DIR" -type f -name "*.go" -print0 | xargs -0 sed -i \
    -e "s|\"../../ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g" \
    -e "s|\"gotgcalls/ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g"

log "Import paths fixed"
log "Done"
