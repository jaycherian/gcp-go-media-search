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
// This file, `queries.go`, centralizes all the BigQuery SQL query strings used
// by the application's services. Storing queries as constants in a dedicated
// file improves maintainability, readability, and reusability. The queries use
// `fmt.Sprintf` format verbs (e.g., %s, %d) as placeholders for dynamic values
// that will be injected at runtime.
package services

const (
	// QrySequenceKnn defines the BigQuery query for performing a k-nearest neighbor (KNN)
	// vector search. This is the core query for the semantic search functionality.
	//
	// How it works:
	// - `VECTOR_SEARCH`: This is a BigQuery native function that efficiently finds the most similar
	//   vectors in a table to a given query vector.
	// - `TABLE %s`: The first placeholder is for the fully qualified name of the embeddings table.
	// - `'embeddings'`: This is the name of the column in the table that stores the embedding vectors.
	// - `(SELECT [ %s ] as embed)`: The second placeholder is for the query vector itself.
	//   The application will generate an embedding from the user's search text and insert it
	//   here as a comma-separated list of floating-point numbers.
	// - `top_k => %d`: The third placeholder is for the 'k' in KNN. It specifies the number
	//   of closest matches to return.
	// - `distance_type => 'EUCLIDEAN'`: This specifies the algorithm used to measure the "distance"
	//   or similarity between vectors. EUCLIDEAN is a common choice for this type of search.
	// - `ORDER BY distance asc`: This sorts the results by the calculated distance in ascending
	//   order, ensuring that the most similar items (with the smallest distance) appear first.
	//
	// The query returns the `media_id` and `sequence_number` of the matching scenes.
	QrySequenceKnn = "SELECT base.media_id, base.sequence_number FROM VECTOR_SEARCH(TABLE `%s`, 'embeddings', (SELECT [ %s ] as embed), top_k => %d, distance_type => 'EUCLIDEAN') ORDER BY distance asc"

	// QryFindMediaById defines a simple lookup query to retrieve a complete media record
	// from the media table using its unique ID.
	//
	// Placeholders:
	// - `%s`: The fully qualified name of the `media` table.
	// - `%s`: The unique ID of the media object to find.
	QryFindMediaById = "SELECT * from `%s` WHERE id = '%s'"

	// QryGetScene defines a query to extract a single, specific scene from the nested
	// `scenes` array within a media record.
	//
	// How it works:
	// - `UNNEST(scenes) as s`: This is a powerful BigQuery function that "flattens" the
	//   repeated `scenes` field (which is an array of structs) into a relational,
	//   table-like structure aliased as `s`. This allows us to query individual scenes
	//   as if they were rows in a table.
	// - `WHERE id = '%s' and s.sequence = %d`: This filters the unnested scenes, first by
	//   finding the correct parent media document by its `id`, and then by finding the
	//   specific scene within that document by its `sequence` number.
	//
	// Placeholders:
	// - `%s`: The fully qualified name of the `media` table.
	// - `%s`: The unique ID of the parent media object.
	// - `%d`: The sequence number of the desired scene.
	QryGetScene = "SELECT sequence, start, `end`, script FROM `%s`, UNNEST(scenes) as s WHERE id = '%s' and s.sequence = %d"
)
