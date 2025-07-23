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

// Package workflow defines the high-level business logic orchestrations,
// combining various commands into coherent pipelines. This file implements the
// workflow for resizing media files.
package workflow

import (
	"strings"

	"cloud.google.com/go/storage"
	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/commands"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
)

// DefaultFfmpegCommand defines the default command to execute FFmpeg.
// It assumes `ffmpeg` is available in the system's PATH.
const DefaultFfmpegCommand = "ffmpeg"

// DefaultWidth sets the default target width for video resizing, a common size for previews.
const DefaultWidth = "240"

// MediaResizeWorkflow orchestrates the video transcoding process. It's designed
// to be triggered by an event (like a file upload to a GCS bucket), and it
// executes a sequence of commands to download, resize, and re-upload a video file.
// This struct holds the necessary configuration and the command chain itself.
type MediaResizeWorkflow struct {
	cor.BaseCommand
	ffmpegCommand    string
	videoFormat      *model.MediaFormatFilter
	storageClient    *storage.Client
	outputBucketName string
	chain            cor.Chain // The underlying chain of commands to be executed.
	config           *cloud.Config
}

// Execute runs the media resize workflow by invoking the underlying command chain.
// This is the entry point for the workflow's execution.
//
// Inputs:
//   - context: The chain of responsibility context for this execution, which carries
//     the initial trigger message and passes state between commands.
func (m *MediaResizeWorkflow) Execute(context cor.Context) {
	m.chain.Execute(context)
}

// initializeChain constructs the sequence of commands that define the resize workflow.
// This method is called by the constructor to set up the processing pipeline.
func (m *MediaResizeWorkflow) initializeChain() {
	// Create a new chain instance to hold the sequence of commands.
	out := cor.NewBaseChain(m.GetName())

	// Step 1: Parse the incoming Pub/Sub trigger message to get the GCS object details.
	out.AddCommand(commands.NewMediaTriggerToGCSObject("gcs-topic-listener"))

	// Step 3: Execute the FFmpeg command on the local file to resize it.
	// The `videoFormat.Width` determines the target resolution.
	out.AddCommand(commands.NewFFMpegCommand("video-resize", m.ffmpegCommand, m.videoFormat.Width, m.config))

	// Assign the fully constructed chain to the workflow instance.
	m.chain = out
}

// NewMediaResizeWorkflow is the constructor for the MediaResizeWorkflow. It initializes
// the workflow with all necessary clients and configurations, and builds the command chain.
//
// Inputs:
//   - config: The application's overall configuration.
//   - serviceClients: A struct containing initialized clients for GCP services.
//   - ffmpegCommand: The path to the FFmpeg executable. If empty, a default is used.
//   - videoFormat: A struct specifying the target width and format for resizing.
//
// Returns:
//   - A pointer to a newly created and fully initialized MediaResizeWorkflow.
func NewMediaResizeWorkflow(
	config *cloud.Config,
	serviceClients *cloud.ServiceClients,
	ffmpegCommand string,
	videoFormat *model.MediaFormatFilter) *MediaResizeWorkflow {

	// If no FFmpeg command path is provided, use the default "ffmpeg" command,
	// assuming it's in the system's PATH.
	if len(strings.Trim(ffmpegCommand, " ")) == 0 {
		ffmpegCommand = DefaultFfmpegCommand
	}

	// If no video format filter is provided, create a default one with the
	// standard width and mp4 format.
	if videoFormat == nil {
		videoFormat = &model.MediaFormatFilter{Width: DefaultWidth, Format: "mp4"}
	}

	// Create the MediaResizeWorkflow instance with all its dependencies.
	out := &MediaResizeWorkflow{
		BaseCommand:      *cor.NewBaseCommand("media-resize-workflow"),
		ffmpegCommand:    ffmpegCommand,
		videoFormat:      videoFormat,
		storageClient:    serviceClients.StorageClient,
		outputBucketName: config.Storage.LowResOutputBucket,
		config:           config}
	// Build the command chain for the new pipeline instance.
	out.initializeChain()
	return out
}
