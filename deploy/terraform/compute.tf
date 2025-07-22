// Copyright 2024 Google, LLC
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     https://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

# This is the server VM that hosts the application
resource "google_compute_instance" "server_vm" {
  name         = "media-search-server"
  machine_type = "n2d-standard-16"
  zone         = var.zone

  # Defines the boot disk for the instance.
  boot_disk {
    auto_delete = true
    device_name = "media-search-server-disk"
    mode = "READ_WRITE"
    initialize_params {
      image = "projects/ubuntu-os-cloud/global/images/family/ubuntu-2404-lts"
      size  = 100
      type  = "pd-ssd"
    }
  }

  network_interface {
    nic_type    = "GVNIC"
    queue_count = 0
    stack_type  = "IPV4_ONLY"
    subnetwork  = local.subnet_self_link
    access_config {
      network_tier = "PREMIUM"
    }
  }

  # Defines the service account and its API access scopes for the instance.
  service_account {
    email = local.service_accounts_default.compute
    scopes = ["cloud-platform"]
  }

  # Enables Shielded VM features to meet security constraints.
  shielded_instance_config {
    enable_secure_boot          = true
    enable_vtpm                 = true
    enable_integrity_monitoring = true
  }

  metadata = {
    enable-osconfig = "true"
    enable-oslogin  = "true"
    # Startup script to configure the instance on boot.
    # This script performs two main tasks:
    # 1. Updates system packages on every boot.
    # 2. Runs an initial setup script only on the first boot using a lock file.
    startup-script = <<-STARTUP_SCRIPT
      #!/bin/bash
      # This script runs on every VM start.

      # --- Part 1: Run on every boot ---
      # Update and upgrade packages.
      echo "--- Updating system packages... ---"
      apt update && apt upgrade -y && apt autoremove -y

      # --- Part 2: Run only on first boot ---
      # Define a lock file to check if the initial setup has been done.
      INIT_LOCK_FILE="/var/run/first-boot-setup-complete"
      if [ ! -f "$INIT_LOCK_FILE" ]; then
          # Use set -e to exit immediately if a command fails, ensuring the setup
          # is atomic. The lock file will only be created upon full success.
          set -e

          echo "--- First boot detected. Running full server setup. ---"

          echo "--> Running environment setup script (Go, Node, etc.)..."
          bash -c "$(curl -fsSL https://raw.githubusercontent.com/jaycherian/gcp-go-media-search/refs/tags/${var.release}/deploy/scripts/setup-server.sh)"

          echo "--> Installing application source code..."
          mkdir -p /opt/media-search
          curl -L https://github.com/jaycherian/gcp-go-media-search/archive/refs/tags/${var.release}.tar.gz | tar -xz --strip-components=1 -C /opt/media-search

          echo "--> Creating backend .env.local.toml configuration file..."
          mkdir -p /opt/media-search/backend/go/configs
          cat <<-TOML > /opt/media-search/backend/go/configs/.env.local.toml
            [application]
            google_project_id = "${local.project.id}"
            signer_service_account_email = "${google_service_account.media-search-sa.email}"

            [storage]
            high_res_input_bucket = "${var.high_res_bucket}"
            low_res_output_bucket = "${var.low_res_bucket}"
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
      (cd /opt/media-search && ./start_both.sh) > /var/log/media-search-app.log 2>&1 &
    STARTUP_SCRIPT
  }

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
    preemptible         = false
    provisioning_model  = "STANDARD"
  }

}
