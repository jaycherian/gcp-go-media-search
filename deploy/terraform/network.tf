# Copyright 2024 Google, LLC
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     https://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
 
locals {
  # Determine which subnet self_link to use based on whether a VPC name was provided.
  # If vpc_name is provided, use the data source for the existing subnet.
  # Otherwise, use the data source for the newly created subnet from the auto-mode VPC.
  subnet_self_link = var.vpc_name != "" ? data.google_compute_subnetwork.existing[0].self_link : data.google_compute_subnetwork.created[0].self_link
}

# Create a new VPC network in auto mode ONLY if a vpc_name is not provided.
resource "google_compute_network" "new_vpc" {
  count                   = var.vpc_name == "" ? 1 : 0
  project                 = local.project.id
  name                    = "media-search-vpc-auto"
  auto_create_subnetworks = true
  routing_mode            = "GLOBAL"
}

# Data source to get information about the newly created subnet in the specified region.
# This is only active when a new VPC is created.
data "google_compute_subnetwork" "created" {
  count      = var.vpc_name == "" ? 1 : 0
  project    = local.project.id
  name       = google_compute_network.new_vpc[0].name # In auto-mode, subnet name matches network name
  region     = var.region
  depends_on = [google_compute_network.new_vpc]
}

# Data source to get information about a pre-existing subnet.
# This is only active when a vpc_name and subnet_name are provided.
data "google_compute_subnetwork" "existing" {
  count   = var.vpc_name != "" && var.subnet_name != "" ? 1 : 0
  project = local.project.id
  name    = var.subnet_name
  region  = local.location.region
}
