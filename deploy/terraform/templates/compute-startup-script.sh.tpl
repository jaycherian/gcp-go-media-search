#!/bin/bash
# This script runs on every VM start.

# --- Part 1: Run on every boot ---
# Update and upgrade packages.
echo "--- Updating system packages... ---"
apt update && apt upgrade -y && apt autoremove -y

# --- Part 2: Run only on first boot ---
echo "--- Checking if first boot initialization is needed... ---"
# Define a lock file to check if the initial setup has been done.
INIT_LOCK_FILE="/var/run/first-boot-setup-complete"
if [ ! -f "$INIT_LOCK_FILE" ]; then
    # Use set -e to exit immediately if a command fails, ensuring the setup
    # is atomic. The lock file will only be created upon full success.
    set -e

    echo "--- First boot detected. Running full server setup. ---"

    echo "--> Running environment setup script (Go, Node, etc.)..."
    bash -c "$(curl -fsSL https://raw.githubusercontent.com/jaycherian/gcp-go-media-search/refs/tags/${release}/deploy/scripts/setup-server.sh)"

    echo "--> Installing application source code..."
    mkdir -p /opt/media-search
    curl -L https://github.com/jaycherian/gcp-go-media-search/archive/refs/tags/${release}.tar.gz | tar -xz --strip-components=1 -C /opt/media-search

    echo "--> Creating backend .env.local.toml configuration file..."
    mkdir -p /opt/media-search/backend/go/configs
    cat <<-'TOML' > /opt/media-search/backend/go/configs/.env.local.toml
      [application]
      google_project_id = "${project_id}"
      signer_service_account_email = "${service_account_email}"

      [storage]
      high_res_input_bucket = "${high_res_bucket}"
      low_res_output_bucket = "${low_res_bucket}"
TOML

    echo "--- Initial setup completed successfully. Creating lock file. ---"
    touch "$INIT_LOCK_FILE"
fi

# --- Part 3: Run the servers every time (but we needed to get set up first) ---
echo "--- Starting application servers... ---"
# Source environment variables and paths needed for Go and Node/PNPM.
# This is necessary because this script runs as root in a non-interactive shell.
export HOME="/root"
export NVM_DIR="/root/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
export PATH=$PATH:/root/go/bin

# Run the start script in the background. The script itself will daemonize
# the backend and frontend processes.
(cd /opt/media-search && ./start_servers.sh) > /var/log/media-search-app.log 2>&1 &