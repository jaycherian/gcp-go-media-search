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

// Package main contains the setup and initialization logic for the application's state.
// This file is responsible for creating and managing a centralized state manager
// that holds all shared dependencies, such as configuration, Google Cloud service clients,
// and application-level services for search and media handling.
//
// It ensures that the application is configured correctly based on the environment,
// initializes all necessary clients (Storage, BigQuery, IAM, etc.), and starts
// background processes like Pub/Sub listeners and the embedding generator workflow.
//
// Functions:
//   - SetupOS: Configures necessary environment variables for the application,
//     pointing to the correct configuration files.
//   - GetConfig: A singleton function that loads the application's configuration
//     from TOML files. It ensures the configuration is loaded only once.
//   - InitState: The core initialization function that creates all service clients,
//     configures application services (MediaService, SearchService), and starts
//     background workflows and Pub/Sub listeners.
package main

import (
	"context"
	"log"
	"os"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/services"
	"github.com/jaycherian/gcp-go-media-search/internal/core/workflow"
)

// StateManager holds all the shared dependencies for the application, acting as a
// centralized container for service clients and configurations. This avoids the
// need for global variables and makes dependency management cleaner.
type StateManager struct {
	config        *cloud.Config
	cloud         *cloud.ServiceClients
	searchService *services.SearchService
	mediaService  *services.MediaService
}

// state is a package-level variable that holds the single instance of StateManager.
var state = &StateManager{}

// SetupOS sets the necessary environment variables that the configuration loader
// uses to find the correct TOML files.
//
// This function sets the prefix for the configuration directory and specifies
// the runtime environment (e.g., "local", "test", "prod"), allowing for
// environment-specific overrides of the base configuration.
//
// Outputs:
//   - error: An error if setting any of the environment variables fails.
func SetupOS() (err error) {
	// Set the directory where configuration files are located.
	err = os.Setenv(cloud.EnvConfigFilePrefix, "configs")
	if err != nil {
		return err
	}
	// Set the current runtime environment to "local". The config loader will
	// look for a ".env.local.toml" file to override base settings.
	err = os.Setenv(cloud.EnvConfigRuntime, "local")
	return err
}

// GetConfig provides a singleton instance of the application configuration.
// It ensures that the configuration is loaded from the file system only once.
// On the first call, it sets up the OS environment and loads the configuration
// from the TOML files. Subsequent calls return the cached configuration.
//
// Outputs:
//   - *cloud.Config: A pointer to the loaded application configuration struct.
func GetConfig() *cloud.Config {
	// If the config has not been loaded yet...
	if state.config == nil {
		// Set up the environment variables required for config loading.
		err := SetupOS()
		if err != nil {
			log.Fatalf("failed to setup os for testing: %v\n", err)
		}
		// Create a new, empty config struct.
		config := cloud.NewConfig()
		// Load the configuration from the .toml files into the struct.
		cloud.LoadConfig(&config)
		// Store the loaded config in the state manager.
		state.config = config
	}
	// Return the cached config.
	return state.config
}

// InitState initializes the entire application state.
// It orchestrates the creation of all necessary services and clients based on the
// application configuration and wires them together.
//
// Inputs:
//   - ctx: The root context.Context for the application, used for managing
//     the lifecycle of client connections and background processes.
//
// This function performs the following steps:
//  1. Loads the application configuration.
//  2. Initializes all Google Cloud service clients (Storage, Pub/Sub, GenAI, BigQuery, IAM).
//  3. Instantiates the application-specific services (SearchService, MediaService)
//     with the required client dependencies.
//  4. Starts background workflows, such as the media embedding generator.
//  5. Sets up and starts the Pub/Sub listeners for processing GCS events.
func InitState(ctx context.Context) {
	// Get the application configuration.
	config := GetConfig()

	// Initialize all the base Google Cloud service clients.
	cloudClients, err := cloud.NewCloudServiceClients(ctx, config)
	if err != nil {
		panic(err)
	}

	// Specifically initialize the IAM credentials client, required for signing URLs.
	iamClient, err := credentials.NewIamCredentialsClient(ctx)
	if err != nil {
		panic(err)
	}
	// Add the IAM client to our set of cloud clients.
	cloudClients.IAMClient = iamClient

	// Store the initialized clients in the global state.
	state.cloud = cloudClients

	// Get BigQuery dataset and table names from the config for easier access.
	datasetName := config.BigQueryDataSource.DatasetName
	mediaTableName := config.BigQueryDataSource.MediaTable
	embeddingTableName := config.BigQueryDataSource.EmbeddingTable

	// Initialize the SearchService with its dependencies.
	state.searchService = &services.SearchService{
		BigqueryClient: cloudClients.BiqQueryClient,
		EmbeddingModel: cloudClients.EmbeddingModels["multi-lingual"],
		ModelName:      config.EmbeddingModels["multi-lingual"].Model,
		DatasetName:    datasetName,
		MediaTable:     mediaTableName,
		EmbeddingTable: embeddingTableName,
	}

	// Initialize the MediaService with its dependencies.
	state.mediaService = &services.MediaService{
		BigqueryClient: cloudClients.BiqQueryClient,
		StorageClient:  cloudClients.StorageClient, // Pass the storage client here
		IAMClient:      cloudClients.IAMClient,
		SignerEmail:    config.Application.SignerServiceAccountEmail,
		DatasetName:    datasetName,
		MediaTable:     mediaTableName,
	}

	// Create and start the background workflow for generating embeddings for new media.
	embeddingGenerator := workflow.NewMediaEmbeddingGeneratorWorkflow(config, cloudClients)
	embeddingGenerator.StartTimer()

	// Configure and start the Pub/Sub listeners that react to GCS bucket events.
	SetupListeners(config, cloudClients, ctx)

}
