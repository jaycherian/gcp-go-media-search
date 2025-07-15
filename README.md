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

This project provides a complete solution for processing video files, extracting intelligence using Google's Generative AI, persisting the metadata to BigQuery, and enabling powerful semantic search through a web interface.

## Architecture

The project is composed of three main parts:

1.  **Go Backend API (`/cmd/server`)**: A Go server built with Gin that exposes a REST API for file uploads and media search. It listens to Cloud Storage events via Pub/Sub to trigger media processing workflows.
2.  **React Frontend (`/web/ui`)**: A React and Material-UI single-page application for interacting with the backend, searching for media, and viewing results.
3.  **GCP Infrastructure (`/deployments/terraform`)**: Terraform scripts to provision all necessary GCP resources, including GCS buckets, Pub/Sub topics, and BigQuery datasets.

The core processing logic uses a **Chain of Responsibility (COR)** pattern, where each step (resizing, summary generation, scene extraction) is an atomic, testable unit of work.

## Prerequisites

Before you begin, ensure you have:

- x
- x
- x
- x

## Getting Started

### 1. Configure Your Environment

