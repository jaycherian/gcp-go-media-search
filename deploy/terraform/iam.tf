// Copyright 2025 Google, LLC
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

###############################
##### 1) SERVICE ACCOUNTS #####
###############################

# Create a service account for the application
resource "google_service_account" "media-search-sa" {
  account_id   = var.app_service_account
  display_name = "TF - Media Search SA"
  project      = local.project.id
}

##########################################################
###### 2.a) MEMBER ROLES - Created Service Accounts ######
##########################################################

# Add roles to the created Media Search service account
module "member_roles_media_search" {
  source                  = "terraform-google-modules/iam/google//modules/member_iam"
  service_account_address = google_service_account.media-search-sa
  prefix                  = "serviceAccount"
  project_id              = local.project.id
  project_roles = [
    "roles/bigquery.admin",
    "roles/pubsub.admin",
    "roles/storage.admin",
    "roles/storage.objectAdmin",
    "roles/telemetry.metricsWriter",
    "roles/cloudtrace.admin",
    "roles/cloudtrace.agent",
    "roles/cloudtrace.user",
    "roles/monitoring.metricWriter",
    "roles/monitoring.metricsScopesAdmin",
  ]
}

##########################################################
###### 2.b) MEMBER ROLES - Default Service Accounts ######
##########################################################

# Add roles to the default Compute service account
module "member_roles_default_compute" {
  source                  = "terraform-google-modules/iam/google//modules/member_iam"
  service_account_address = local.service_accounts_default.compute
  prefix                  = "serviceAccount"
  project_id              = local.project.id
  project_roles = [
    "roles/bigquery.admin",
    "roles/pubsub.admin",
    "roles/storage.admin",
    "roles/storage.objectAdmin",
    "roles/telemetry.metricsWriter",
    "roles/cloudtrace.admin",
    "roles/cloudtrace.agent",
    "roles/cloudtrace.user",
    "roles/monitoring.metricWriter",
    "roles/monitoring.metricsScopesAdmin",
  ]

  depends_on = [google_project_service_identity.service_identity]

}
