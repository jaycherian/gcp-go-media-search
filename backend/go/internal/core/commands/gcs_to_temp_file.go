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
// command for downloading an object from Google Cloud Storage (GCS) to a
// local temporary file.
//
// Logic Flow:
// This command serves as a bridge between a GCS-based workflow and a
// local-file-based tool (like FFmpeg). It takes GCS object information,
// downloads the file to the local machine, and passes the local file's path
// to the next command in the chain.
//
//  1. Receives a `cloud.GCSObject` struct from the context, which contains the
//     bucket and object name.
//  2. Creates a reader for the specified GCS object.
//  3. Creates a new empty temporary file on the local disk.
//  4. Efficiently streams the content from the GCS reader directly into the
//     local temporary file using `io.Copy`.
//  5. Adds the path of the newly created temporary file to the context for two
//     purposes:
//     a) To be used as input for the next command in the workflow.
//     b) To be tracked for cleanup after the entire workflow is complete.
package commands

import (
	"fmt"
	"io"
	"log"
	"os"

	"cloud.google.com/go/storage"
	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
)

// GCSToTempFile is a command implementation that downloads an object from GCS
// and saves it as a temporary file on the local filesystem.
type GCSToTempFile struct {
	cor.BaseCommand                 // Embeds the BaseCommand for common functionality like naming and metrics.
	client          *storage.Client // The GCS client for interacting with the storage service.
	tempFilePrefix  string          // A prefix to use when naming the temporary file (e.g., "ffmpeg-").
}

// NewGCSToTempFile is the constructor for creating a new GCSToTempFile command.
//
// Inputs:
//   - name: A string name for this command instance, used for logging and telemetry.
//   - client: An initialized *storage.Client for communicating with GCS.
//   - tempFilePrefix: A string prefix for the temporary file's name.
//
// Outputs:
//   - *GCSToTempFile: A pointer to the newly instantiated command.
func NewGCSToTempFile(name string, client *storage.Client, tempFilePrefix string) *GCSToTempFile {
	return &GCSToTempFile{
		BaseCommand:    *cor.NewBaseCommand(name),
		client:         client,
		tempFilePrefix: tempFilePrefix,
	}
}

// Execute contains the core logic for downloading the GCS object.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (c *GCSToTempFile) Execute(context cor.Context) {
	// Retrieve the GCS object metadata from the context's input parameter.
	msg := context.Get(c.GetInputParam()).(*cloud.GCSObject)

	// Get a client handle for the specified bucket and object.
	readerBucket := c.client.Bucket(msg.Bucket)
	obj := readerBucket.Object(msg.Name)

	// Create a new reader to stream the object's data from GCS.
	reader, err := obj.NewReader(context.GetContext())
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to create GCS reader for gs://%s/%s: %w", msg.Bucket, msg.Name, err))
		return
	}
	// Defer closing the reader. This is important to release resources.
	defer func(reader *storage.Reader) {
		err := reader.Close()
		if err != nil {
			// Log the error but don't stop the workflow, as the data might have been read successfully.
			log.Printf("failed to close GCS reader: %v\n", err)
		}
	}(reader)

	// Create a temporary file on the local disk. The "" for the first argument
	// means it will be created in the default temporary directory for the OS.
	tempFile, err := os.CreateTemp("", c.tempFilePrefix)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("could not create temp file: %w", err))
		return
	}

	// Use io.Copy to stream the content from the GCS reader to the local temp file.
	// This is memory-efficient because it streams the data in chunks rather than
	// loading the entire file into memory at once.
	written, err := io.Copy(tempFile, reader)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		log.Printf("failed to copy GCS object to local file, %d bytes written: %v\n", written, err)
		context.AddError(c.GetName(), err)
		// It's good practice to close the temp file handle here before returning.
		_ = tempFile.Close()
		return
	}
	// The copy is complete, so we close the file handle to ensure data is flushed to disk.
	_ = tempFile.Close()

	c.GetSuccessCounter().Add(context.GetContext(), 1)
	log.Printf("Successfully downloaded gs://%s/%s to local file %s (%d bytes)", msg.Bucket, msg.Name, tempFile.Name(), written)
	// Add the path of the new temp file to the context's list of tracked temp files.
	// This allows the main workflow manager to clean them up automatically at the end.
	context.AddTempFile(tempFile.Name())
	// Place the temp file's path into the context's output parameter, making it
	// the default input for the next command in the chain.
	context.Add(c.GetOutputParam(), tempFile.Name())
}
