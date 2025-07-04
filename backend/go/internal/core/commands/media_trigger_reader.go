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
// initial command in the media processing workflow.
//
// Logic Flow:
// This command serves as the entry point for any workflow that is triggered by
// a file being uploaded to Google Cloud Storage (GCS). GCS publishes a
// detailed notification message to a Pub/Sub topic when a new object is
// created or updated. This command is responsible for parsing that message.
//
//  1. The command receives the raw Pub/Sub message data as a JSON string from the context.
//  2. It unmarshals (parses) this JSON string into a `cloud.GCSPubSubNotification`
//     struct, which represents the full, complex structure of the GCS notification.
//  3. It then extracts the most essential pieces of information—the bucket name,
//     the object name, and the content type.
//  4. It creates a new, much simpler `cloud.GCSObject` struct containing only
//     this essential information.
//  5. This simplified `GCSObject` is then placed back into the context, making it
//     easy for subsequent commands in the chain to access the file's location
//     without needing to understand the full GCS notification format.
package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"

	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
)

// MediaTriggerToGCSObject is a command that parses a GCS Pub/Sub notification
// and extracts key file information into a simplified GCSObject.
type MediaTriggerToGCSObject struct {
	cor.BaseCommand // Embeds the BaseCommand for common functionality.
}

// NewMediaTriggerToGCSObject is the constructor for the MediaTriggerToGCSObject command.
//
// Inputs:
//   - name: A string name for this command instance.
//
// Outputs:
//   - *MediaTriggerToGCSObject: A pointer to the newly instantiated command.
func NewMediaTriggerToGCSObject(name string) *MediaTriggerToGCSObject {
	return &MediaTriggerToGCSObject{BaseCommand: *cor.NewBaseCommand(name)}
}

// Execute contains the core logic for parsing the GCS notification message.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution, containing
//     the raw message data in the input parameter.
func (c *MediaTriggerToGCSObject) Execute(context cor.Context) {
	// Retrieve the raw JSON message string from the context.
	in := context.Get(c.GetInputParam()).(string)

	// Declare a variable of the target type to hold the unmarshaled data.
	var out cloud.GCSPubSubNotification

	// Parse the JSON string into the GCSPubSubNotification struct.
	err := json.Unmarshal([]byte(in), &out)
	if err != nil {
		// If parsing fails, it's a critical error for the workflow.
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to unmarshal GCS notification: %w", err))
		return
	}

	// If successful, increment the success counter for telemetry.
	c.GetSuccessCounter().Add(context.GetContext(), 1)

	// Create a new, simplified GCSObject containing only the essential information
	// needed by downstream commands.
	msg := &cloud.GCSObject{Bucket: out.Bucket, Name: out.Name, MIMEType: out.ContentType}

	// Add the simplified GCSObject to the context using a well-known key
	// so that other commands can easily access it.
	context.Add(cloud.GetGCSObjectName(), msg)

	// Also add the GCSObject to the default output parameter so it becomes the
	// input for the very next command in the chain.
	context.Add(c.GetOutputParam(), msg)
}
