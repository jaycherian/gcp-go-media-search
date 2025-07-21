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

locals {
  project = {
    id      = var.project_id
    name    = data.google_project.project.name
    number  = data.google_project.project.number
  }
  location = {
    region = var.region
    zone = var.zone
  }
  _services = [
    "aiplatform",
    "compute",
    "pubsub",
    "storage"
  ]
  service_accounts_default = {
    compute      = data.google_compute_default_service_account.default.email
    storage      = data.google_storage_project_service_account.default.email_address
  }
  service_account_cloud_services = (
    "${local.project.number}@cloudservices.gserviceaccount.com"
  )
  service_accounts_services_api = {
    for s in local._services : s => "${s}.googleapis.com"
  }
}

data "google_project" "project" {
  project_id = var.project_id
}

data "google_storage_project_service_account" "default" {}

data "google_compute_default_service_account" "default" {}

resource "google_project_service_identity" "service_identity" {
  for_each   = local.service_accounts_services_api
  provider   = google-beta
  project    = local.project.id
  service    = each.value
}

resource "time_sleep" "wait_for_service_agent_readiness" {
  depends_on = [
    google_project_service_identity.service_identity,
  ]
  # SLO for IAM provisioning of Service Agents is 7min.
  create_duration = "420s"
}
