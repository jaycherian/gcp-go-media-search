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
// Responsibility (COR) pattern's Command interface. This file defines a
// command for uploading a local file to a specified Google Cloud Storage (GCS) bucket.
//
// Logic Flow:
// This command is a step in a larger workflow, typically following a command
// that creates a file locally (like the FFMpegCommand). Its purpose is to
// take that local file and persist it to GCS.
//
//  1. Get the path of the local file to upload from the COR context.
//  2. Get the original GCS object metadata from the context. This is used to
//     ensure the uploaded file retains the same name as the object that
//     triggered the workflow. If this metadata is not present, it falls back
//     to using the local file's name.
//  3. Open the local file for reading.
//  4. Schedule the local temporary file for deletion after the function completes.
//  5. Get a handle to the destination GCS bucket and create a writer object for
//     the destination object name.
//  6. Use `io.Copy` to efficiently stream the file's contents from the local disk
//     directly to the GCS bucket.
//  7. Handle any errors and perform cleanup.
package commands

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
)

// GCSFileUpload is a command implementation responsible for uploading a file
// from the local filesystem to a Google Cloud Storage bucket.
type GCSFileUpload struct {
	cor.BaseCommand                 // Embeds the BaseCommand for common functionality like naming and metrics.
	client          *storage.Client // The GCS client for interacting with the storage service.
	bucket          string          // The name of the destination GCS bucket.
}

// NewGCSFileUpload is the constructor for creating a new GCSFileUpload command.
//
// Inputs:
//   - name: A string name for this command instance, used for logging and telemetry.
//   - client: An initialized *storage.Client for communicating with GCS.
//   - bucket: The name of the target GCS bucket for the upload.
//
// Outputs:
//   - *GCSFileUpload: A pointer to the newly instantiated command.
func NewGCSFileUpload(name string, client *storage.Client, bucket string) *GCSFileUpload {
	return &GCSFileUpload{BaseCommand: *cor.NewBaseCommand(name), client: client, bucket: bucket}
}

// Execute contains the core logic for the command. It reads a local file
// and streams its content to a GCS bucket.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (c *GCSFileUpload) Execute(context cor.Context) {
	// Retrieve the local file path from the context, which was put there by a previous command.
	path := context.Get(c.GetInputParam()).(string)
	// Extract just the filename from the full path.
	name := filepath.Base(path)

	// Retrieve the metadata of the original GCS object that started the workflow.
	// This ensures that if the input was "video.mov", the output is also named "video.mov"
	// in the destination bucket, even if the local temp file had a different name.
	original := context.Get(cloud.GetGCSObjectName()).(*cloud.GCSObject)

	// Open the local file for reading.
	dat, err := os.Open(path)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to open file %s: %w", path, err))
		return
	}

	// Defer the removal of the local temporary file. This function will be executed
	// when the Execute function returns, ensuring cleanup.
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Printf("failed to remove file from OS: %v\n", err)
		}
	}(path)

	// Get a handle for the destination bucket.
	writerBucket := c.client.Bucket(c.bucket)
	var obj *storage.ObjectHandle

	// Determine the name of the object to be created in GCS.
	if original != nil {
		// If we have the original object's metadata, use its name.
		obj = writerBucket.Object(original.Name)
	} else {
		// Otherwise, fall back to using the name of the local file.
		obj = writerBucket.Object(name)
	}

	// Create a new writer for the GCS object. This opens a stream to GCS.
	writer := obj.NewWriter(context.GetContext())

	// Defer closing the writer. This is critical to finalize the upload.
	// If the writer is not closed, the object may not be created or may be incomplete.
	defer func(writer *storage.Writer) {
		err := writer.Close()
		if err != nil {
			log.Printf("failed to close GCS writer: %v\n", err)
		}
	}(writer)

	// Use io.Copy to stream the file content from the local reader (`dat`) to the GCS writer.
	// This is memory-efficient as it doesn't load the entire file into memory.
	if written, err := io.Copy(writer, dat); err != nil {
		log.Printf("failed to copy to GCS or partial write: %d total bytes, %v\n", written, err)
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), err)
		return
	}

	// If the copy is successful, increment the success counter.
	c.GetSuccessCounter().Add(context.GetContext(), 1)
	log.Printf("Successfully uploaded %s to gs://%s/%s", name, c.bucket, obj.ObjectName())
}
