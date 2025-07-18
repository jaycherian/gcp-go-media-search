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

module "iam" {
  source = "./modules/iam"
  app_service_account =  var.app_service_account
}

# module "low_res_resources" {
#   source = "./modules/low_res"
#   region = var.region
#   low_res_bucket = var.low_res_bucket
# }

# module "high_res_resources" {
#   source = "./modules/high_res"
#   region = var.region
#   high_res_bucket = var.high_res_bucket
# }

# module "bigquery" {
#   source = "./modules/bigquery"
#   region = var.region
# }


