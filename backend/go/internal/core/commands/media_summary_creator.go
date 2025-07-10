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
// command responsible for generating a high-level summary of a media file.
//
// Logic Flow:
// This command is one of the first and most critical steps in the AI analysis
// pipeline. It takes a media file that has been uploaded to the Vertex AI
// File Service and uses a generative model (like Gemini) to extract key
// metadata. This metadata includes the title, a descriptive summary, a list of
// cast members, and, most importantly, a series of timestamps that identify
// distinct scenes within the media.
//
//  1. It receives a `genai.File` object from the context, which is a handle
//     to the processed media file in the Vertex AI File Service.
//  2. It constructs a detailed prompt for the generative model using a Go template.
//     This prompt instructs the model on what information to extract and in what
//     format (JSON) it should be returned.
//  3. The prompt is populated with dynamic data, such as a list of valid media
//     categories and an example of the desired JSON output structure, to guide
//     the model's response (few-shot prompting).
//  4. It sends the media file handle and the generated prompt to the generative
//     model in a multi-modal request.
//  5. It receives the raw JSON string response from the model.
//  6. It places this JSON string into the context for the next command in the
//     chain (`MediaSummaryJsonToStruct`) to parse and process.
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"go.opentelemetry.io/otel/metric"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"google.golang.org/genai"
)

// MediaSummaryCreator is a command that uses a generative model to create a
// summary and extract metadata from a video file.
type MediaSummaryCreator struct {
	cor.BaseCommand
	config                   *cloud.Config                      // Application configuration, used for prompt templating.
	generativeAIModel        *cloud.QuotaAwareGenerativeAIModel // The rate-limited generative model client.
	template                 *template.Template                 // The Go template for building the prompt.
	geminiInputTokenCounter  metric.Int64Counter                // OTel counter for input tokens.
	geminiOutputTokenCounter metric.Int64Counter                // OTel counter for output tokens.
	geminiRetryCounter       metric.Int64Counter                // OTel counter for retries.
}

// NewMediaSummaryCreator is the constructor for the MediaSummaryCreator command.
//
// Inputs:
//   - name: A string name for this command instance.
//   - config: The application's configuration object.
//   - generativeAIModel: The rate-limited wrapper for the generative model client.
//   - template: A parsed Go template for the prompt.
//
// Outputs:
//   - *MediaSummaryCreator: A pointer to the newly instantiated command, including initialized telemetry counters.
func NewMediaSummaryCreator(
	name string,
	config *cloud.Config,
	generativeAIModel *cloud.QuotaAwareGenerativeAIModel,
	template *template.Template) *MediaSummaryCreator {

	out := &MediaSummaryCreator{
		BaseCommand:       *cor.NewBaseCommand(name),
		config:            config,
		generativeAIModel: generativeAIModel,
		template:          template}

	// Initialize OpenTelemetry counters for monitoring Gemini API usage for this specific command.
	out.geminiInputTokenCounter, _ = out.GetMeter().Int64Counter(fmt.Sprintf("%s.gemini.token.input", out.GetName()))
	out.geminiOutputTokenCounter, _ = out.GetMeter().Int64Counter(fmt.Sprintf("%s.gemini.token.output", out.GetName()))
	out.geminiRetryCounter, _ = out.GetMeter().Int64Counter(fmt.Sprintf("%s.gemini.token.retry", out.GetName()))

	return out
}

// GenerateParams creates the map of dynamic data to be injected into the prompt template.
//
// Inputs:
//   - context: The shared `cor.Context` (currently unused in this function but required by the interface).
//
// Outputs:
//   - map[string]interface{}: A map of keys and values for template substitution.
func (t *MediaSummaryCreator) GenerateParams(_ cor.Context) map[string]interface{} {
	params := make(map[string]interface{})

	// Create a string representation of the media categories from the config
	// to help the model choose a valid category. Example: "trailer - A short...; movie - A feature..."
	catStr := ""
	for key, cat := range t.config.Categories {
		catStr += fmt.Sprintf("%s - %s; ", key, cat.Definition)
	}
	params["CATEGORIES"] = catStr

	// Provide a complete, well-formed JSON example in the prompt. This technique (few-shot prompting)
	// significantly improves the reliability and structure of the model's output.
	exampleSummary, _ := json.Marshal(model.GetExampleSummary())
	params["EXAMPLE_JSON"] = string(exampleSummary)
	return params
}

// Execute contains the core logic for prompting the generative model.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (t *MediaSummaryCreator) Execute(context cor.Context) {
	// Retrieve the `genai.File` object (the handle to the media file in the File Service) from the context.
	mediaFile := context.Get(t.GetInputParam()).(*genai.FileData)

	// Use a buffer to execute the Go template, substituting the dynamic params.
	var buffer bytes.Buffer
	err := t.template.Execute(&buffer, t.GenerateParams(context))
	if err != nil {
		t.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(t.GetName(), fmt.Errorf("failed to execute prompt template: %w", err))
		return
	}

	// Muziris Change
	// // Prepare the parts for the multi-modal request to Gemini.
	// parts := make([]genai.Part, 0)
	// // Part 1: The media file itself, referenced by its URI.
	// parts = append(parts, cloud.NewFileData(mediaFile.URI, mediaFile.MIMEType))
	// // Part 2: The text prompt generated from the template.
	// parts = append(parts, cloud.NewTextPart(buffer.String()))

	//Muziris Change
	// Prepare the parts for the multi-modal request to Gemini.
	contents := []*genai.Content{
		{Parts: []*genai.Part{
			{Text: buffer.String()},
			{FileData: &genai.FileData{
				FileURI:  mediaFile.FileURI,
				MIMEType: mediaFile.MIMEType,
			}},
		},
			Role: "user"},
	}

	// Call the helper function to send the request to the model. This helper
	// encapsulates retry logic and telemetry updates.
	// Muziris Change
	out, err := cloud.GenerateMultiModalResponse(context.GetContext(), t.geminiInputTokenCounter, t.geminiOutputTokenCounter, t.geminiRetryCounter, 0, t.generativeAIModel, contents)
	fmt.Print("\nThe output of the GnerateMultiModalResponse function is:", out)
	if err != nil {
		t.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(t.GetName(), fmt.Errorf("gemini request failed: %w", err))
		return
	}

	// On success, update the success counter and place the raw JSON string
	// response into the context for the next command.
	t.GetSuccessCounter().Add(context.GetContext(), 1)
	context.Add(t.GetOutputParam(), out)
}
