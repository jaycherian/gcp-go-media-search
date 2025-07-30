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

// Package cloud provides components for interacting with Google Cloud services.
// This file contains general-purpose utility functions that support the cloud package.
// These helpers cover tasks like hierarchical configuration loading, file system checks,
// and resilient interaction with the Generative AI API.
//
// Functions:
//   - fileExists: A simple helper to check if a file exists.
//   - LoadConfig: Implements a hierarchical configuration loader. It first reads a base
//     configuration file and then overwrites values with a second, environment-specific
//     file (e.g., .env.local.toml, .env.test.toml). The environment is determined by
//     an environment variable.
//   - GenerateMultiModalResponse: A wrapper for making calls to the GenAI model. It includes
//     a retry mechanism to handle transient errors and integrates with OpenTelemetry to
//     record metrics for token usage and retries.
//   - NewTextPart, NewFileData: Simple factory functions for creating genai.Part objects,
//     improving code readability when constructing multi-modal prompts.
package cloud

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"go.opentelemetry.io/otel/metric"

	"github.com/BurntSushi/toml"
	"google.golang.org/genai"
)

// Cloud Constants define key strings and values used throughout the package,
// primarily for configuration loading and API interaction policies.
const (
	ConfigFileBaseName  = ".env"              // The base name for configuration files (e.g., ".env.toml").
	ConfigFileExtension = ".toml"             // The file extension for configuration files.
	ConfigSeparator     = "."                 // The separator used in config file names (e.g., ".env.local.toml").
	EnvConfigFilePrefix = "GCP_CONFIG_PREFIX" // The environment variable for specifying the config directory.
	EnvConfigRuntime    = "GCP_RUNTIME"       // The environment variable for specifying the runtime context (e.g., "local", "test", "prod").
	MaxRetries          = 3                   // The maximum number of times to retry a failed API call.
)

// fileExists checks if a file or directory exists at the given path.
//
// Inputs:
//   - in: The path to the file or directory as a string.
//
// Outputs:
//   - bool: Returns true if the file exists, and false if it does not.
func fileExists(in string) bool {
	// os.Stat returns information about the file. If it returns an error,
	// the file likely doesn't exist.
	_, err := os.Stat(in)
	// We specifically check if the error is `os.ErrNotExist`.
	// If it is, we know the file is missing and return false. Otherwise, it exists.
	return !errors.Is(err, os.ErrNotExist)
}

// LoadConfig provides a hierarchical configuration loading mechanism. It first loads a
// base configuration file and then merges or overwrites its values with an environment-specific
// configuration file. The paths and environment are determined by environment variables.
//
// Inputs:
//   - baseConfig: An interface{} representing a pointer to the target configuration struct
//     that will be populated from the TOML files.
func LoadConfig(baseConfig interface{}) {
	// Print all environment variables to the log.
	fmt.Println("Environment Variables:")
	for _, env := range os.Environ() {
		fmt.Println(env)
	}

	// Read the directory path for config files from an environment variable.
	configurationFilePrefix := os.Getenv(EnvConfigFilePrefix)
	// Ensure the prefix ends with a path separator if it's not empty.
	if len(configurationFilePrefix) > 0 && !strings.HasSuffix(configurationFilePrefix, string(os.PathSeparator)) {
		configurationFilePrefix = configurationFilePrefix + string(os.PathSeparator)
	}

	// Read the runtime environment (e.g., "local", "test") from an environment variable.
	// Default to "test" if the variable is not set.
	runtimeEnvironment := os.Getenv(EnvConfigRuntime)
	if runtimeEnvironment == "" {
		runtimeEnvironment = "test"
	}

	// Construct the path for the base configuration file (e.g., "configs/.env.toml").
	baseConfigFileName := configurationFilePrefix + ConfigFileBaseName + ConfigFileExtension
	fmt.Printf("Base Configuration File: %s\n", baseConfigFileName)

	// Construct the path for the environment-specific override file (e.g., "configs/.env.test.toml").
	envConfigFileName := configurationFilePrefix + ConfigFileBaseName + ConfigSeparator + runtimeEnvironment + ConfigFileExtension
	fmt.Printf("Environment Configuration File: %s\n", envConfigFileName)

	// If the base configuration file exists, decode it into the baseConfig struct.
	if fileExists(baseConfigFileName) {
		_, err := toml.DecodeFile(baseConfigFileName, baseConfig)
		if err != nil {
			log.Fatalf("failed to decode base configuration file %s with error: %s", baseConfigFileName, err)
		}
	}

	// If the environment-specific configuration file exists, decode it.
	// Any values in this file will overwrite the values from the base config.
	if fileExists(envConfigFileName) {
		_, err := toml.DecodeFile(envConfigFileName, baseConfig)
		if err != nil {
			log.Fatalf("failed to decode environment configuration file: %s with error: %s", envConfigFileName, err)
		}
	}
}

// GenerateMultiModalResponse is a helper function for executing multi-modal requests
// against a Generative AI model. It includes logic for retries and telemetry.
//
// Inputs:
//   - ctx: The context for the request, which controls cancellation and tracing.
//   - inputTokenCounter: An OpenTelemetry counter for prompt tokens used.
//   - outputTokenCounter: An OpenTelemetry counter for response tokens generated.
//   - retryCounter: An OpenTelemetry counter for tracking the number of retries.
//   - tryCount: The current attempt number for this request (starts at 0).
//   - model: The rate-limited, quota-aware generative model to use.
//   - parts: A variadic slice of `genai.Part` (e.g., text, images, video) that form the prompt.
//
// Outputs:
//   - string: The concatenated text content from the model's response.
//   - error: An error if the request fails after all retries.
func GenerateMultiModalResponse(
	ctx context.Context,
	inputTokenCounter metric.Int64Counter,
	outputTokenCounter metric.Int64Counter,
	retryCounter metric.Int64Counter,
	tryCount int,
	model *QuotaAwareGenerativeAIModel,
	content []*genai.Content) (value string, err error) {
	// Make the request to the generative model.
	resp, err := model.GenerateContent(ctx, content)

	// If there's an error, check if we can retry.
	if err != nil {
		if tryCount < MaxRetries {
			// If we haven't reached the max retry count, increment the retry counter
			// and recursively call this function to try again.
			retryCounter.Add(ctx, 1)
			return GenerateMultiModalResponse(ctx, inputTokenCounter, outputTokenCounter, retryCounter, tryCount+1, model, content)
		} else {
			// If max retries have been reached, return the error.
			return "", err
		}
	}
	// Record the token counts for both the prompt and the generated candidates.
	inputTokenCounter.Add(ctx, int64(resp.UsageMetadata.PromptTokenCount))
	outputTokenCounter.Add(ctx, int64(resp.UsageMetadata.CandidatesTokenCount))

	// If the request was successful, process the response.
	value = ""
	// The response can have multiple candidates; iterate through them.
	for _, candidate := range resp.Candidates {
		if candidate.Content != nil {
			// Each candidate's content can have multiple parts; iterate and concatenate them.
			for _, part := range candidate.Content.Parts {
				value += fmt.Sprint(part.Text)
			}
		}
	}
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimSuffix(value, "```")
	return value, nil
}

// NewTextPart is a simple factory function (delegate) for creating a text part.
// This improves readability by providing a clear, named function for this action.
//
// Inputs:
//   - in: The string content for the text part.
//
// Outputs:
//   - genai.Part: A `genai.Part` containing the text.
//
// Muziris Change
func NewTextPart(in string) []*genai.Content {
	return genai.Text(in)
}

// NewFileData is a simple factory function (delegate) for creating a file data part.
// This improves readability by abstracting the creation of this struct.
//
// Inputs:
//   - in: The URI of the file (e.g., a GCS path).
//   - mimeType: The MIME type of the file (e.g., "video/mp4").
//
// Outputs:
//   - genai.Part: A `genai.Part` containing the file data.
//
// Muziris Change
func NewFileData(in string, mimeType string) genai.FileData {
	return genai.FileData{FileURI: in, MIMEType: mimeType}
}
