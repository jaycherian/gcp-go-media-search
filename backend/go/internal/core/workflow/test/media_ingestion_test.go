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
// This file, `media_ingestion_test.go`, tests the complete `MediaReaderPipeline`.
// This workflow is the core of the media analysis process, triggered after a
// video has been resized. It handles downloading the video, sending it to
// Vertex AI for summary and scene analysis, assembling the results, and persisting
// the final media object to BigQuery.
package workflow_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"github.com/jaycherian/gcp-go-media-search/internal/core/workflow"
	test "github.com/jaycherian/gcp-go-media-search/internal/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
)

// TestMediaChain performs an end-to-end integration test of the media ingestion
// and analysis workflow (`MediaReaderPipeline`). It simulates a Pub/Sub trigger
// from a low-resolution video upload and runs the entire chain of commands to
// process it. The test's success is determined by whether the workflow completes
// without any errors being added to its context.
//
// Inputs:
//   - t: A pointer to the testing.T object, provided by the Go testing framework,
//     used for logging, error reporting, and assertions.
func TestMediaChain(t *testing.T) {
	// Start a new OpenTelemetry trace span. This allows us to trace the execution
	// of this specific test within a distributed tracing system like Google Cloud Trace.
	traceCtx, span := tracer.Start(ctx, "media-ingestion-test")
	defer span.End()

	// Initialize the primary workflow to be tested: the MediaReaderPipeline.
	// We pass it the shared config and cloud clients, and specify "creative-flash"
	// as the name of the generative model configuration to use for the analysis.
	mediaIngestion := workflow.NewMediaReaderPipeline(config, cloudClients, "creative-flash")

	// Create a new chain of responsibility (cor) context to manage state
	// throughout the workflow execution.
	chainCtx := cor.NewBaseContext()
	// Pass the Go context (which includes our tracing information) into the chain context.
	chainCtx.SetContext(traceCtx)
	// Set the initial input for the workflow. We use a helper function to get a
	// JSON string that mimics a real Pub/Sub notification from a GCS event.
	chainCtx.Add(cor.CtxIn, test.GetTestLowResMessageText())

	// Execute the entire media ingestion workflow.
	mediaIngestion.Execute(chainCtx)

	// After execution, loop through any errors that were recorded in the context
	// by the workflow's commands and print them for debugging.
	for k, err := range chainCtx.GetErrors() {
		fmt.Printf("Error: (%s): %v\n", k, err)
	}

	// If the context contains any errors, we mark the trace span with an error status.
	if chainCtx.HasErrors() {
		span.SetStatus(codes.Error, "failed to execute media ingestion test")
	}

	// The primary assertion of the test: verify that the workflow's context has no errors.
	// If this passes, it means every command in the chain executed successfully.
	assert.False(t, chainCtx.HasErrors())

	// Mark the trace span as "Ok" to signify a successful test run.
	span.SetStatus(codes.Ok, "passed - media ingestion test")

	// For debugging purposes, log the final media object that was assembled
	// by the workflow. This can be useful for manually verifying the output.
	log.Println(chainCtx.Get("MEDIA"))
}
