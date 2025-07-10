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
// command for cleaning up temporary files from the Vertex AI File Service.
//
// Logic Flow:
// After a media file has been uploaded to the Vertex AI File Service for
// analysis (e.g., by the MediaUpload command) and all necessary information
// has been extracted, it's good practice to delete the temporary file to
// manage resources and costs. This command handles that final cleanup step.
//
//  1. It retrieves the `genai.File` object from the context. This object
//     was created by the `MediaUpload` command and contains the unique name
//     (handle) of the file in the Vertex AI File Service.
//  2. It calls the `client.DeleteFile` method, passing the file's name to
//     permanently remove it from the service.
//  3. It handles any errors that might occur during the deletion process.

//	IMPORTANT UPDATE:
//
// Muziris Change: the fact of the matter is that with the new genai libraries, there is no need to
// "Upload" video files for processing. We can just provide the URI to GCS objects for the model
// This function can ideally be deprecrated but for now to keep the chain of responsibility intact,
// this function will be a shell that returns the GCS object name and Mime type.
package commands

import (
	"fmt"

	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"google.golang.org/genai"
)

// MediaCleanup is a command that deletes a file from the Vertex AI File Service.
type MediaCleanup struct {
	cor.BaseCommand               // Embeds the BaseCommand for common functionality.
	client          *genai.Client // The client for interacting with the Vertex AI service.
}

// NewMediaCleanup is the constructor for the MediaCleanup command.
//
// Inputs:
//   - name: A string name for this command instance.
//   - client: An initialized *genai.Client for communicating with Vertex AI.
//
// Outputs:
//   - *MediaCleanup: A pointer to the newly instantiated command.
func NewMediaCleanup(name string, client *genai.Client) *MediaCleanup {
	return &MediaCleanup{BaseCommand: *cor.NewBaseCommand(name), client: client}
}

// IsExecutable overrides the default behavior to ensure that the file object to be
// deleted exists in the context before attempting to execute.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
//
// Outputs:
//   - bool: True if the file object exists in the context, otherwise false.
func (v *MediaCleanup) IsExecutable(context cor.Context) bool {
	// Checks that the context is not nil, a value for the video file parameter exists,
	// and that the value is a valid *genai.File object.
	return context != nil && context.Get(GetVideoUploadFileParameterName()) != nil &&
		//Muziris change
		context.Get(GetVideoUploadFileParameterName()).(*genai.FileData) != nil
}

// Execute performs the deletion logic.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (v *MediaCleanup) Execute(context cor.Context) {
	// Retrieve the file object from the context using a shared parameter name function
	// to ensure consistency across commands.
	//Muziris Change
	fil := context.Get(GetVideoUploadFileParameterName()).(*genai.FileData)
	fmt.Print("Within meadia cleanup for file :", fil.FileURI)

	// Muziris Change: the fact of the matter is that with the new genai libraries, there is no need to
	// "Upload" video files for processing. We can just provide the URI to GCS objects for the model
	// This function can ideally be deprecrated but for now to keep the chain of responsibility intact,
	// this function will be a shell that returns the GCS object name and Mime type.
	// Call the Vertex AI API to delete the file using its unique name.
	// err := v.client.DeleteFile(context.GetContext(), fil.Name)
	// if err != nil {
	// 	// If an error occurs, record it in the context and update metrics.
	// 	v.GetErrorCounter().Add(context.GetContext(), 1)
	// 	context.AddError(v.GetName(), fmt.Errorf("failed to delete file %s from Vertex AI: %w", fil.Name, err))
	// 	return
	// }
	// If successful, increment the success counter.
	v.GetSuccessCounter().Add(context.GetContext(), 1)
}
