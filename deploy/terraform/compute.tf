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
      image = "projects/debian-cloud/global/images/debian-12-bookworm-v20250709"
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
    scopes = ["cloud_platform"]
  }

  # Enables Shielded VM features to meet security constraints.
  shielded_instance_config {
    enable_secure_boot          = true
    enable_vtpm                 = true
    enable_integrity_monitoring = true
  }

  metadata = {
    enable-osconfig = "TRUE"
    enable-oslogin  = "true"
  }

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
    preemptible         = false
    provisioning_model  = "STANDARD"
  }
}
