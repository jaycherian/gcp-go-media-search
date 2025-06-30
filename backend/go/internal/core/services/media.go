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

package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	credentials "cloud.google.com/go/iam/credentials/apiv1"

	//	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"cloud.google.com/go/storage"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	// credentialspb "google.golang.org/genproto/googleapis/iam/credentials/v1"
)

type MediaService struct {
	BigqueryClient *bigquery.Client
	StorageClient  *storage.Client // This field was missing
	IAMClient      *credentials.IamCredentialsClient
	SignerEmail    string
	DatasetName    string
	MediaTable     string
}

// GetFQN returns the fully qualified BQ Table Name
func (s *MediaService) GetFQN() string {
	return strings.Replace(s.BigqueryClient.Dataset(s.DatasetName).Table(s.MediaTable).FullyQualifiedName(), ":", ".", -1)
}

// Get returns a media object by id, or an error if it doesn't exist
func (s *MediaService) Get(ctx context.Context, id string) (media *model.Media, err error) {
	queryText := fmt.Sprintf(QryFindMediaById, s.GetFQN(), id)
	q := s.BigqueryClient.Query(queryText)
	itr, err := q.Read(ctx)
	if err != nil {
		return media, err
	}
	// Since this should only return a single result
	media = &model.Media{}
	err = itr.Next(media)
	return media, err
}

// GetScene returns a scene in a specified media type by its sequence number
func (s *MediaService) GetScene(ctx context.Context, id string, sceneSequence int) (scene *model.Scene, err error) {
	fqMediaTableName := strings.Replace(s.BigqueryClient.Dataset(s.DatasetName).Table(s.MediaTable).FullyQualifiedName(), ":", ".", -1)
	queryText := fmt.Sprintf(QryGetScene, fqMediaTableName, id, sceneSequence)
	q := s.BigqueryClient.Query(queryText)
	itr, err := q.Read(ctx)
	if err != nil {
		return scene, err
	}
	scene = &model.Scene{}
	// Since this should only return a single result
	err = itr.Next(scene)
	return scene, err
}

// GenerateSignedURL creates a time-limited URL to access a private GCS object.
func (s *MediaService) GenerateSignedURL(ctx context.Context, gcsURI string, expires time.Duration) (string, error) {
	// Parse the GCS URI to get the bucket and object name.
	// Example URI: https://storage.mtls.cloud.google.com/bucket_name/Serenity.mp4
	parts := strings.Split(strings.TrimPrefix(gcsURI, "https://storage.mtls.cloud.google.com/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GCS URI: %s", gcsURI)
	}
	bucketName := parts[0]
	objectName := strings.Join(parts[1:], "/")
	print("---------------------------------------------------\n")
	print(fmt.Sprintf("Google Project ID is %s\n", s.SignerEmail))
	print(fmt.Sprintf("Email is %s\n", s.SignerEmail))
	print(fmt.Sprintf("projects/-/serviceAccounts/%s\n", s.SignerEmail))
	print("---------------------------------------------------\n")
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expires),
		// Use the IAM credentials client to sign the bytes for the URL.
		// This is the recommended approach for services running on GCP.

		// SignBytes: func(b []byte) ([]byte, error) {
		// 	req := &credentialspb.SignBlobRequest{
		// 		Name:    fmt.Sprintf("projects/-/serviceAccounts/%s", s.SignerEmail),
		// 		Payload: b,
		// 	}
		// 	resp, err := s.IAMClient.SignBlob(ctx, req)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	return resp.SignedBlob, nil
	}
	u, err := s.StorageClient.Bucket(bucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", fmt.Errorf("Bucket(%q).SignedURL: %w", bucketName, err)
	}

	fmt.Println("Generated GET signed URL:")
	fmt.Printf("%s\n", u)
	fmt.Println("You can use this URL with any user agent, for example:")
	fmt.Print("curl \n", u)
	return u, nil

	// url, err := s.StorageClient.Bucket(bucketName).Object(objectName).SignedURL(ctx, opts)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to sign URL: %w", err)
	// }
	// return url, nil
	// Use the top-level SignedURL function for V4 signing.
	// print("just before returning from GenerateSignedURL\n")
	// return storage.SignedURL(bucketName, objectName, opts)
}
