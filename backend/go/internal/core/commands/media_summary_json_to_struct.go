// Copyright 2024 Google, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package commands provides the concrete implementations of the Chain of
// Responsibility (COR) pattern's Command interface. This file defines a
// command that acts as a data transformation step in the workflow.
//
// Logic Flow:
// This command follows the `MediaSummaryCreator` in the chain. It takes the
// raw JSON string output from the generative model and transforms it into a
// strongly-typed Go struct (`model.MediaSummary`). This is a crucial step for
// making the data easy and safe to work with in subsequent parts of the
// application. It also enriches the data by adding the full GCS URL for the
// media file.
//
//  1. It receives the raw JSON string from the context (output of the previous command).
//  2. It also retrieves the original `GCSObject` from the context to get the
//     bucket and object name.
//  3. It uses Go's standard `json.Unmarshal` function to parse the JSON string
//     into a `model.MediaSummary` struct.
//  4. After successful parsing, it constructs the full, accessible URL for the
//     media file (e.g., "https://storage.mtls.cloud.google.com/bucket/file.mp4").
//     This is done here because the generative model only knows about the file's
//     temporary handle, not its final storage location.
//  5. It puts the final, populated `model.MediaSummary` struct back into the
//     context, ready for the next command (like `SceneExtractor`).
package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
)

// MediaSummaryJsonToStruct is a command that parses a JSON string into a MediaSummary struct.
type MediaSummaryJsonToStruct struct {
	cor.BaseCommand // Embeds the BaseCommand for common functionality.
}

// NewMediaSummaryJsonToStruct is the constructor for the MediaSummaryJsonToStruct command.
//
// Inputs:
//   - name: A string name for this command instance.
//   - outputParamName: The context key where the resulting struct will be stored.
//
// Outputs:
//   - *MediaSummaryJsonToStruct: A pointer to the newly instantiated command.
func NewMediaSummaryJsonToStruct(name string, outputParamName string) *MediaSummaryJsonToStruct {
	out := MediaSummaryJsonToStruct{BaseCommand: *cor.NewBaseCommand(name)}
	// Set the specific output parameter name for this command instance.
	out.OutputParamName = outputParamName
	return &out
}

// Execute contains the core logic for parsing the JSON and enriching the data.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (s *MediaSummaryJsonToStruct) Execute(context cor.Context) {
	// Retrieve the raw JSON string from the context, which was the output of the previous command.
	in := context.Get(s.GetInputParam()).(string)

	// Retrieve the GCSObject which contains details about the original file location.
	gcsFile := context.Get(cloud.GetGCSObjectName()).(*cloud.GCSObject)

	// Create an empty MediaSummary struct to hold the parsed data.
	doc := &model.MediaSummary{}

	// Unmarshal (parse) the JSON string into the Go struct.
	err := json.Unmarshal([]byte(in), &doc)
	if err != nil {
		// If parsing fails, it's a critical error. Record it and stop.
		s.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(s.GetName(), fmt.Errorf("failed to unmarshal media summary JSON: %w", err))
		return
	}

	// If parsing is successful, increment the success counter.
	s.GetSuccessCounter().Add(context.GetContext(), 1)

	// Enrich the data: construct the full, direct-access URL to the media file in GCS.
	// The model doesn't know this URL, so we construct it here using the bucket and object name.
	doc.MediaUrl = fmt.Sprintf("https://storage.mtls.cloud.google.com/%s/%s", gcsFile.Bucket, gcsFile.Name)

	// Place the populated and enriched struct into the designated output parameter in the context.
	context.Add(s.GetOutputParam(), doc)

	// Also place it in the general-purpose output slot for the next command in the chain.
	context.Add(cor.CtxOut, doc)
}
