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

variable "project_id" {
  type = string
}

variable "region" {
    type = string
    default = "us-central1"
}

variable "zone" {
    type = string
    default = "us-central1-b"
}

variable "vpc_name" {
    type = string
    description = "The name of the VPC network to use. If not provided, a new one will be created."
}

variable "subnet_name" {
    type = string
    description = "The name of the subnetwork to use. Required if vpc_name is provided."
}

variable "media_low_res_schema_name" {
    type = string
    default = "media_low_res_schema"
}

variable "low_res_bucket" {
    type = string
}

variable "high_res_bucket" {
    type = string
}

variable "app_service_account" {
    type = string
    default = "media-search-sa"
}

variable "release" {
  type        = string
  description = "The release tag for the setup scripts."
  default     = "release-0.0.4"
}