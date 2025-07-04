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

// Package cor (Chain of Responsibility) provides the fundamental building blocks
// for creating workflows. This file defines `BaseCommand`, the default and
// foundational implementation of the `Command` interface.
//
// Every command in the system should embed `BaseCommand` to inherit common
// functionality, reducing boilerplate code. This includes:
//   - A name for identification in logs and telemetry.
//   - Built-in OpenTelemetry tracing (`Tracer`) and metrics (`Meter`, counters).
//   - Default logic for handling input and output parameter keys from the context,
//     which is essential for the "piping" mechanism in a `BaseChain`.
package cor

import (
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// BaseCommand is the default implementation of the Command interface. It provides
// core functionality that all concrete commands can reuse.
type BaseCommand struct {
	Name            string              // A unique name for the command, used for tracing and metrics.
	InputParamName  string              // The key to look up this command's primary input in the context.
	OutputParamName string              // The key to store this command's primary output in the context.
	Tracer          trace.Tracer        // An OpenTelemetry tracer for creating spans.
	Meter           metric.Meter        // An OpenTelemetry meter for creating metrics.
	SuccessCounter  metric.Int64Counter // A metric counter that increments on successful execution.
	ErrorCounter    metric.Int64Counter // A metric counter that increments when an error occurs.
}

// NewBaseCommand is the constructor for BaseCommand. It initializes a command
// with a name and sets up all the necessary OpenTelemetry instrumentation.
//
// Inputs:
//   - name: The string name for this command.
//
// Outputs:
//   - *BaseCommand: A pointer to the newly instantiated command.
func NewBaseCommand(name string) *BaseCommand {
	// Get a meter from the global OpenTelemetry provider using a standard namespace.
	meter := otel.Meter("github.com/GoogleCloudPlatform/solutions/media")

	// Create a counter for successful operations, namespaced with the command name.
	successCounter, err := meter.Int64Counter(fmt.Sprintf("%s.counter.success", name))
	if err != nil {
		log.Printf("error creating success counter for command '%s': %v\n", name, err)
	}
	// Create a counter for failed operations.
	errorCounter, err := meter.Int64Counter(fmt.Sprintf("%s.counter.error", name))
	if err != nil {
		log.Printf("error creating error counter for command '%s': %v\n", name, err)
	}

	// The struct is initialized with the name and all OTel components.
	return &BaseCommand{
		Name:           name,
		Tracer:         otel.Tracer(name), // Get a tracer from the global provider.
		Meter:          meter,
		SuccessCounter: successCounter,
		ErrorCounter:   errorCounter,
	}
}

// GetName returns the name of the command.
func (c *BaseCommand) GetName() string {
	return c.Name
}

// IsExecutable provides a default implementation for checking if a command can run.
// It performs a basic safety check to ensure the context is valid and that the
// expected input data (identified by the input parameter key) exists in the context.
//
// Inputs:
//   - context: The shared `Context` for the workflow.
//
// Outputs:
//   - bool: True if the command is ready to execute, false otherwise.
func (c *BaseCommand) IsExecutable(context Context) bool {
	return context != nil && context.Get(c.GetInputParam()) != nil && context.GetContext() != nil
}

// GetInputParam returns the key for the command's primary input data.
// If a custom `InputParamName` has not been set, it defaults to `CtxIn`.
// This default is crucial for the `BaseChain`'s ability to pipe the output
// of one command to the input of the next.
func (c *BaseCommand) GetInputParam() string {
	if len(c.InputParamName) == 0 {
		return CtxIn
	}
	return c.InputParamName
}

// GetOutputParam returns the key where the command will store its primary output data.
// If a custom `OutputParamName` has not been set, it defaults to `CtxOut`.
// This allows the `BaseChain` to find the result and prepare it for the next command.
func (c *BaseCommand) GetOutputParam() string {
	if len(c.OutputParamName) == 0 {
		return CtxOut
	}
	return c.OutputParamName
}

// GetTracer returns the OpenTelemetry Tracer for this command.
func (c *BaseCommand) GetTracer() trace.Tracer {
	return c.Tracer
}

// GetMeter returns the OpenTelemetry Meter for this command.
func (c *BaseCommand) GetMeter() metric.Meter {
	return c.Meter
}

// GetSuccessCounter returns the success metric counter for this command.
func (c *BaseCommand) GetSuccessCounter() metric.Int64Counter {
	return c.SuccessCounter
}

// GetErrorCounter returns the error metric counter for this command.
func (c *BaseCommand) GetErrorCounter() metric.Int64Counter {
	return c.ErrorCounter
}
