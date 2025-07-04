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
// for creating workflows. This file defines the core interfaces that govern the
// behavior of all components within this pattern. By using interfaces, the
// framework remains flexible and extensible, allowing different implementations
// of commands, chains, and contexts to be used interchangeably.
package cor

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// CtxIn and CtxOut are constant keys used to manage the primary data flow
// within a BaseChain.
const (
	// CtxIn is the default key for the primary input of a command. The BaseChain
	// will automatically populate the value of this key with the output from the
	// previous command.
	CtxIn = "__IN__"
	// CtxOut is the default key where a command should place its primary output.
	// The BaseChain will pick up the value from this key to use as the input
	// for the next command.
	CtxOut = "__OUT__"
)

// Context defines the interface for a shared state object that is passed
// through a chain of commands. It acts as a "property bag" or "state machine"
// for a single workflow execution, carrying data, errors, and other state
// between commands.
type Context interface {
	// SetContext sets the standard Go `context.Context`. This is primarily
	// used for passing request-scoped data like cancellation signals and
	// OpenTelemetry trace information.
	SetContext(context context.Context)

	// GetContext retrieves the standard Go `context.Context`.
	GetContext() context.Context

	// Add stores a key-value pair in the context. This is the primary way
	// commands share data with each other. It returns the Context to allow
	// for fluent method chaining.
	Add(key string, value interface{}) Context

	// AddError records an error that occurred within a command. The key should
	// typically be the name of the command that produced the error.
	AddError(key string, err error)

	// GetErrors returns a map of all errors collected during the workflow.
	GetErrors() map[string]error

	// Get retrieves a value from the context by its key.
	Get(key string) interface{}

	// Remove deletes a key-value pair from the context.
	Remove(key string)

	// HasErrors checks if any errors have been recorded in the context.
	HasErrors() bool

	// AddTempFile tracks a temporary file that was created during the workflow.
	// This allows the context to clean up all temporary files at the end.
	AddTempFile(file string)

	// GetTempFiles returns a list of all tracked temporary file paths.
	GetTempFiles() []string

	// Close performs any necessary cleanup, such as deleting all temporary files
	// tracked by AddTempFile. This should be deferred at the start of a workflow.
	Close()
}

// Executable is a simple interface for any object that has a core execution logic.
type Executable interface {
	// Execute contains the primary business logic of the object. It takes a
	// Context object to read its inputs from and write its outputs to.
	Execute(context Context)
}

// Command represents an atomic, testable, and thread-safe unit of work.
// It is the fundamental building block of a workflow.
type Command interface {
	Executable // Embeds the Execute method.

	// GetName returns the unique name of the command, used for logging and telemetry.
	GetName() string

	// GetInputParam returns the key that the command will use to look up its
	// primary input in the Context.
	GetInputParam() string

	// GetOutputParam returns the key that the command will use to store its
	// primary output in the Context.
	GetOutputParam() string

	// IsExecutable checks if the command can be run with the current state of
	// the Context. This is a precondition check before calling Execute.
	IsExecutable(context Context) bool

	// GetTracer returns the OpenTelemetry tracer for this command.
	GetTracer() trace.Tracer

	// GetMeter returns the OpenTelemetry meter for creating metrics.
	GetMeter() metric.Meter

	// GetSuccessCounter returns a metric counter for successful executions.
	GetSuccessCounter() metric.Int64Counter

	// GetErrorCounter returns a metric counter for failed executions.
	GetErrorCounter() metric.Int64Counter
}

// Chain represents a sequence of commands. It is itself a Command, which allows
// chains to be nested within other chains (Composite Pattern). The Chain is
// responsible for orchestrating the execution of its child commands.
type Chain interface {
	Command // A Chain is a Command.

	// ContinueOnFailure is a configuration method that tells the chain whether
	// to stop executing if one of its commands fails (adds an error to the context).
	ContinueOnFailure(bool) Chain

	// AddCommand adds a new command to the end of the execution sequence.
	AddCommand(command Command) Chain
}
