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
// This file, `media.go`, defines the MediaService, which is responsible for
// retrieving media and scene data from BigQuery and generating secure,
// time-limited URLs for accessing media files stored in Google Cloud Storage (GCS).
package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/storage"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
)

// MediaService is a struct that encapsulates the clients and configuration
// needed to perform media-related operations. It acts as a data access layer,
// abstracting the details of interacting with BigQuery and GCS.
type MediaService struct {
	BigqueryClient *bigquery.Client                  // Client for interacting with Google BigQuery.
	StorageClient  *storage.Client                   // Client for interacting with Google Cloud Storage.
	IAMClient      *credentials.IamCredentialsClient // Client for interacting with IAM, used for signing URLs.
	SignerEmail    string                            // The service account email used to sign URLs.
	DatasetName    string                            // The name of the BigQuery dataset (e.g., "media_ds").
	MediaTable     string                            // The name of the BigQuery table containing media metadata.
}

// GetFQN (Get Fully Qualified Name) returns the complete, queryable name for the
// media table in BigQuery, formatted with dots instead of colons.
// Example: `gcp-project-id.media_ds.media`
//
// Outputs:
//   - string: The fully qualified table name.
func (s *MediaService) GetFQN() string {
	// Get the default fully qualified name (e.g., "gcp-project-id:media_ds.media").
	fqn := s.BigqueryClient.Dataset(s.DatasetName).Table(s.MediaTable).FullyQualifiedName()
	// Replace the colon with a period for compatibility with standard SQL queries.
	return strings.Replace(fqn, ":", ".", -1)
}

// Get retrieves a single media object from BigQuery based on its unique ID.
//
// Inputs:
//   - ctx: The context for the request, used for cancellation and tracing.
//   - id: The unique identifier of the media object to retrieve.
//
// Outputs:
//   - *model.Media: A pointer to the retrieved media object.
//   - error: An error if the query fails or no media is found.
func (s *MediaService) Get(ctx context.Context, id string) (media *model.Media, err error) {
	// Construct the SQL query using the fully qualified table name and the provided ID.
	queryText := fmt.Sprintf(QryFindMediaById, s.GetFQN(), id)
	// Create a new BigQuery query object.
	q := s.BigqueryClient.Query(queryText)
	// Execute the query.
	itr, err := q.Read(ctx)
	if err != nil {
		return media, err // Return an error if the query execution fails.
	}
	// Since ID is a primary key, we expect only one result.
	// Initialize a new Media object to hold the result.
	media = &model.Media{}
	// Scan the next (and only) row from the iterator into the media struct.
	err = itr.Next(media)
	// Return the populated media object and any error from the row scan.
	return media, err
}

// GetScene retrieves a specific scene from a media object by its sequence number.
//
// Inputs:
//   - ctx: The context for the request.
//   - id: The unique ID of the parent media object.
//   - sceneSequence: The sequence number of the scene to retrieve.
//
// Outputs:
//   - *model.Scene: A pointer to the retrieved scene object.
//   - error: An error if the query fails or the scene is not found.
func (s *MediaService) GetScene(ctx context.Context, id string, sceneSequence int) (scene *model.Scene, err error) {
	// Get the fully qualified name for the media table.
	fqMediaTableName := s.GetFQN()
	// Construct the SQL query. This query unnests the `scenes` array in BigQuery
	// to allow filtering by the scene's sequence number.
	queryText := fmt.Sprintf(QryGetScene, fqMediaTableName, id, sceneSequence)
	q := s.BigqueryClient.Query(queryText)
	itr, err := q.Read(ctx)
	if err != nil {
		return scene, err
	}
	// Initialize a new Scene object to hold the result.
	scene = &model.Scene{}
	// Scan the next (and only) row from the iterator into the scene struct.
	err = itr.Next(scene)
	// Return the populated scene object and any error from the row scan.
	return scene, err
}

// GenerateSignedURL creates a time-limited, secure URL to access a private GCS object.
// This allows clients (like a web browser) to stream video directly from GCS
// without needing their own credentials. The URL is signed using the credentials
// of the service account specified in `s.SignerEmail`.
//
// Inputs:
//   - ctx: The context for the request.
//   - gcsURI: The URI of the GCS object (e.g., "https://storage.mtls.cloud.google.com/bucket/object.mp4").
//   - expires: The duration for which the URL will be valid.
//
// Outputs:
//   - string: The generated signed URL.
//   - error: An error if parsing the URI or signing the URL fails.
func (s *MediaService) GenerateSignedURL(ctx context.Context, gcsURI string, expires time.Duration) (string, error) {
	// ---- 1. Parse the GCS URI ----
	// The full URI needs to be broken down into its bucket and object components.
	// Example URI: https://storage.mtls.cloud.google.com/my-bucket/my-folder/my-video.mp4
	prefix := "https://storage.mtls.cloud.google.com/"
	if !strings.HasPrefix(gcsURI, prefix) {
		return "", fmt.Errorf("invalid GCS URI format: %s", gcsURI)
	}
	// Remove the prefix to get "my-bucket/my-folder/my-video.mp4"
	path := strings.TrimPrefix(gcsURI, prefix)
	// Split the remaining path by the first slash.
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GCS URI: unable to determine bucket and object from %s", gcsURI)
	}
	bucketName := parts[0] // "my-bucket"
	objectName := parts[1] // "my-folder/my-video.mp4"

	print("---------------------------------------------------\n")
	print(fmt.Sprintf("Google Project ID is %s\n", s.SignerEmail))
	print(fmt.Sprintf("Email is %s\n", s.SignerEmail))
	print(fmt.Sprintf("projects/-/serviceAccounts/%s\n", s.SignerEmail))
	print("---------------------------------------------------\n")

	// ---- 2. Define Signing Options ----
	// Configure the parameters for the V4 signing process.
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4, // Use the modern and more secure V4 signing scheme.
		Method:  "GET",                   // The URL will only be valid for GET requests.
		Expires: time.Now().Add(expires), // Set the expiration time.

		// // The SignBytes field provides a function that signs the request data.
		// // This is the recommended approach when running on GCP infrastructure, as it
		// // uses the IAM Credentials API and avoids the need for local service account keys.
		// SignBytes: func(b []byte) ([]byte, error) {
		// 	// Construct the full resource name of the service account that will perform the signing.
		// 	req := &credentialspb.SignBlobRequest{
		// 		Name:    fmt.Sprintf("projects/-/serviceAccounts/%s", s.SignerEmail),
		// 		Payload: b, // The byte slice to be signed.
		// 	}
		// 	// Call the IAM Credentials API to sign the blob.
		// 	resp, err := s.IAMClient.SignBlob(ctx, req)
		// 	if err != nil {
		// 		// Return the error if the signing process fails.
		// 		return nil, fmt.Errorf("IAMClient.SignBlob: %w", err)
		// 	}
		// 	// Return the resulting signed byte slice.
		// 	return resp.SignedBlob, nil
		// },
	}

	// ---- 3. Generate and Return the URL ----
	// Call the GCS client library's SignedURL method with the object details and signing options.
	u, err := s.StorageClient.Bucket(bucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", fmt.Errorf("Bucket(%q).Object(%q).SignedURL: %w", bucketName, objectName, err)
	}
	fmt.Println("Generated GET signed URL:")
	fmt.Printf("%s\n", u)
	fmt.Println("You can use this URL with any user agent, for example:")
	fmt.Print("curl \n", u)
	return u, nil
}
