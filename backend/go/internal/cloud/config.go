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

// Package cloud defines the data structures for application configuration,
// loaded from TOML files. It provides a structured way to manage settings
// for various components, including Google Cloud services, AI models,
// Pub/Sub topics, and prompt templates.
//
// This file centralizes all configuration-related structs, making it easy
// to understand and manage the application's configurable parameters.
//
// Structs:
//   - BigQueryDataSource: Configuration for BigQuery dataset and tables.
//   - PromptTemplates: Holds the text templates for prompts sent to GenAI models.
//   - VertexAiEmbeddingModel: Configuration for a Vertex AI embedding model.
//   - VertexAiLLMModel: Configuration for a Vertex AI Large Language Model (LLM).
//   - TopicSubscription: Configuration for a single Pub/Sub topic subscription.
//   - Storage: Configuration for Google Cloud Storage buckets.
//   - Category: Defines a media category and its associated LLM overrides.
//   - Config: The top-level struct that aggregates all other configuration structs.
//
// Functions:
//   - NewConfig: A constructor that initializes a new Config object with empty maps.
package cloud

import "google.golang.org/genai"

// DefaultSafetySettings defines the default content safety thresholds for GenAI models.
// These settings are configured to be non-restrictive, allowing all content categories
// (Dangerous Content, Harassment, Hate Speech, Sexually Explicit) to pass through without
// being blocked. This is a common setup for internal or controlled environments where
// the input data is trusted.
var DefaultSafetySettings = []*genai.SafetySetting{
	{
		Category:  genai.HarmCategoryDangerousContent,
		Threshold: genai.HarmBlockThresholdBlockNone,
	},
	{
		Category:  genai.HarmCategoryHarassment,
		Threshold: genai.HarmBlockThresholdBlockNone,
	},
	{
		Category:  genai.HarmCategoryHateSpeech,
		Threshold: genai.HarmBlockThresholdBlockNone,
	},
	{
		Category:  genai.HarmCategorySexuallyExplicit,
		Threshold: genai.HarmBlockThresholdBlockNone,
	},
}

// BigQueryDataSource represents the configuration for a BigQuery data source.
type BigQueryDataSource struct {
	DatasetName    string `toml:"dataset"`         // The name of the BigQuery dataset.
	MediaTable     string `toml:"media_table"`     // The name of the BigQuery table containing media information.
	EmbeddingTable string `toml:"embedding_table"` // The name of the BigQuery table containing embedding vectors.
}

// PromptTemplates holds the templates for different types of prompts.
type PromptTemplates struct {
	SummaryPrompt string `toml:"summary"` // The template for generating summaries.
	ScenePrompt   string `toml:"scene"`   // The template for generating scene descriptions.
}

// VertexAiEmbeddingModel represents the configuration for a Vertex AI embedding model.
type VertexAiEmbeddingModel struct {
	Model                string `toml:"model"`                   // The name of the Vertex AI embedding model.
	MaxRequestsPerMinute int    `toml:"max_requests_per_minute"` // The maximum number of requests allowed per minute.
}

// VertexAiLLMModel represents the configuration for a Vertex AI large language model (LLM).
type VertexAiLLMModel struct {
	Model              string  `toml:"model"`               // The name of the Vertex AI LLM.
	SystemInstructions string  `toml:"system_instructions"` // The system instructions for the LLM.
	Temperature        float32 `toml:"temperature"`         // The temperature parameter for the LLM.
	TopP               float32 `toml:"top_p"`               // The top_p parameter for the LLM.
	TopK               float32 `toml:"top_k"`               // The top_k parameter for the LLM.
	MaxTokens          int32   `toml:"max_tokens"`          // The maximum number of tokens for the LLM output.
	OutputFormat       string  `toml:"output_format"`       // The desired output format for the LLM.
	EnableGoogle       bool    `toml:"enable_google"`       // Whether to enable Google Search for the LLM.
	RateLimit          int     `toml:"rate_limit"`          // The rate limit for the LLM in requests per second.
}

// TopicSubscription represents the configuration for a Pub/Sub topic subscription.
type TopicSubscription struct {
	Name             string `toml:"name"`               // The name of the Pub/Sub subscription.
	DeadLetterTopic  string `toml:"dead_letter_topic"`  // The name of the dead-letter topic for the subscription.
	TimeoutInSeconds int    `toml:"timeout_in_seconds"` // The timeout for the subscription in seconds.
}

// Storage represents the configuration for storage buckets.
type Storage struct {
	HiResInputBucket   string `toml:"high_res_input_bucket"` // The name of the bucket for high-resolution input files.
	LowResOutputBucket string `toml:"low_res_output_bucket"` // The name of the bucket for low-resolution output files.
	GCSFuseMountPoint  string `toml:"gcs_fuse_mount_point"`  // The mount point for GCS FUSE.
}

// Category defines a specific type of media and allows for overriding LLM behaviors
// such as system instructions or prompt templates for that category.
type Category struct {
	Name               string `toml:"name"`                // The user-friendly name of the category (e.g., "Trailer").
	Definition         string `toml:"definition"`          // A short description of what the category represents.
	SystemInstructions string `toml:"system_instructions"` // Optional override for LLM system instructions for this category.
	Summary            string `toml:"summary"`             // Optional override for the summary prompt template.
	Scene              string `toml:"scene"`               // Optional override for the scene extraction prompt template.
}

// Config represents the overall configuration for the application, loaded from TOML files.
// It acts as the root container for all other configuration structs.
type Config struct {
	// Application holds general application settings.
	Application struct {
		Name                      string `toml:"name"`                         // The name of the application.
		GoogleProjectId           string `toml:"google_project_id"`            // The Google Cloud project ID.
		GoogleLocation            string `toml:"location"`                     // The Google Cloud location.
		ThreadPoolSize            int    `toml:"thread_pool_size"`             // The size of the worker pool for parallel processing tasks.
		SignerServiceAccountEmail string `toml:"signer_service_account_email"` // The service account email used for signing GCS URLs.
	} `toml:"application"`
	Storage            Storage                           `toml:"storage"`               // Storage configuration.
	BigQueryDataSource BigQueryDataSource                `toml:"big_query_data_source"` // BigQuery data source configuration.
	PromptTemplates    PromptTemplates                   `toml:"prompt_templates"`      // Prompt templates configuration.
	TopicSubscriptions map[string]TopicSubscription      `toml:"topic_subscriptions"`   // A map of Pub/Sub topic subscriptions, keyed by a logical name (e.g., "HiResTopic").
	EmbeddingModels    map[string]VertexAiEmbeddingModel `toml:"embedding_models"`      // A map of Vertex AI embedding models, keyed by a logical name (e.g., "multi-lingual").
	AgentModels        map[string]VertexAiLLMModel       `toml:"agent_models"`          // A map of Vertex AI LLM models, keyed by a logical name (e.g., "creative-flash").
	Categories         map[string]Category               `toml:"categories"`            // A map of media categories, keyed by a logical name (e.g., "trailer").
}

// NewConfig is a constructor function that creates a new, initialized Config instance.
// It's important to initialize the maps within the struct to avoid nil pointer panics
// when the configuration loader tries to populate them.
//
// Outputs:
//   - *Config: A pointer to a new Config struct with its map fields initialized.
func NewConfig() *Config {
	return &Config{
		TopicSubscriptions: make(map[string]TopicSubscription),
		EmbeddingModels:    make(map[string]VertexAiEmbeddingModel),
		AgentModels:        make(map[string]VertexAiLLMModel),
		Categories:         make(map[string]Category),
	}
}
