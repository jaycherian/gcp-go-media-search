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

variable "type" {
  description = "The type of bucket to create. Should be one of 'high_res' or 'low_res'"
  type        = string
}

variable "bucket_name" {
  description = "The name of the bucket to create"
  type        = string
}

variable "region" {
  description = "The region to create the bucket in"
  type        = string
}

variable "app_service_account_email" {
  description = "The email address of the service account for the application"
  type        = string
}
