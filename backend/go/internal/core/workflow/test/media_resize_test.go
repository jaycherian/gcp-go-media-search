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

// Package workflow_test contains integration tests for the core application workflows.
// This file, `media_resize_test.go`, specifically tests the `MediaResizeWorkflow`.
// This workflow is responsible for handling the initial video processing step:
// taking a high-resolution video from a GCS bucket, resizing it to a smaller,
// standard resolution using FFmpeg, and uploading the result to a separate GCS bucket
// for low-resolution videos.
package workflow_test

import (
	"log"
	"testing"

	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"github.com/jaycherian/gcp-go-media-search/internal/core/workflow"
	test "github.com/jaycherian/gcp-go-media-search/internal/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
)

// TestFFMpegCommand performs an end-to-end integration test of the media resizing
// workflow (`MediaResizeWorkflow`). It simulates a Pub/Sub trigger from a high-resolution
// video upload, runs the entire chain of commands (download, resize, upload), and
// asserts that the workflow completes without any errors. Although named TestFFMpegCommand,
// it tests the entire workflow, not just the FFmpeg command in isolation.
//
// Inputs:
//   - t: A pointer to the testing.T object, provided by the Go testing framework,
//     used for logging, error reporting, and assertions.
func TestFFMpegCommand(t *testing.T) {
	// Start an OpenTelemetry trace span for this test run. This helps in monitoring
	// and debugging by providing a detailed trace of the test's execution.
	// `defer span.End()` ensures the span is closed when the function completes.
	traceContext, span := tracer.Start(ctx, "media-resize-test")
	defer span.End()

	// Initialize the media resize workflow. We provide it with the shared test configuration,
	// cloud clients, a specific path to the ffmpeg executable for the test environment,
	// and a filter that defines the target width for the resized video.
	mediaResizeWorkflow := workflow.NewMediaResizeWorkflow(config, cloudClients, "bin/ffmpeg", &model.MediaFormatFilter{Width: "240"})

	// Create a new chain of responsibility (cor) context. This context will carry
	// state and data through the various steps (commands) of the workflow.
	chainCtx := cor.NewBaseContext()
	// Set the Go context (which includes tracing information) into the chain context.
	chainCtx.SetContext(traceContext)
	// Add the initial input data to the context. `GetTestHighResMessageText` provides
	// a mock JSON string representing a GCS notification for a new high-res video.
	chainCtx.Add(cor.CtxIn, test.GetTestHighResMessageText())

	// This is a pre-check assertion to ensure that the workflow considers itself
	// executable with the provided context. It's a good practice to verify that
	// initial conditions are met before running the main logic.
	assert.True(t, mediaResizeWorkflow.IsExecutable(chainCtx))

	// Execute the entire media resize workflow. This will trigger the sequence of
	// commands defined within the workflow: download from GCS, run FFmpeg, upload to GCS.
	mediaResizeWorkflow.Execute(chainCtx)

	// After the workflow has run, check if any errors were added to the context
	// by the commands and log them for debugging purposes.
	for _, err := range chainCtx.GetErrors() {
		log.Printf("error in chain: %v", err.Error())
	}

	// If the context has errors, mark the trace span with an error status to
	// clearly indicate a failure in the tracing system.
	if chainCtx.HasErrors() {
		span.SetStatus(codes.Error, "failed - media-resize-test")
	}

	// This is the final and most important assertion. It verifies that the workflow
	// completed without any errors, which is the definition of a successful test run.
	assert.False(t, chainCtx.HasErrors())

	// If the test passed, mark the trace span as "Ok".
	span.SetStatus(codes.Ok, "passed - media-resize-test")
}
