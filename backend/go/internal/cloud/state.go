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
// This file is central to the application's architecture, as it's responsible for
// initializing and holding all the client objects needed to communicate with
// various Google Cloud services. It acts as a dependency injection container,
// creating a single, shared `ServiceClients` struct that can be passed throughout
// the application.
//
// Logic Flow:
//  1. The `NewCloudServiceClients` function is called at application startup.
//  2. It takes the application's configuration (`Config`) and a `context.Context`.
//  3. It iteratively initializes clients for Storage, Pub/Sub, GenAI, and BigQuery.
//  4. It then reads the configuration to create and configure specific service wrappers,
//     like Pub/Sub listeners and AI models, storing them in maps.
//  5. All initialized clients and services are bundled into a single `ServiceClients` struct.
//  6. This struct is then used by other parts of the application (like API handlers and workflows)
//     to perform their tasks.
//
// Structs:
//   - ServiceClients: A container struct holding all initialized Google Cloud service clients
//     and service wrappers, acting as a central state manager for external connections.
//
// Functions:
//   - Close: A convenience method to gracefully shut down all client connections.
//   - NewCloudServiceClients: A factory function that creates and configures all necessary
//     Google Cloud clients based on the application's configuration.
package cloud

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/bigquery"
	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"google.golang.org/genai"
)

// ServiceClients is a struct that acts as a central container for all the clients
// that interact with external Google Cloud services. This pattern is a form of
// dependency injection, making it easy to manage and share these client connections
// across the entire application.
type ServiceClients struct {
	StorageClient   *storage.Client                   // Client for Google Cloud Storage (GCS).
	PubsubClient    *pubsub.Client                    // Client for Google Cloud Pub/Sub.
	GenAIClient     *genai.Client                     // Client for Google's Generative AI services (Vertex AI).
	BiqQueryClient  *bigquery.Client                  // Client for Google Cloud BigQuery.
	IAMClient       *credentials.IamCredentialsClient // Client for IAM to sign things like GCS URLs.
	PubSubListeners map[string]*PubSubListener        // A map of active Pub/Sub listeners, keyed by a logical name from the config.
	//TODO: Do this later when we do step 2 embedding
	EmbeddingModels map[string]*genai.Models                // A map of configured GenAI embedding models, keyed by a logical name.
	AgentModels     map[string]*QuotaAwareGenerativeAIModel // A map of configured GenAI agent (LLM) models, keyed by a logical name.
}

// Close is a utility method to gracefully shut down all the active client connections.
// While client connections are typically managed by the application's root context,
// this method provides an explicit way to release resources, which is especially
// useful in tests or for controlled shutdowns.
func (c *ServiceClients) Close() {
	_ = c.StorageClient.Close()
	_ = c.PubsubClient.Close()
	//TODO: New library does not have a client close function
	//TODO _ = c.GenAIClient.Close()
	_ = c.BiqQueryClient.Close()
}

// NewCloudServiceClients is a factory function that initializes all required Google Cloud
// service clients based on the provided configuration. It serves as the main entry point
// for setting up the application's external dependencies.
//
// Inputs:
//   - ctx: The root context.Context for the application, used to manage the lifecycle of the clients.
//   - config: A pointer to the loaded application configuration (`Config`).
//
// Outputs:
//   - *ServiceClients: A pointer to the fully initialized ServiceClients struct.
//   - error: An error if any of the clients fail to initialize.
func NewCloudServiceClients(ctx context.Context, config *Config) (cloud *ServiceClients, err error) {
	// Create a new Google Cloud Storage client.
	sc, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	// Create a new Google Cloud Pub/Sub client for the specified project.
	pc, err := pubsub.NewClient(ctx, config.Application.GoogleProjectId)
	if err != nil {
		return nil, err
	}

	// Create a new Generative AI client using an API key for authentication.
	fmt.Print("The project ID is:", config.Application.GoogleProjectId)
	fmt.Print("The project location is", config.Application.GoogleLocation)
	gc, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  config.Application.GoogleProjectId,
		Location: config.Application.GoogleLocation,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		fmt.Print("Error creating genai client:", err)
		log.Printf("error creating genai client: %v", err)
		return nil, err
	}

	// Create a new Google Cloud BigQuery client.
	bc, err := bigquery.NewClient(ctx, config.Application.GoogleProjectId)
	if err != nil {
		return nil, err
	}

	// Iterate through the subscription configurations and create a PubSubListener for each one.
	// The command is initially set to `nil` because it will be attached later when the workflows are built.
	subscriptions := make(map[string]*PubSubListener)
	for subKey := range config.TopicSubscriptions {
		values := config.TopicSubscriptions[subKey]
		actual, err := NewPubSubListener(pc, values.Name, nil)
		if err != nil {
			return nil, err
		}
		subscriptions[subKey] = actual
	}

	// Iterate through the embedding model configurations and create a reference to each model.
	embeddingModels := make(map[string]*genai.Models)
	for embKey := range config.EmbeddingModels {
		//	embeddingModels[embKey] = gc.GenerativeModel(config.EmbeddingModels[embKey].Model)
		embeddingModels[embKey] = gc.Models
		fmt.Print("looping through embeddingmodels, this one is: \n", config.EmbeddingModels[embKey].Model)
	}

	// Iterate through the agent model configurations, create a generative model for each,
	// apply its specific settings (temperature, TopK, etc.), and wrap it in our
	// custom rate-limiting (`QuotaAware`) model.
	agentModels := make(map[string]*QuotaAwareGenerativeAIModel)
	for amKey := range config.AgentModels {
		values := config.AgentModels[amKey]
		fmt.Print("The value of amkey is: \n", amKey)
		fmt.Print("The content in Agnentmodel is \n", values)

		model := &genai.GenerateContentConfig{
			Temperature:       genai.Ptr[float32](values.Temperature),
			TopP:              genai.Ptr[float32](values.TopP),
			TopK:              genai.Ptr[float32](values.TopK),
			MaxOutputTokens:   values.MaxTokens,
			SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: values.SystemInstructions}}},
			SafetySettings:    DefaultSafetySettings,
			ResponseMIMEType:  values.OutputFormat,
			Tools:             []*genai.Tool{},
		}
		// 	model.SetTemperature(values.Temperature)
		// 	model.SetTopK(values.TopK)
		// 	model.SetTopP(values.TopP)
		// 	model.SetMaxOutputTokens(values.MaxTokens)
		// 	model.SystemInstruction = &genai.Content{
		// 		Parts: []genai.Part{genai.Text(values.SystemInstructions)},
		// 	}
		// 	// Apply the default safety settings and desired output format.
		// 	model.SafetySettings = DefaultSafetySettings
		// 	model.ResponseMIMEType = values.OutputFormat
		// 	model.Tools = []*genai.Tool{} // Initialize with no tools by default.

		// 	// Wrap the configured model with our rate limiter.
		wrappedAgent := NewQuotaAwareModel(model, values.Model, gc.Models, values.RateLimit)
		agentModels[amKey] = wrappedAgent
	}

	// Assemble the final ServiceClients struct with all the initialized clients and models.
	cloud = &ServiceClients{
		StorageClient:   sc,
		PubsubClient:    pc,
		GenAIClient:     gc,
		BiqQueryClient:  bc,
		PubSubListeners: subscriptions,
		EmbeddingModels: embeddingModels,
		AgentModels:     agentModels,
	}

	return cloud, err
}
