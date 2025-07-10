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
// background process for generating vector embeddings for media scenes.
package workflow

import (
	goctx "context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"cloud.google.com/go/bigquery"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"google.golang.org/api/iterator"
	"google.golang.org/genai"
)

// MediaEmbeddingGeneratorWorkflow defines a background job that periodically
// scans for media records in BigQuery that haven't been processed for embeddings.
// For each unprocessed media, it generates vector embeddings for every scene's
// script using a Vertex AI model and saves them back to a separate BigQuery table.
// This implements the cor.Command interface, allowing it to be part of a larger chain,
// although it's designed to run independently as a background task.

type MediaEmbeddingGeneratorWorkflow struct {
	cor.BaseCommand
	// Muziris change genai does not have an EmbeddingModels structure but has a Models structure now.
	genaiEmbedding *genai.Models
	bigqueryClient *bigquery.Client
	//Muziris change need model name for later in the code when embedding content
	modelName              string
	dataset                string
	mediaTable             string
	embeddingTable         string
	findEligibleMediaQuery string
}

// StartTimer kicks off the background process for the workflow. It creates a
// time.Ticker that fires at a regular interval (every 60 seconds). On each tick,
// it executes the embedding generation logic within a new trace span for observability.
// This function runs indefinitely in a separate goroutine until the application is shut down.
func (m *MediaEmbeddingGeneratorWorkflow) StartTimer() {
	// Obtain a tracer for creating spans.
	tracer := otel.Tracer("embedding-batch")
	// Set the ticker to run the job every 60 seconds.
	ticker := time.NewTicker(60 * time.Second)
	// A channel to signal when the ticker should be stopped (for graceful shutdown).
	closeTicker := make(chan struct{})

	// Start a new goroutine to handle the timed execution.
	go func(m *MediaEmbeddingGeneratorWorkflow) {
		for {
			select {
			// This case is triggered each time the ticker fires.
			case <-ticker.C:
				// Start a new OpenTelemetry trace span for this execution run.
				traceCtx, span := tracer.Start(goctx.Background(), "media-embeddings")
				// Create a fresh context for this run of the workflow.
				chainCtx := cor.NewBaseContext()
				chainCtx.SetContext(traceCtx)

				// Execute the main logic of the workflow.
				m.Execute(chainCtx)

				// Check if any errors occurred during execution and update the span status.
				if chainCtx.HasErrors() {
					span.SetStatus(codes.Error, "failed to execute embedding chain")
				} else {
					span.SetStatus(codes.Ok, "executed embeddings")
				}
				// End the span for this execution.
				span.End()
			// This case would be triggered if a value is sent to the closeTicker channel.
			case <-closeTicker:
				ticker.Stop()
				return
			}
		}
	}(m)
}

// NewMediaEmbeddingGeneratorWorkflow is the constructor for the embedding workflow.
// It initializes the workflow with all necessary clients and configuration.
// It also constructs the specific BigQuery query needed to find media records
// that are missing embeddings.
//
// Inputs:
//   - config: The application's overall configuration object.
//   - serviceClients: A struct containing all the initialized Google Cloud service clients.
//
// Returns:
//   - A pointer to a newly created and configured MediaEmbeddingGeneratorWorkflow.
func NewMediaEmbeddingGeneratorWorkflow(config *cloud.Config, serviceClients *cloud.ServiceClients) *MediaEmbeddingGeneratorWorkflow {

	// Construct fully qualified names for the BigQuery tables to prevent ambiguity.
	fqMediaTableName := strings.Replace(serviceClients.BiqQueryClient.Dataset(config.BigQueryDataSource.DatasetName).Table(config.BigQueryDataSource.MediaTable).FullyQualifiedName(), ":", ".", -1)
	fqEmbeddingTable := strings.Replace(serviceClients.BiqQueryClient.Dataset(config.BigQueryDataSource.DatasetName).Table(config.BigQueryDataSource.EmbeddingTable).FullyQualifiedName(), ":", ".", -1)

	// Define the SQL query to find all media IDs from the main media table that do NOT
	// exist in the embedding table. This identifies unprocessed media.
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE ID NOT IN (SELECT MEDIA_ID FROM `%s`)", fqMediaTableName, fqEmbeddingTable)

	// Return a new instance of the workflow struct, populated with clients,
	// configuration, and the generated query.
	return &MediaEmbeddingGeneratorWorkflow{
		BaseCommand:            *cor.NewBaseCommand("media-embedding-generator"),
		genaiEmbedding:         serviceClients.EmbeddingModels["multi-lingual"],
		bigqueryClient:         serviceClients.BiqQueryClient,
		dataset:                config.BigQueryDataSource.DatasetName,
		mediaTable:             config.BigQueryDataSource.MediaTable,
		embeddingTable:         config.BigQueryDataSource.EmbeddingTable,
		findEligibleMediaQuery: query,
	}
}

// IsExecutable determines if the command can be run. For this workflow, it always
// returns true because it's a self-contained background job that doesn't depend on
// prior outputs in a chain context.
func (m *MediaEmbeddingGeneratorWorkflow) IsExecutable(_ cor.Context) bool {
	return true
}

// Execute contains the core logic for the workflow. It queries BigQuery for
// unprocessed media, iterates through them, generates embeddings for each scene,
// and inserts the new embeddings back into BigQuery.
//
// Inputs:
//   - context: The chain of responsibility context, used for passing state and errors.
func (m *MediaEmbeddingGeneratorWorkflow) Execute(context cor.Context) {
	// Create a BigQuery query object from the predefined SQL string.
	q := m.bigqueryClient.Query(m.findEligibleMediaQuery)
	// Execute the query and get an iterator for the results.
	it, err := q.Read(context.GetContext())
	if err != nil {
		context.AddError(m.GetName(), err)
		return
	}

	// Loop through all the rows (unprocessed media records) returned by the query.
	for {
		var value model.Media
		// Get the next row and deserialize it into a Media struct.
		err := it.Next(&value)
		// `iterator.Done` is the standard way to detect the end of the results.
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			context.AddError(m.GetName(), err)
			return
		}

		// Create a slice to hold the new embedding objects for the current media file.
		toInsert := make([]*model.SceneEmbedding, 0)

		// Iterate over each scene within the media object.
		for _, scene := range value.Scenes {
			// Create a new embedding object, initializing it with metadata.
			in := model.NewSceneEmbedding(value.Id, scene.SequenceNumber, m.modelName)
			// Call the Vertex AI model to generate an embedding for the scene's script text.

			contents := []*genai.Content{
				genai.NewContentFromText(scene.Script, genai.RoleUser),
			}

			// Embed the content using the specified embedding model.
			// Replace "gemini-embedding-exp-03-07" with your desired embedding model.
			resultemb, erremb := m.genaiEmbedding.EmbedContent(context.GetContext(), m.modelName, contents, nil)
			if err != nil {
				fmt.Print("Fatal error when creating embeddings", erremb)
			}
			// Commented for Muziris
			// resp, err := m.genaiEmbedding.EmbedContent(context.GetContext(), genai.Text(scene.Script))
			// if err != nil {
			// 	context.AddError(m.GetName(), err)
			// 	return
			// }
			// The response contains the vector embedding as a slice of floats.
			// Append these values to our embedding object.
			for _, f := range resultemb.Embeddings {
				for _, g := range f.Values {
					in.Embeddings = append(in.Embeddings, float64(g))
				}
			}
			// Add the fully populated embedding object to the list for insertion.
			toInsert = append(toInsert, in)
		}

		// Once all scenes for a media file are processed, insert the batch of
		// new embeddings into the BigQuery embedding table.
		inserter := m.bigqueryClient.Dataset(m.dataset).Table(m.embeddingTable).Inserter()
		if err := inserter.Put(context.GetContext(), toInsert); err != nil {
			context.AddError(m.GetName(), err)
			return
		}
	}
}
