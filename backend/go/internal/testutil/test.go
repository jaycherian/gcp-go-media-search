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

// Package test provides utility functions and mock data to support the application's
// test suite. It helps in setting up a consistent test environment, loading
// test-specific configurations, and providing sample data for workflows and services.
package test

import (
	"log"
	"os"
	"testing"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
)

// StateManager acts as a simple in-memory cache for the application configuration
// during test runs. This prevents the need to reload configuration files for every
// test, speeding up the test suite.
type StateManager struct {
	config *cloud.Config
}

// state is a package-level variable that holds the singleton instance of StateManager,
// ensuring that the configuration is loaded only once per test run.
var state = &StateManager{}

// HandleErr is a simple test helper function that checks if an error is not nil.
// If an error exists, it fails the test immediately by calling t.Errorf.
// This is a convenience function to reduce boilerplate error-checking code in tests.
//
// Inputs:
//   - err: The error to check.
//   - t: The *testing.T object from the current test.
func HandleErr(err error, t *testing.T) {
	if err != nil {
		t.Errorf("Error reading config file: %v", err)
	}
}

// GetTestHighResMessageText returns a hardcoded JSON string that simulates a
// Pub/Sub notification message from Google Cloud Storage for a file finalized
// in the "high-resolution" bucket. This mock data is used to test the media
// resize workflow trigger.
//
// Returns:
//   - A string containing the JSON payload of a GCS notification.
func GetTestHighResMessageText() string {
	return `{
  "kind": "storage#object",
  "id": "media_high_res_resources/test-trailer-001.mp4/1728615848664286",
  "selfLink": "https://www.googleapis.com/storage/v1/b/media_high_res_resources/o/test-trailer-001.mp4",
  "name": "test-trailer-001.mp4",
  "bucket": "media_high_res_resources",
  "generation": "1728615848664286",
  "metageneration": "1",
  "contentType": "video/mp4",
  "timeCreated": "2024-10-11T03:04:08.672Z",
  "updated": "2024-10-11T03:04:08.672Z",
  "storageClass": "STANDARD",
  "timeStorageClassUpdated": "2024-10-11T03:04:08.672Z",
  "size": "259348037",
  "md5Hash": "67c1rAU+1RYZzK5zp8iBkA==",
  "mediaLink": "https://storage.googleapis.com/download/storage/v1/b/media_high_res_resources/o/test-trailer-001.mp4?generation=1728615848664286&alt=media",
  "metadata": { "touch": "18" },
  "crc32c": "IYeSTw==",
  "etag": "CN658+yrhYkDEAE="
	}`
}

// GetTestLowResMessageText returns a hardcoded JSON string that simulates a
// Pub/Sub notification message for a file finalized in the "low-resolution"
// bucket. This mock data is used to test the media ingestion and analysis
// workflow trigger.
//
// Returns:
//   - A string containing the JSON payload of a GCS notification.
func GetTestLowResMessageText() string {
	return `{
  "kind": "storage#object",
  "id": "media_low_res_resources/test-trailer-001.mp4/1728615848664286",
  "selfLink": "https://www.googleapis.com/storage/v1/b/media_low_res_resources/o/test-trailer-001.mp4",
  "name": "test-trailer-001.mp4",
  "bucket": "media_low_res_resources",
  "generation": "1728615848664286",
  "metageneration": "1",
  "contentType": "video/mp4",
  "timeCreated": "2024-10-11T03:04:08.672Z",
  "updated": "2024-10-11T03:04:08.672Z",
  "storageClass": "STANDARD",
  "timeStorageClassUpdated": "2024-10-11T03:04:08.672Z",
  "size": "259348037",
  "md5Hash": "67c1rAU+1RYZzK5zp8iBkA==",
  "mediaLink": "https://storage.googleapis.com/download/storage/v1/b/media_low_res_resources/o/test-trailer-001.mp4?generation=1728615848664286&alt=media",
  "metadata": { "touch": "18" },
  "crc32c": "IYeSTw==",
  "etag": "CN658+yrhYkDEAE="
}
`
}

// SetupOS configures the necessary environment variables that the configuration
// loader (`cloud.LoadConfig`) depends on. By setting these variables, we can
// direct the loader to use the test-specific configuration files (e.g.,
// `configs/.env.test.toml`) instead of production or development ones.
//
// Returns:
//   - An error if setting any environment variable fails.
func SetupOS() (err error) {
	// Set the directory where the configuration files are located.
	err = os.Setenv(cloud.EnvConfigFilePrefix, "configs")
	if err != nil {
		return err
	}
	// Set the runtime environment identifier to "test". This causes the loader
	// to look for a file named ".env.test.toml" for overrides.
	err = os.Setenv(cloud.EnvConfigRuntime, "test")
	return err
}

// GetConfig is a singleton accessor for the test configuration.
// It ensures that the configuration is loaded from TOML files only once and
// is cached in the package-level `state` variable for subsequent calls.
// This is the primary way tests should retrieve their configuration.
//
// Returns:
//   - A pointer to the loaded and cached cloud.Config struct.
func GetConfig() *cloud.Config {
	// Check if the config is already cached.
	if state.config == nil {
		// If not cached, set up the OS environment for the test configuration.
		err := SetupOS()
		if err != nil {
			log.Fatalf("failed to setup environment for test: %v\n", err)
		}
		// Create a new, empty config struct.
		config := cloud.NewConfig()
		// Load the configuration from the TOML files into the struct.
		// `LoadConfig` handles the hierarchical loading (base file + test override).
		cloud.LoadConfig(&config)
		// Cache the loaded config in our state manager.
		state.config = config
	}
	// Return the cached configuration.
	return state.config
}
