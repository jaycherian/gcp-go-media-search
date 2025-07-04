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

// Package cloud contains data structures and utilities for interacting with Google Cloud services.
// This file specifically defines models related to Google Cloud Storage (GCS), including the
// structure for GCS Pub/Sub notifications and a simplified internal representation of a GCS object.
//
// Structs:
//   - GCSPubSubNotification: Maps to the JSON payload from GCS event notifications.
//   - GCSObject: A simplified internal model for GCS objects used in processing workflows.
//
// Functions:
//   - GetGCSObjectName: Returns a constant key used for storing GCS object data in a context.
package cloud

// GetGCSObjectName returns a constant string that is used as a key within the
// Chain of Responsibility (CoR) context. This key allows different commands in a workflow
// to consistently access the `GCSObject` data that is being processed.
//
// Outputs:
//   - string: A constant placeholder string "__GCS__OBJ__".
func GetGCSObjectName() string {
	return "__GCS__OBJ__"
}

// GCSPubSubNotification is the structure that maps to the JSON message payload
// received from a Google Cloud Storage (GCS) Pub/Sub notification. When an event
// (like object creation or update) occurs in a monitored bucket, GCS sends a message
// with this structure to the configured Pub/Sub topic.
type GCSPubSubNotification struct {
	Kind                    string                 `json:"kind"`                    // The kind of the object, typically "storage#object".
	ID                      string                 `json:"id"`                      // The full ID of the object, including bucket and generation.
	SelfLink                string                 `json:"selfLink"`                // The URI for this object.
	Name                    string                 `json:"name"`                    // The name of the object within the bucket.
	Bucket                  string                 `json:"bucket"`                  // The name of the bucket containing the object.
	Generation              string                 `json:"generation"`              // The generation number of the object's content.
	MetaGeneration          string                 `json:"metageneration"`          // The generation number of the object's metadata.
	ContentType             string                 `json:"contentType"`             // The MIME type of the object's content.
	TimeCreated             string                 `json:"timeCreated"`             // The creation time of the object.
	Updated                 string                 `json:"updated"`                 // The last modification time of the object.
	StorageClass            string                 `json:"storageClass"`            // The storage class of the object.
	TimeStorageClassUpdated string                 `json:"timeStorageClassUpdated"` // The time the storage class was last updated.
	Size                    string                 `json:"size"`                    // The size of the object in bytes.
	MD5Hash                 string                 `json:"md5Hash"`                 // The MD5 hash of the object's content.
	MediaLink               string                 `json:"mediaLink"`               // A link to download the object's content.
	MetaData                map[string]interface{} `json:"metadata"`                // User-provided metadata, if any.
	Crc32c                  string                 `json:"crc32c"`                  // The CRC32C checksum of the object's content.
	ETag                    string                 `json:"etag"`                    // The HTTP ETag of the object.
}

// GCSObject is a simplified, internal representation of a Google Cloud Storage (GCS)
// object. It distills the most essential information from the more verbose
// GCSPubSubNotification into a lightweight struct that is easier to pass
// between commands in a processing workflow.
type GCSObject struct {
	Bucket   string // The name of the GCS bucket.
	Name     string // The name of the object.
	MIMEType string // The MIME type of the object (e.g., "video/mp4").
}
