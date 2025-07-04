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
// for creating workflows as a sequence of commands. This file defines the
// `BaseChain`, which is the default implementation of the `Chain` interface.
//
// Logic Flow:
// A `BaseChain` is itself a `Command`, allowing chains to be nested within
// other chains. Its primary role is to execute a list of `Command` objects in
// a predefined order. It manages the flow of execution and the "piping" of
// data between commands.
//
//  1. **Execution starts**: The `Execute` method is called with a shared context.
//  2. **Telemetry**: An OpenTelemetry span is created for the entire chain's execution
//     to monitor its performance and status.
//  3. **Command Loop**: The chain iterates through its list of commands.
//  4. **Error Handling**: Before executing a command, it checks if the context already
//     has errors. If it does, and if `continueOnFailure` is `false` (the default),
//     the chain stops executing immediately.
//  5. **Execution & Context Management**: For each command:
//     - A new child OpenTelemetry span is created for the specific command.
//     - The `cor.Context` is updated with this new span's Go context.
//     - The command's `Execute` method is called.
//     - The Go context is reset to the parent's context to keep command traces nested correctly.
//  6. **Data Piping**: After a command executes, the `BaseChain` performs a "flip-flop".
//     It takes the value that the command placed in the `CtxOut` (output) parameter
//     of the context and moves it to the `CtxIn` (input) parameter. This makes the
//     output of one command the direct input for the next, creating a processing pipeline.
//  7. **Completion**: Once all commands are run (or the chain is stopped by an error),
//     the main OpenTelemetry span for the chain is closed and its status is set to
//     success or failure based on the final state of the context.
package cor

import (
	"fmt"

	"go.opentelemetry.io/otel/codes"
)

// BaseChain is the default implementation of the Chain interface. It holds a slice
// of commands to be executed sequentially.
type BaseChain struct {
	BaseCommand
	continueOnFailure bool      // A flag that determines if the chain should continue executing subsequent commands after one fails.
	commands          []Command // The ordered list of commands that this chain will execute.
}

// NewBaseChain is the constructor for BaseChain.
//
// Inputs:
//   - name: A string name for this chain instance, used for logging and telemetry.
//
// Outputs:
//   - *BaseChain: A pointer to the newly instantiated chain.
func NewBaseChain(name string) *BaseChain {
	return &BaseChain{BaseCommand: *NewBaseCommand(name)}
}

// ContinueOnFailure is a builder method that sets the error handling behavior of the chain.
//
// Inputs:
//   - continueOnFailure: A boolean. If true, the chain will execute all its commands
//     even if some of them add errors to the context. If false, the chain will stop
//     at the first command that fails.
//
// Outputs:
//   - Chain: The chain instance, allowing for fluent method chaining (e.g., `NewBaseChain(...).ContinueOnFailure(...)`).
func (c *BaseChain) ContinueOnFailure(continueOnFailure bool) Chain {
	c.continueOnFailure = continueOnFailure
	return c
}

// AddCommand is a builder method that adds a command to the end of the chain's execution sequence.
//
// Inputs:
//   - command: A component that implements the `Command` interface.
//
// Outputs:
//   - Chain: The chain instance, allowing for fluent method chaining (e.g., `NewBaseChain(...).AddCommand(...)`).
func (c *BaseChain) AddCommand(command Command) Chain {
	c.commands = append(c.commands, command)
	return c
}

// IsExecutable checks if the chain can be executed. For a chain, this simply means
// that a valid Go context exists.
func (c *BaseChain) IsExecutable(context Context) bool {
	return context.GetContext() != nil
}

// Execute orchestrates the sequential execution of all commands in the chain.
//
// Inputs:
//   - chCtx: The shared `cor.Context` for the entire workflow execution.
func (c *BaseChain) Execute(chCtx Context) {
	// Keep a reference to the Go context that this chain started with.
	parentCtx := chCtx.GetContext()

	// Start a new OpenTelemetry span for the entire chain's execution.
	outerCtx, chainSpan := c.Tracer.Start(parentCtx, fmt.Sprintf("%s_execute", c.GetName()))
	defer chainSpan.End() // Ensure the span is closed when the function returns.

	// Loop through each command in the chain's list.
	for _, command := range c.commands {
		// Start a new child span for the individual command. This allows us to trace
		// the performance and status of each step in the chain.
		commandContext, commandSpan := c.Tracer.Start(outerCtx, command.GetName())
		commandSpan.SetName(command.GetName()) // Explicitly set the span name for clarity.

		// Check if a previous command in the chain has already failed.
		// If so, and if we are not configured to continue, stop processing.
		if chCtx.HasErrors() && !c.continueOnFailure {
			commandSpan.SetStatus(codes.Error, "previous error on chain; skipping execution")
			commandSpan.End()
			break // Exit the loop.
		}

		// Check if the current command is able to run with the current context.
		if command.IsExecutable(chCtx) {
			// Set the Go context for the command to the new child span's context.
			// This ensures that any operations within the command are traced as children of this command's span.
			chCtx.SetContext(commandContext)

			// Execute the command's core logic.
			command.Execute(chCtx)

			// **Important**: Reset the shared context's Go context back to the chain's main
			// context. This prevents the next command's span from being a grandchild of
			// the previous command's span, keeping the trace hierarchy flat and clean.
			chCtx.SetContext(outerCtx)

		} else {
			// If the command is not executable, record it as an error in the trace.
			commandSpan.SetStatus(codes.Error, fmt.Sprintf("command not executable: %s", command.GetName()))
		}

		// After execution, check the context for errors and set the command's span status accordingly.
		if chCtx.HasErrors() {
			commandSpan.SetStatus(codes.Error, "error during or after command execution")
		} else {
			commandSpan.SetStatus(codes.Ok, "command completed successfully")
		}
		commandSpan.End() // End the span for the individual command.

		// --- Data Piping Logic ---
		// "Flip-flop" the input and output to create a pipeline effect.
		// The value placed in CtxOut by the command that just ran...
		outputValue := chCtx.Get(CtxOut)
		// ...is now placed in CtxIn for the next command in the loop.
		chCtx.Remove(CtxIn) // Clean up old input
		if outputValue != nil {
			chCtx.Add(CtxIn, outputValue)
		}
		chCtx.Remove(CtxOut) // Clean up the output to prepare for the next command.
	}

	// After the loop finishes, set the final status for the entire chain's span.
	if !chCtx.HasErrors() {
		chainSpan.SetStatus(codes.Ok, "chain completed successfully")
	} else {
		chainSpan.SetStatus(codes.Error, "chain failed to execute")
	}
}
