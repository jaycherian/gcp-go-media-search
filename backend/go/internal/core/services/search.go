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

// Package services contains the business logic for interacting with data sources.
// This file, `search.go`, defines the SearchService, which is responsible for
// handling the core semantic search functionality. It takes a natural language
// query, converts it into a vector embedding using a generative AI model, and
// then uses that vector to find the most similar items in a BigQuery table.
package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"google.golang.org/api/iterator"
	"google.golang.org/genai"
)

// SearchService encapsulates the clients and configuration needed to perform
// semantic search operations. It holds references to the BigQuery client for
// database interaction and a GenAI embedding model for converting text to vectors.
type SearchService struct {
	BigqueryClient *bigquery.Client // Client for interacting with Google BigQuery.
	EmbeddingModel *genai.Models    // The generative AI model used to create vector embeddings from text.
	//Muziris Change: Accomodate the latest genai libraries
	ModelName      string // The name of the model
	DatasetName    string // The name of the BigQuery dataset.
	MediaTable     string // The name of the table holding primary media metadata.
	EmbeddingTable string // The name of the table holding the vector embeddings for scenes.
}

// FindScenes takes a text query, generates a vector embedding for it, and then
// performs a vector search (k-nearest neighbor) in BigQuery to find the most
// semantically similar scenes.
//
// Inputs:
//   - ctx: The context for the request, used for cancellation, deadlines, and tracing.
//   - query: The natural language search string from the user (e.g., "a scene with a car chase").
//   - maxResults: The maximum number of similar scenes to return (the 'k' in k-nearest neighbor).
//
// Outputs:
//   - []*model.SceneMatchResult: A slice of pointers to SceneMatchResult objects,
//     each containing the ID of the media and the sequence number of the matching scene.
//   - error: An error if any step (embedding, query, or row scanning) fails.
func (s *SearchService) FindScenes(ctx context.Context, query string, maxResults int) (out []*model.SceneMatchResult, err error) {
	// Initialize the output slice to ensure it's not nil, even if no results are found.
	out = make([]*model.SceneMatchResult, 0)

	// --- Step 1: Generate Embedding for the Query ---
	// Call the generative AI model to convert the user's text query into a vector embedding.
	//Muziris Change: new Embedcontent call because of deprecrated genai libraries

	contents := []*genai.Content{
		genai.NewContentFromText(query, genai.RoleUser),
	}

	// Embed the content using the specified embedding model.
	// Replace "gemini-embedding-exp-03-07" with your desired embedding model.
	searchEmbeddings, erremb := s.EmbeddingModel.EmbedContent(ctx, s.ModelName, contents, nil)
	if err != nil {
		fmt.Print("Fatal error when creating embeddings", erremb)
	}

	// --- Step 2: Prepare the Query for BigQuery ---
	// Get the fully qualified name of the embeddings table (e.g., `project.dataset.table`).
	fqEmbeddingTable := strings.Replace(s.BigqueryClient.Dataset(s.DatasetName).Table(s.EmbeddingTable).FullyQualifiedName(), ":", ".", -1)

	// The BigQuery VECTOR_SEARCH function expects the query vector as a comma-separated
	// string of float values. We convert the float32 slice from the embedding model
	// into a slice of strings.
	var stringArray []string
	for _, f := range searchEmbeddings.Embeddings[0].Values {
		stringArray = append(stringArray, strconv.FormatFloat(float64(f), 'f', -1, 64))
	}

	// Construct the final SQL query by injecting the table name, the vector string,
	// and the max number of results into the QrySequenceKnn template.
	queryText := fmt.Sprintf(QrySequenceKnn, fqEmbeddingTable, strings.Join(stringArray, ","), maxResults)

	// --- Step 3: Execute the Query and Process Results ---
	// Create a new BigQuery query object.
	q := s.BigqueryClient.Query(queryText)
	// Execute the query and get an iterator for the results.
	itr, err := q.Read(ctx)
	if err != nil {
		return out, fmt.Errorf("failed to read from BigQuery: %w", err)
	}

	// Iterate through all the rows returned by BigQuery.
	for {
		var r = &model.SceneMatchResult{}
		// Scan the next row into the SceneMatchResult struct.
		err := itr.Next(r)
		// If we've reached the end of the results, break the loop.
		if err == iterator.Done {
			break
		}
		// If any other error occurs during iteration, return it.
		if err != nil {
			return out, fmt.Errorf("failed to iterate results: %w", err)
		}
		// If the row was scanned successfully, append the result to our output slice.
		out = append(out, r)
	}

	// Return the populated slice of results and a nil error, indicating success.
	return out, nil
}
