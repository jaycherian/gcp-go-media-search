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

# Data source to get the latest Ubuntu 24.04 LTS image.
# This ensures we always use the most up-to-date image from the family
# without hardcoding a specific version.
data "google_compute_image" "latest_ubuntu_2404" {
  project = "ubuntu-os-cloud"
  family  = "ubuntu-2404-lts-amd64"
}

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
      image = data.google_compute_image.latest_ubuntu_2404.self_link
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
    # We use a template file to keep the main Terraform file clean and pass
    # variables to the script.
    startup-script = templatefile("${path.module}/templates/compute-startup-script.sh.tpl", {
      release               = var.release
      project_id            = local.project.id
      region                = local.location.region
      service_account_email = google_service_account.media-search-sa.email
      high_res_bucket       = var.high_res_bucket
      low_res_bucket        = var.low_res_bucket
      service_account_key   = google_service_account_key.media-search-sa-key.private_key
      key_location          = "/opt/media-search/backend/go/configs"
    })
  }

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
    preemptible         = false
    provisioning_model  = "STANDARD"
  }

}
