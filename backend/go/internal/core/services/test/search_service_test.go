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

// Package services_test contains the test suite for the services package.
// This file specifically tests the functionality of the SearchService.
package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/services"
	test "github.com/jaycherian/gcp-go-media-search/internal/testutil"
	"github.com/zeebo/assert"
)

// TestSearchService is an integration test for the FindScenes method of the
// SearchService. It initializes a full application stack (configuration, cloud clients),
// then creates an instance of the SearchService. It executes a sample search query
// against a live BigQuery backend and asserts that the operation completes
// without errors. This test validates the end-to-end flow of generating an
// embedding for a query and performing a vector search in BigQuery.
//
// Inputs:
//   - t: The testing framework's test handler.
func TestSearchService(t *testing.T) {
	// Create a new context with a cancel function. This allows us to gracefully
	// manage the lifecycle of the cloud clients and any background operations.
	ctx, cancel := context.WithCancel(context.Background())
	// The defer statement ensures that cancel() is called when the function exits,
	// which is crucial for releasing resources and preventing leaks.
	defer cancel()

	// Load the application configuration from .toml files using a test helper.
	// This helper sets the necessary environment variables to load test-specific configs.
	config := test.GetConfig()

	// Initialize all necessary Google Cloud service clients (Storage, Pub/Sub, GenAI, BigQuery)
	// based on the loaded configuration. This creates the 'live' environment for the test.
	cloudClients, err := cloud.NewCloudServiceClients(ctx, config)
	// Use a test helper to fail the test immediately if client initialization fails.
	test.HandleErr(err, t)
	// Ensure that all client connections are closed when the test function completes.
	defer cloudClients.Close()

	// Retrieve a specific generative AI model from the initialized clients.
	// While not directly used in this test, this confirms that the agent models
	// were loaded correctly from the configuration.
	genModel := cloudClients.AgentModels["creative-flash"]
	assert.NotNil(t, genModel)

	// Retrieve the multi-lingual embedding model, which is a required dependency
	// for the SearchService to convert text queries into vector embeddings.
	embeddingModel := cloudClients.EmbeddingModels["multi-lingual"]

	// Instantiate the SearchService with its dependencies: the BigQuery client,
	// the embedding model, and the names of the dataset and tables to query.
	searchService := &services.SearchService{
		BigqueryClient: cloudClients.BiqQueryClient,
		EmbeddingModel: embeddingModel,
		DatasetName:    "media_ds",
		MediaTable:     "media",
		EmbeddingTable: "scene_embeddings",
	}

	// Execute the method under test: FindScenes.
	// This sends the query "Scenes that Woody Harrelson" to the service. The service
	// will generate an embedding for this text and then perform a k-nearest neighbor
	// (KNN) vector search in BigQuery to find the top 5 most similar scenes.
	out, err := searchService.FindScenes(ctx, "Scenes that Woody Harrelson", 5)

	// Perform a basic check for an error. If an error occurred, the test fails.
	if err != nil {
		t.Error(err)
	}

	// Use the testify/assert library for a more explicit assertion that the
	// error should be nil, providing clearer test output on failure.
	assert.Nil(t, err)

	// If the search is successful, iterate through the results and print them.
	// This is useful for debugging and manually verifying the search results
	// during development. The output will show the media ID and sequence number
	// for each matching scene.
	for _, o := range out {
		fmt.Printf("%s - %d\n", o.MediaId, o.SequenceNumber)
	}
}
