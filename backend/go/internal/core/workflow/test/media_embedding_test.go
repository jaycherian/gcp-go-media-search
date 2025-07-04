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
// This file, `media_embedding_test.go`, specifically tests the functionality of the
// `MediaEmbeddingGeneratorWorkflow`. This workflow is responsible for periodically
// finding media objects in BigQuery that do not yet have embeddings and generating
// those embeddings using a Vertex AI model.
package workflow_test

import (
	"fmt"
	"testing"

	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"

	"github.com/jaycherian/gcp-go-media-search/internal/core/workflow"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
)

// TestMediaEmbeddings is an integration test that verifies the end-to-end process
// of the embedding generation workflow. It initializes and executes the workflow,
// then asserts that no errors occurred during the process. This confirms that the
// workflow can correctly query BigQuery, interact with the Vertex AI embedding model,
// and persist the results.
//
// Inputs:
//   - t: A pointer to the testing.T object, provided by the Go testing framework,
//     used for logging, error reporting, and assertions.
func TestMediaEmbeddings(t *testing.T) {
	// Start a new OpenTelemetry trace span for this test. This helps in monitoring
	// and debugging the test's execution in a distributed tracing system.
	// The `defer span.End()` ensures the span is closed when the function exits.
	traceCtx, span := tracer.Start(ctx, "generate_embeddings")
	defer span.End()

	// Create a new chain of responsibility (cor) context. This context is used to pass
	// data and state between different commands within the workflow.
	chainCtx := cor.NewBaseContext()
	// Set the Go context (with tracing information) into our custom chain context.
	chainCtx.SetContext(traceCtx)

	// Initialize the media embedding generator workflow using the shared configuration
	// and cloud clients that were set up in `base_test.go`.
	embeddingWorkflow := workflow.NewMediaEmbeddingGeneratorWorkflow(config, cloudClients)

	// Execute the workflow. This will trigger the logic to find unprocessed media
	// and generate embeddings for their scenes.
	embeddingWorkflow.Execute(chainCtx)

	// After execution, check the context for any errors that may have been added
	// by commands within the workflow. Log them for debugging purposes.
	for _, e := range chainCtx.GetErrors() {
		fmt.Printf("Error: %v \n", e)
	}

	// Use the testify/assert library to check that the workflow completed without errors.
	// This is the primary success condition for this test.
	assert.False(t, chainCtx.HasErrors())

	// If the test passes, set the status of the trace span to "Ok" to indicate a
	// successful execution in the tracing system.
	span.SetStatus(codes.Ok, "success")
}
