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
// for creating workflows. This file defines `BaseContext`, the default
// implementation of the `Context` interface.
//
// The `Context` is a critical component of the COR pattern. It acts as a shared
// "state machine" or a "property bag" that is passed through the entire chain
// of commands. Each command can read data from the context, perform its work,
// and then write its results back to the context for subsequent commands to use.
//
// This implementation includes:
//   - A map to hold arbitrary data (`data`).
//   - A map to collect errors from any command in the chain (`errors`).
//   - A slice to track temporary files created during the workflow, ensuring
//     they can be cleaned up at the end (`tempFiles`).
//   - A standard Go `context.Context` for handling cancellations, deadlines,
//     and passing request-scoped values like OpenTelemetry spans.
package cor

import (
	"context"
	"log"
	"os"
)

// BaseContext is the default implementation of the Context interface. It holds
// the shared state for a workflow execution.
type BaseContext struct {
	data      map[string]interface{} // A map to store arbitrary key-value data.
	errors    map[string]error       // A map to store errors, keyed by the command name that produced them.
	tempFiles []string               // A slice of paths to temporary files that need to be cleaned up.
	context   context.Context        // The standard Go context for cancellation and passing request-scoped values.
}

// NewBaseContext is the constructor for BaseContext.
// It initializes all the internal maps and slices to ensure they are ready for use.
//
// Outputs:
//   - Context: A new, empty context object.
func NewBaseContext() Context {
	return &BaseContext{
		data:      make(map[string]interface{}),
		errors:    make(map[string]error),
		tempFiles: make([]string, 0),
	}
}

// SetContext sets the underlying standard Go context. This is used by the
// BaseChain to manage the context for OpenTelemetry spans.
//
// Inputs:
//   - context: The standard `context.Context` to set.
func (c *BaseContext) SetContext(context context.Context) {
	c.context = context
}

// GetContext retrieves the underlying standard Go context.
//
// Outputs:
//   - context.Context: The currently set Go context.
func (c *BaseContext) GetContext() context.Context {
	return c.context
}

// Close is a cleanup method that should be called at the end of a workflow.
// It iterates through any temporary files tracked by the context and removes them
// from the filesystem.
func (c *BaseContext) Close() {
	// Clean up any temp files created along the way.
	for _, file := range c.GetTempFiles() {
		err := os.Remove(file)
		if err != nil {
			log.Printf("failed to remove temporary file '%s': %v\n", file, err)
		}
	}
}

// Add stores a key-value pair in the context's data map.
//
// Inputs:
//   - key: The string key to store the data under.
//   - value: The data (of any type) to store.
//
// Outputs:
//   - Context: The context instance, allowing for fluent method chaining.
func (c *BaseContext) Add(key string, value interface{}) Context {
	c.data[key] = value
	return c
}

// AddTempFile adds a file path to the list of temporary files that need cleanup.
//
// Inputs:
//   - file: The string path to the temporary file.
func (c *BaseContext) AddTempFile(file string) {
	c.tempFiles = append(c.tempFiles, file)
}

// GetTempFiles returns the slice of all tracked temporary file paths.
//
// Outputs:
//   - []string: A slice of file paths.
func (c *BaseContext) GetTempFiles() []string {
	return c.tempFiles
}

// AddError adds an error to the context's error map, keyed by the command name.
//
// Inputs:
//   - key: The name of the command that generated the error.
//   - err: The error object.
func (c *BaseContext) AddError(key string, err error) {
	c.errors[key] = err
}

// GetErrors returns the map of all errors collected during the workflow.
//
// Outputs:
//   - map[string]error: A map where keys are command names and values are the errors.
func (c *BaseContext) GetErrors() map[string]error {
	return c.errors
}

// Get retrieves a value from the context's data map by its key.
//
// Inputs:
//   - key: The string key of the data to retrieve.
//
// Outputs:
//   - interface{}: The stored value, or `nil` if the key does not exist.
func (c *BaseContext) Get(key string) interface{} {
	return c.data[key]
}

// Remove deletes a key-value pair from the context's data map.
//
// Inputs:
//   - key: The key of the item to remove.
func (c *BaseContext) Remove(key string) {
	delete(c.data, key)
}

// HasErrors checks if any errors have been added to the context.
//
// Outputs:
//   - bool: True if the error map is not empty, false otherwise.
func (c *BaseContext) HasErrors() bool {
	return len(c.errors) > 0
}
