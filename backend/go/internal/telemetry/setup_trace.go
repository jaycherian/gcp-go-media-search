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
// This file focuses on initializing the OpenTelemetry SDK for capturing and
// exporting trace and metric data to Google Cloud's observability suite.
package telemetry

import (
	"context"
	"errors"
	"log"
	"log/slog"

	"go.opentelemetry.io/otel/sdk/metric"

	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	telemetryexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// SetupOpenTelemetry initializes and configures the OpenTelemetry SDK for the entire application.
// It sets up both tracing and metrics, configuring them to export data to Google Cloud's
// observability suite (Cloud Trace for traces, Cloud Monitoring for metrics). It returns
// a `shutdown` function that must be called on application exit to ensure all buffered
// telemetry data is flushed before the application terminates.
//
// Inputs:
//   - ctx: The parent context, used for initialization of clients.
//   - config: The application's configuration struct, which provides necessary
//     details like the Google Project ID and the application's service name.
//
// Returns:
//   - shutdown: A function that should be deferred by the caller to gracefully
//     shut down all telemetry components (TracerProvider, MeterProvider).
//   - err: An error if any part of the setup fails.
func SetupOpenTelemetry(ctx context.Context, config *cloud.Config) (shutdown func(context.Context) error, err error) {
	// A slice to hold all the shutdown functions for the various telemetry components.
	var shutdownFuncs []func(context.Context) error

	// The returned shutdown function iterates over the shutdownFuncs slice and calls
	// each one, joining any errors that occur. This provides a single function to
	// cleanly tear down the entire telemetry pipeline.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// --- Resource Detection ---
	// A "resource" in OpenTelemetry represents the entity producing telemetry (i.e., this application).
	// It's a collection of attributes (key-value pairs) that describe the entity.
	res, err := resource.New(ctx,
		// The GCP detector automatically discovers resource attributes when running on
		// Google Cloud infrastructure (e.g., GCE instance ID, GKE cluster name).
		resource.WithDetectors(gcp.NewDetector()),
		// Includes default resource attributes like host, OS, and process info.
		resource.WithTelemetrySDK(),
		// Adds a custom service name attribute, which is crucial for identifying and
		// filtering telemetry data in the observability backend.
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.Application.Name),
		),
	)
	// Handle potential partial errors during resource detection without stopping execution.
	if errors.Is(err, resource.ErrPartialResource) || errors.Is(err, resource.ErrSchemaURLConflict) {
		slog.Warn("partial resource detection", "error", err)
	} else if err != nil {
		// Handle fatal errors during resource creation.
		slog.Error("resource.New failed", "error", err)
		return nil, err
	}

	// --- Propagator Setup ---
	// A propagator is responsible for injecting and extracting trace context data
	// (like trace IDs) into and from requests, enabling distributed tracing across
	// different services. `autoprop` automatically configures standard propagators
	// like W3C Trace Context and B3, which are widely supported.
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	// --- Trace Exporter and Provider Setup ---
	// An exporter is responsible for sending telemetry data to a specific backend.
	// This exporter sends trace data to Google Cloud Trace.
	traceExporter, err := telemetryexporter.New(telemetryexporter.WithProjectID(config.Application.GoogleProjectId))
	if err != nil {
		slog.Error("unable to set up trace exporter", "error", err)
		return nil, err
	}

	// The TracerProvider is the factory for creating Tracers. It's configured here.
	tp := sdktrace.NewTracerProvider(
		// WithBatcher sends spans in batches, which is much more efficient than sending one by one.
		sdktrace.WithBatcher(traceExporter),
		// Attaches the resource information (service name, GCP details, etc.) to all spans.
		sdktrace.WithResource(res),
	)

	// Add the TracerProvider's shutdown function to our list for graceful shutdown.
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	// Register our configured TracerProvider as the global one for the application.
	otel.SetTracerProvider(tp)

	// --- Metric Exporter and Provider Setup ---
	// This exporter sends metric data to Google Cloud Monitoring.
	mExporter, err := mexporter.New(
		mexporter.WithProjectID(config.Application.GoogleProjectId),
	)

	if err != nil {
		log.Printf("Failed to create metric exporter: %v", err)
		return nil, err
	}

	// The MeterProvider is the factory for creating Meters.
	mProvider := metric.NewMeterProvider(
		// Configures the provider to read and export metrics periodically.
		metric.WithReader(metric.NewPeriodicReader(mExporter)),
		metric.WithResource(res),
	)

	// Create a named "Meter" for the application. Using a namespace is a best practice
	// to avoid metric name collisions if the application uses multiple libraries that
	// also produce metrics.
	otel.Meter("github.com/GoogleCloudPlatform/solutions/media")

	// Add the MeterProvider's shutdown function to our list.
	shutdownFuncs = append(shutdownFuncs, mProvider.Shutdown)
	// Register our configured MeterProvider as the global one.
	otel.SetMeterProvider(mProvider)

	return shutdown, nil
}
