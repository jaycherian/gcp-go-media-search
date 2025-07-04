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

// Package telemetry provides utilities for setting up and configuring
// application observability, including logging, tracing, and metrics.
// This file specifically handles the setup of structured logging that
// is compatible with Google Cloud Logging and integrates with OpenTelemetry traces.
package telemetry

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// spanContextLogHandler is a custom slog.Handler that wraps another handler.
// Its purpose is to intercept each log record and automatically inject
// OpenTelemetry trace and span IDs if they exist in the context. This allows
// for seamless correlation between logs and traces in observability platforms
// like Google Cloud Trace.
type spanContextLogHandler struct {
	slog.Handler
}

// handlerWithSpanContext is a constructor function that creates a new
// spanContextLogHandler, wrapping the provided base handler.
func handlerWithSpanContext(handler slog.Handler) *spanContextLogHandler {
	return &spanContextLogHandler{Handler: handler}
}

// Handle is the core method of the custom handler. It is called for every log
// message. It checks the provided context for a valid OpenTelemetry SpanContext.
// If found, it adds the trace ID, span ID, and trace sampled flag to the log
// record using the specific field names required by Google Cloud Logging for
// automatic correlation.
func (t *spanContextLogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Get the SpanContext from the Go context.
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		// Add trace context attributes following the Cloud Logging structured log format.
		// See: https://cloud.google.com/logging/docs/structured-logging#special-payload-fields

		// Add the Trace ID.
		record.AddAttrs(
			slog.Any("logging.googleapis.com/trace", s.TraceID()),
		)
		// Add the Span ID.
		record.AddAttrs(
			slog.Any("logging.googleapis.com/spanId", s.SpanID()),
		)
		// Add a boolean indicating if the trace was sampled.
		record.AddAttrs(
			slog.Bool("logging.googleapis.com/trace_sampled", s.TraceFlags().IsSampled()),
		)
	}
	// Pass the (potentially modified) log record to the underlying wrapped handler.
	return t.Handler.Handle(ctx, record)
}

// replacer is a function used to modify log attributes before they are written.
// It renames the default slog attribute keys (e.g., "level", "time", "msg")
// to the specific keys expected by Google Cloud Logging ("severity", "timestamp", "message").
// This ensures that logs are parsed correctly and displayed with the proper severity
// and timestamp in the Google Cloud Console.
func replacer(_ []string, a slog.Attr) slog.Attr {
	// Rename attribute keys to match Cloud Logging structured log format.
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
		// Map slog.Level string values to Cloud Logging LogSeverity enum.
		// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity
		// Example: Convert slog's "WARN" to Cloud Logging's "WARNING".
		if level := a.Value.Any().(slog.Level); level == slog.LevelWarn {
			a.Value = slog.StringValue("WARNING")
		}
	case slog.TimeKey:
		a.Key = "timestamp"
	case slog.MessageKey:
		a.Key = "message"
	}
	return a
}

// SetupLogging initializes the logging system for the entire application.
// It configures both the standard `log` package and the structured `slog` package.
// It sets up a JSON-based logger that writes to both a file (`app.log`) and
// standard output, and it enables the automatic injection of trace context.
func SetupLogging() {
	// Create a log file. The file will be created if it doesn't exist,
	// or truncated if it does.
	file, _ := os.Create("app.log")
	// Create a multi-writer that directs log output to both standard output
	// (the console) and the log file simultaneously.
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Configure the standard Go `log` package.
	// Set its output to our multi-writer.
	log.SetOutput(multiWriter)
	// Add a prefix to all messages from the standard logger.
	log.SetPrefix("[INFO] ")
	// Configure the standard logger to include the date and time in its output.
	log.SetFlags(log.Ldate | log.Ltime)

	// --- Setup the structured logger (slog) ---
	// 1. Create a handler that writes logs in JSON format.
	//    It uses our multi-writer and the replacer function to format attributes for GCP.
	jsonHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{ReplaceAttr: replacer})

	// 2. Wrap the JSON handler with our custom handler to add trace context.
	instrumentedHandler := handlerWithSpanContext(jsonHandler)

	// 3. Set this fully configured handler as the global default for the slog package.
	//    Any call to slog.Info, slog.Error, etc., will now use this handler.
	slog.SetDefault(slog.New(instrumentedHandler))
	// Set the minimum log level to Info. Debug messages will be ignored.
	slog.SetLogLoggerLevel(slog.LevelInfo)
}
