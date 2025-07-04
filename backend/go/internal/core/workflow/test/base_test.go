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

// Package workflow_test contains integration tests for the core application workflows.
// This file, `base_test.go`, provides the foundational setup and teardown logic
// for all tests within this package. It uses the special `TestMain` function,
// which acts as the main entry point for the test suite, allowing for global
// initialization of resources like configuration, service clients, and telemetry.
// These shared resources are then available to all other test files in this package.
package workflow_test

import (
	"context"
	"os"
	"testing"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/telemetry"
	test "github.com/jaycherian/gcp-go-media-search/internal/testutil"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
)

// Declare global variables to hold shared resources for the test suite.
// These are initialized once in TestMain and can be accessed by other
// test functions in the `workflow_test` package.
var (
	err          error
	cloudClients *cloud.ServiceClients // Holds all initialized Google Cloud service clients.
	ctx          context.Context       // The root context for all tests in the suite.
	config       *cloud.Config         // The application configuration loaded from test files.
)

// Constants and global tracers/loggers for telemetry.
const tName = "cloud.google.com/media/tests/workflow"

var (
	tracer = otel.Tracer(tName)
	logger = otelslog.NewLogger(tName)
)

// TestMain is a special function that Go's testing framework executes before any other
// tests in this package. It allows for setting up shared state and performing
// teardown actions after all tests have run.
//
// Inputs:
//   - m: A pointer to testing.M, which provides access to the test suite and allows
//     running the tests via m.Run().
func TestMain(m *testing.M) {
	// ---- Setup Phase ----

	// Create a root context with a cancellation function. This context will be used for all
	// initializations and passed down to tests. `defer cancel()` ensures that the context
	// is canceled and associated resources are released when TestMain exits.
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Load application configuration from test-specific files (`.env.test.toml`).
	config = test.GetConfig()

	// Initialize structured logging.
	telemetry.SetupLogging()

	// Initialize OpenTelemetry for distributed tracing and metrics. This returns a `shutdown`
	// function that must be called later to flush any buffered telemetry data.
	shutdown, err := telemetry.SetupOpenTelemetry(ctx, config)
	if err != nil {
		// If telemetry fails to set up, we cannot proceed. Panic will stop the test run.
		panic(err)
	}

	// Initialize all the Google Cloud service clients (Storage, BigQuery, etc.)
	// using the loaded configuration. These clients are stored in the global `cloudClients`
	// variable, making them accessible to all tests in the package.
	cloudClients, err = cloud.NewCloudServiceClients(ctx, config)
	if err != nil {
		panic(err)
	}
	// `defer` the closing of cloud clients to ensure connections are terminated cleanly
	// after all tests have run.
	defer cloudClients.Close()

	logger.Info("completed test setup")

	// ---- Execution Phase ----

	// m.Run() executes all the other TestXxx functions in the package. The result is
	// an exit code that indicates whether the tests passed or failed.
	exitCode := m.Run()

	// ---- Teardown Phase ----

	// Gracefully shut down the OpenTelemetry provider, ensuring all telemetry data is exported.
	if err := shutdown(ctx); err != nil {
		logger.Error("failed to shutdown telemetry", "error", err)
	}

	// Exit the test process with the code from the test run.
	os.Exit(exitCode)
}
