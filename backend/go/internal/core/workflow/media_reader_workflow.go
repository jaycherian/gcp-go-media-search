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

// Package workflow defines the high-level business logic orchestrations,
// combining various commands into coherent pipelines. This file implements the
// primary media analysis workflow.
package workflow

import (
	"text/template"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/commands"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"google.golang.org/genai"
)

// MediaReaderWorkflow orchestrates the entire process of analyzing a low-resolution
// media file. It's structured as a Chain of Responsibility (cor.Chain) that executes
// a sequence of commands to perform tasks like file handling, AI-based analysis,
// data assembly, and persistence.
//
// This workflow is typically triggered by a Pub/Sub message indicating that a new
// low-resolution video is available in a GCS bucket.
type MediaReaderWorkflow struct {
	cor.BaseCommand
	config          *cloud.Config
	bigqueryClient  *bigquery.Client
	genaiClient     *genai.Client
	genaiModel      *cloud.QuotaAwareGenerativeAIModel
	storageClient   *storage.Client
	numberOfWorkers int
	summaryTemplate *template.Template
	sceneTemplate   *template.Template
	chain           cor.Chain // The underlying chain of commands to be executed.
}

// Execute runs the entire media reader workflow by invoking the underlying chain.
// It passes the context, which contains the initial trigger message and will
// be used to pass state between commands.
//
// Inputs:
//   - context: The chain of responsibility context for this execution.
func (m *MediaReaderWorkflow) Execute(context cor.Context) {
	m.chain.Execute(context)
}

// initializeChain builds the sequence of commands that make up this workflow.
// Each command is an atomic unit of work. The output of one command often serves
// as the input for the next, creating a processing pipeline.
// This method is called by the constructor.
func (m *MediaReaderWorkflow) initializeChain() {
	// Define constants for parameter names to avoid magic strings. These keys are used
	// to store and retrieve data from the chain's context.
	const SummaryOutputParamName = "__summary_output__"
	const SceneOutputParamName = "__scene_output__"
	const MediaOutputParamName = "__media_output__"

	// Create the chain that will hold all the command steps.
	out := cor.NewBaseChain(m.GetName())

	// Step 1: Parse the incoming Pub/Sub message (which is in JSON format)
	// and extract a structured GCS object reference from it.
	out.AddCommand(commands.NewMediaTriggerToGCSObject("media-trigger-to-gcs-object"))

	// Step 2: Download the media file from the GCS bucket specified in the trigger
	// message and save it to a temporary local file on the server's disk.
	// Muziris change: With the new libraries it is no longer necessary to have a temp file locally and upload it.
	// We can analyze and extract scenes right from the file in GCS bucket
	out.AddCommand(commands.NewGCSToTempFile("gcs-to-temp-file", m.storageClient, "media-summary-"))

	// Step 3: Upload the temporary local file to the Vertex AI File Service.
	// This service makes the file available for analysis by Gemini models.
	// The operation is given a 5-minute timeout.
	// Muziris change: With the new libraries it is no longer necessary to have a temp file locally and upload it.
	// We can analyze and extract scenes right from the file in GCS bucket
	out.AddCommand(commands.NewMediaUpload("media-upload", m.genaiClient, 300*time.Second))

	// Step 4: Generate a high-level summary of the media file using Gemini.
	// This command takes the file handle from the previous step and a prompt template
	// as input and produces a JSON string with the summary, cast, scenes, etc.
	out.AddCommand(commands.NewMediaSummaryCreator("generate-media-summary", m.config, m.genaiModel, m.summaryTemplate))

	// Step 5: Convert the JSON string summary from the previous step into a Go struct (`model.MediaSummary`).
	// This makes the data easier to work with in subsequent steps. The result is stored
	// in the context with the key `SummaryOutputParamName`.
	out.AddCommand(commands.NewMediaSummaryJsonToStruct("convert-media-summary", SummaryOutputParamName))

	// Step 6: Extract detailed descriptions for each scene timestamp identified in the summary.
	// This command runs scene analysis jobs in parallel using a worker pool for efficiency.
	// The collected scene descriptions are stored in the context with the key `SceneOutputParamName`.
	sceneExtractor := commands.NewSceneExtractor("extract-media-scenes", m.genaiModel, m.sceneTemplate, m.numberOfWorkers)
	sceneExtractor.BaseCommand.OutputParamName = SceneOutputParamName
	out.AddCommand(sceneExtractor)

	// Step 7: Assemble the final, complete `model.Media` object. This command takes the
	// summary struct and the list of scene descriptions and combines them into a single,
	// unified data structure. The result is stored with the key `MediaOutputParamName`.
	out.AddCommand(commands.NewMediaAssembly("assemble-media-scenes", SummaryOutputParamName, SceneOutputParamName, MediaOutputParamName))

	// Step 8: Persist the final assembled media object to the main 'media' table in BigQuery.
	// This makes the structured data available for querying but does not include the vector embeddings yet.
	out.AddCommand(commands.NewMediaPersistToBigQuery(
		"write-to-bigquery",
		m.bigqueryClient,
		m.config.BigQueryDataSource.DatasetName,
		m.config.BigQueryDataSource.MediaTable, MediaOutputParamName))

	// Step 9: Clean up by deleting the temporary file from the Vertex AI File Service
	// to avoid incurring unnecessary storage costs.
	out.AddCommand(commands.NewMediaCleanup("cleanup-file-system", m.genaiClient))

	// Assign the fully constructed chain to the workflow instance.
	m.chain = out
}

// NewMediaReaderPipeline is the constructor for the MediaReaderWorkflow. It sets up
// all dependencies, compiles the prompt templates, and initializes the command chain.
//
// Inputs:
//   - config: The application's overall configuration.
//   - serviceClients: A struct containing initialized clients for GCP services.
//   - agentModelName: The name of the Vertex AI agent model config to use (e.g., "creative-flash").
//
// Returns:
//   - A pointer to a newly created and fully initialized MediaReaderWorkflow.
func NewMediaReaderPipeline(
	config *cloud.Config,
	serviceClients *cloud.ServiceClients,
	agentModelName string) *MediaReaderWorkflow {

	// Parse the summary prompt template from the configuration file.
	summaryTemplate, err := template.New("summary-template").Parse(config.PromptTemplates.SummaryPrompt)
	if err != nil {
		panic(err) // Panic on failure, as the app cannot run without valid templates.
	}
	// Parse the scene extraction prompt template.
	sceneTemplate, err := template.New("scene-template").Parse(config.PromptTemplates.ScenePrompt)
	if err != nil {
		panic(err)
	}

	// Create the MediaReaderWorkflow instance with all its dependencies.
	pipeline := &MediaReaderWorkflow{
		BaseCommand:     *cor.NewBaseCommand("media-reader-pipeline"),
		config:          config,
		bigqueryClient:  serviceClients.BiqQueryClient,
		genaiClient:     serviceClients.GenAIClient,
		genaiModel:      serviceClients.AgentModels[agentModelName],
		storageClient:   serviceClients.StorageClient,
		numberOfWorkers: config.Application.ThreadPoolSize,
		summaryTemplate: summaryTemplate,
		sceneTemplate:   sceneTemplate,
	}
	// Build the command chain for the new pipeline instance.
	pipeline.initializeChain()
	return pipeline
}
