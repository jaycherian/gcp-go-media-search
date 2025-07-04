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

// Package commands provides the concrete implementations of the Chain of
// Responsibility (COR) pattern's Command interface. This file defines the
// command responsible for persisting the final media object to BigQuery.
//
// Logic Flow:
// This command is a crucial persistence step in the workflow. It takes the
// fully assembled `model.Media` struct, which contains all the extracted
// metadata (summary, cast, scenes, etc.), and inserts it as a new row into
// a specified BigQuery table. This makes the data available for later querying
// and for the asynchronous embedding generation process.
//
//  1. It retrieves the complete `model.Media` object from the context.
//  2. It gets a BigQuery `Inserter`. The inserter is an optimized client for
//     streaming data into a table, which is more efficient than running
//     individual `INSERT` statements.
//  3. It uses the `Put` method of the inserter to send the `model.Media` struct
//     to BigQuery. The Go client library automatically handles marshalling the
//     struct fields into the corresponding BigQuery table columns based on the
//     `bigquery` struct tags in `model.Media`.
//  4. It performs error handling and updates telemetry counters.
package commands

import (
	"fmt"
	"log"

	"cloud.google.com/go/bigquery"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
)

// MediaPersistToBigQuery is a command that saves a Media object to a BigQuery table.
type MediaPersistToBigQuery struct {
	cor.BaseCommand
	client     *bigquery.Client // The client for interacting with the BigQuery service.
	dataset    string           // The name of the BigQuery dataset.
	table      string           // The name of the target table within the dataset.
	mediaParam string           // The context key for the input `model.Media` object.
}

// NewMediaPersistToBigQuery is the constructor for the MediaPersistToBigQuery command.
//
// Inputs:
//   - name: A string name for this command instance.
//   - client: An initialized *bigquery.Client.
//   - dataset: The name of the BigQuery dataset.
//   - table: The name of the target table.
//   - mediaParam: The name of the context parameter holding the `model.Media` object to be saved.
//
// Outputs:
//   - *MediaPersistToBigQuery: A pointer to the newly instantiated command.
func NewMediaPersistToBigQuery(name string, client *bigquery.Client, dataset string, table string, mediaParam string) *MediaPersistToBigQuery {
	return &MediaPersistToBigQuery{BaseCommand: *cor.NewBaseCommand(name), client: client, dataset: dataset, table: table, mediaParam: mediaParam}
}

// IsExecutable overrides the default behavior to ensure that the Media object
// to be persisted exists in the context before execution.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
//
// Outputs:
//   - bool: True if the Media object exists in the context, otherwise false.
func (s *MediaPersistToBigQuery) IsExecutable(context cor.Context) bool {
	return context != nil && context.Get(s.mediaParam) != nil
}

// Execute contains the core logic for writing the data to BigQuery.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (s *MediaPersistToBigQuery) Execute(context cor.Context) {
	log.Println("Persisting media metadata to BigQuery...")

	// Retrieve the fully assembled Media object from the context.
	media := context.Get(s.mediaParam).(*model.Media)

	// Get an Inserter for the target table. This provides a streaming interface
	// for inserting rows into BigQuery, which is highly efficient.
	i := s.client.Dataset(s.dataset).Table(s.table).Inserter()

	// Use the Put method to insert the Media object. The BigQuery client library
	// automatically maps the fields of the struct to the table columns.
	if err := i.Put(context.GetContext(), media); err != nil {
		log.Printf("failed to write media to database. title %s error %s\n", media.Title, err)
		s.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(s.GetName(), fmt.Errorf("bigquery insert failed for title '%s': %w", media.Title, err))
		return
	}

	// On success, update telemetry and pass the media object to the next command.
	s.GetSuccessCounter().Add(context.GetContext(), 1)
	context.Add(cor.CtxOut, media)
	log.Printf("Successfully persisted media metadata for '%s' (ID: %s)", media.Title, media.Id)
}
