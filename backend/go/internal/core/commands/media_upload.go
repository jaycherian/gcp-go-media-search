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
// command responsible for uploading a local media file to the Vertex AI
// File Service.
//
// Logic Flow:
// This command acts as a critical bridge between a temporary local file (which
// might have been downloaded from GCS or created by a process like FFmpeg) and
// the Vertex AI ecosystem. The Gemini models require a file handle from their
// native File Service to perform multimodal analysis on video content.
//
//  1. It takes the path of a local temporary file as its input from the context.
//  2. It uses the `genai.Client` to upload this local file to the Vertex AI File
//     Service using `UploadFileFromPath`. Metadata from the original GCS object,
//     like its name and MIME type, is passed along to maintain consistency.
//  3. **Crucial Step**: After the initial upload API call, the file enters a
//     `FileStateProcessing` state on the backend. The file is not yet ready for
//     use by the model.
//  4. The command enters a `for` loop that polls the status of the file every
//     five seconds by calling `client.GetFile`.
//  5. This loop continues until the file's state is no longer `Processing`,
//     meaning it has become `ACTIVE` (ready) or `FAILED`.
//  6. Once the file is `ACTIVE`, the command places the resulting `genai.File`
//     object, which acts as a handle or reference, into the context. Subsequent
//     commands, like `MediaSummaryCreator`, will use this handle to tell the
//     Gemini model which file to analyze.

//	IMPORTANT UPDATE:
//
// Muziris Change: the fact of the matter is that with the new genai libraries, there is no need to
// "Upload" video files for processing. We can just provide the URI to GCS objects for the model
// This function can ideally be deprecrated but for now to keep the chain of responsibility intact,
// this function will be a shell that returns the GCS object name and Mime type.
package commands

import (
	"fmt"
	"time"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"

	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"google.golang.org/genai"
)

// MediaUpload is a command that uploads a local file to the Vertex AI File Service
// and waits for it to become active.
type MediaUpload struct {
	cor.BaseCommand
	client           *genai.Client // The client for interacting with the Vertex AI File Service.
	timeoutInSeconds time.Duration // A timeout for the upload operation (currently unused but available).
}

// NewMediaUpload is the constructor for the MediaUpload command.
//
// Inputs:
//   - name: A string name for this command instance.
//   - genaiClient: The client for the Vertex AI API.
//   - timeoutInSeconds: The duration to wait before timing out.
//
// Outputs:
//   - *MediaUpload: A pointer to the newly instantiated command.
func NewMediaUpload(name string, genaiClient *genai.Client, timeoutInSeconds time.Duration) *MediaUpload {
	return &MediaUpload{BaseCommand: *cor.NewBaseCommand(name), client: genaiClient, timeoutInSeconds: timeoutInSeconds}
}

// GetVideoUploadFileParameterName returns the canonical key used to store the
// resulting `genai.File` handle in the context. Using a function for this
// ensures consistency across different commands that need to access this value.
func GetVideoUploadFileParameterName() string {
	return "__VIDEO_UPLOAD_FILE__"
}

// Execute contains the core logic for uploading the file and polling for its status.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
//
// Muziris Change: the fact of the matter is that with the new genai libraries, there is no need to
// "Upload" video files for processing. We can just provide the URI to GCS objects for the model
// This function can ideally be deprecrated but for now to keep the chain of responsibility intact,
// this function will be a shell that returns the GCS object name and Mime type.
func (v *MediaUpload) Execute(context cor.Context) {
	// Retrieve the original GCS object details from the context to get metadata.
	gcsFile := context.Get(cloud.GetGCSObjectName()).(*cloud.GCSObject)
	GCSFileLink := fmt.Sprintf("gs://%s/%s ", gcsFile.Bucket, gcsFile.Name)
	fmt.Print("\nThe GCS filename for media upload is: ", GCSFileLink)
	var GCSFileStruct genai.FileData
	GCSFileStruct.FileURI = GCSFileLink
	GCSFileStruct.MIMEType = gcsFile.MIMEType
	// // Retrieve the local filesystem path of the temporary file to be uploaded.
	// fileName := context.Get(v.GetInputParam()).(string)

	// // Call the SDK to upload the file from the local path. Pass along the original
	// // display name and MIME type for consistency in the File Service.
	// genFil, err := v.client.UploadFileFromPath(context.GetContext(), fileName, &genai.UploadFileOptions{DisplayName: gcsFile.Name, MIMEType: gcsFile.MIMEType})
	// if err != nil {
	// 	v.GetErrorCounter().Add(context.GetContext(), 1)
	// 	context.AddError(v.GetName(), fmt.Errorf("failed to upload file to File Service: %w", err))
	// 	return
	// }

	// // === Polling Loop ===
	// // The file is not ready for use immediately after the upload call.
	// // We must wait for Vertex AI to finish processing it.
	// for genFil.State == genai.FileStateProcessing {
	// 	// Pause for a short duration to avoid excessive API calls.
	// 	time.Sleep(5 * time.Second)
	// 	var err error
	// 	// Fetch the latest status of the file.
	// 	if genFil, err = v.client.GetFile(context.GetContext(), genFil.Name); err != nil {
	// 		v.GetErrorCounter().Add(context.GetContext(), 1)
	// 		context.AddError(v.GetName(), fmt.Errorf("failed to get file status during processing: %w", err))
	// 		return
	// 	}
	// }

	// // If the file processing failed on the backend, this is a critical error.
	// if genFil.State == genai.FileStateFailed {
	// 	v.GetErrorCounter().Add(context.GetContext(), 1)
	// 	context.AddError(v.GetName(), err)
	// 	return
	// }

	// Once the loop completes and the file is active, record the success.
	v.GetSuccessCounter().Add(context.GetContext(), 1)

	// Store the `genai.File` handle in the context using the canonical key.
	context.Add(GetVideoUploadFileParameterName(), &GCSFileStruct)
	// Also place it in the default output parameter for the next command in the chain.
	context.Add(v.GetOutputParam(), &GCSFileStruct)
}
