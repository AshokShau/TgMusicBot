set -e


if ! command -v jq &> /dev/null
then
    echo "jq could not be found. Please install jq to continue."
    exit
fi

BASE_URL="https://raw.githubusercontent.com/ashokshau/ntgcalls/master/examples/go"
API_URL="https://api.github.com/repos/ashokshau/ntgcalls/contents/examples/go"
MODULE_PATH="ashokshau/tgmusic"

update_dir() {
    local remote_dir="$1"
    local local_dir="$2"

    echo "Updating $local_dir..."

    files=$(curl -s "$API_URL/$remote_dir" | jq -r '.[] | select(.type == "file") | .name')

    if [ -z "$files" ]; then
        echo "Error: Could not fetch file list for $remote_dir. Skipping."
        return
    fi

    for file in $files; do
        echo "Downloading $file..."
        wget -q -O "$local_dir/$file" "$BASE_URL/$remote_dir/$file"
    done
}

# Update ntgcalls, ubot, and ubot/types
update_dir "ntgcalls" "src/vc/ntgcalls"
update_dir "ubot" "src/vc/ubot"
update_dir "ubot/types" "src/vc/ubot/types"

echo "Update complete."

echo "Fixing Go import paths..."

find src/vc/ubot -maxdepth 1 -type f -name "*.go" -print0 | xargs -0 sed -i \
    -e "s|\"../ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g" \
    -e "s|\"gotgcalls/ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g" \
    -e "s|\"gotgcalls/ubot/types\"|\"${MODULE_PATH}/src/vc/ubot/types\"|g"

find src/vc/ubot/types -type f -name "*.go" -print0 | xargs -0 sed -i \
    -e "s|\"../../ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g" \
    -e "s|\"gotgcalls/ntgcalls\"|\"${MODULE_PATH}/src/vc/ntgcalls\"|g"

echo "Import paths fixed."
