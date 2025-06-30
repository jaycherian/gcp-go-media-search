<!--
 Copyright 2024 Google, LLC
 
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 
     https://www.apache.org/licenses/LICENSE-2.0
 
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->
# Media Metadata Extraction & Search

[cite_start]This project provides a complete solution for processing video files, extracting intelligence using Google's Generative AI, persisting the metadata to BigQuery, and enabling powerful semantic search through a web interface. [cite: 564]

## Architecture

The project is composed of three main parts:

1.  **Go Backend API (`/cmd/server`)**: A Go server built with Gin that exposes a REST API for file uploads and media search. It listens to Cloud Storage events via Pub/Sub to trigger media processing workflows.
2.  **React Frontend (`/web/ui`)**: A React and Material-UI single-page application for interacting with the backend, searching for media, and viewing results.
3.  **GCP Infrastructure (`/deployments/terraform`)**: Terraform scripts to provision all necessary GCP resources, including GCS buckets, Pub/Sub topics, and BigQuery datasets.

[cite_start]The core processing logic uses a **Chain of Responsibility (COR)** pattern [cite: 565][cite_start], where each step (resizing, summary generation, scene extraction) is an atomic, testable unit of work. [cite: 566]

## Prerequisites

Before you begin, ensure you have the following installed:

* [cite_start]**Go**: Version 1.23 or later [cite: 1]
* [cite_start]**Node.js**: Version 20.x or later [cite: 529]
* [cite_start]**PNPM**: `npm install -g pnpm@8` [cite: 532]
* **FFmpeg**: Must be installed and available in your system's PATH.
* **Google Cloud SDK**: Authenticated to your GCP account.
* **Terraform**: For deploying infrastructure.

## Getting Started

### 1. Configure Your Environment

Copy the example Terraform variables file:

```shell
cp deployments/terraform/terraform.tfvars.example deployments/terraform/terraform.tfvars

Modify the tfvars files to suite your project, api and bucket names. 

deployments/terraform/terraform.tfvars and set your project_id and unique names for high_res_bucket and low_res_bucket. 

2. Deploy GCP Infrastructure
Shell

cd deploy/terraform
terraform init
terraform apply
3. Configure the Go Backend
The backend reads its configuration from TOML files. Create a local configuration for development:

Shell

cp backend/go/configs/.env.toml backend/go/configs/.env.local.toml

Edit 
backend/go/configs/.env.local.toml and fill in the values for your GCP project, API key, and the bucket names you defined in the previous step. 


4. Running the Application Locally
You will need two separate terminal windows.

Terminal 1: Start the Go Backend API

Shell

./start_backend.sh

Terminal 2: Start the React Frontend

./start_frontend.sh

Testing
To run the entire Go test suite:

Shell

# From the project root
go test ./...



You can also run the whole application in one go by using the following shell script
Run the application with: ./start.sh
