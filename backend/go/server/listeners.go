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

// Package main contains the logic for setting up and starting the Pub/Sub message listeners.
// These listeners are responsible for initiating backend processing workflows in response to events,
// such as new file uploads to Google Cloud Storage.
//
// Functions:
//   - SetupListeners: Initializes and starts the listeners for both high-resolution
//     and low-resolution media topics, attaching the corresponding processing workflows.
package main

import (
	"context"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"github.com/jaycherian/gcp-go-media-search/internal/core/workflow"
)

// SetupListeners configures and starts the background Pub/Sub listeners.
// It creates the necessary media processing workflows and attaches them to the
// appropriate topic listeners.
//
// Inputs:
//   - config: The application's configuration, containing settings for storage, topics, etc.
//   - cloudClients: A struct containing all the initialized Google Cloud service clients.
//   - ctx: The application's root context, used to manage the lifecycle of the listeners.
//
// Outputs:
//   - This function does not return any value. It starts the listeners as background goroutines.
func SetupListeners(config *cloud.Config, cloudClients *cloud.ServiceClients, ctx context.Context) {
	// TODO - Externalize the destination topic and ffmpeg command

	// Create the workflow for resizing high-resolution videos.
	// This workflow is triggered by messages on the HiResTopic and uses FFmpeg to transcode files.
	mediaResizeWorkflow := workflow.NewMediaResizeWorkflow(config, cloudClients, "/snap/bin/ffmpeg", &model.MediaFormatFilter{Width: "240"})
	// Assign the resize workflow as the command to be executed by the listener for the high-resolution topic.
	cloudClients.PubSubListeners["HiResTopic"].SetCommand(mediaResizeWorkflow)
	// Start the listener in a background goroutine. It will now begin receiving and processing messages from its subscription.
	cloudClients.PubSubListeners["HiResTopic"].Listen(ctx)

	// Create the workflow for ingesting and analyzing low-resolution videos.
	// This workflow uses the "creative-flash" GenAI model for analysis.
	mediaIngestion := workflow.NewMediaReaderPipeline(config, cloudClients, "creative-flash")

	// Assign the ingestion workflow to the listener for the low-resolution topic.
	cloudClients.PubSubListeners["LowResTopic"].SetCommand(mediaIngestion)
	// Start the listener for the low-resolution topic.
	cloudClients.PubSubListeners["LowResTopic"].Listen(ctx)
}
