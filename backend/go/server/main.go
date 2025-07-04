// Copyright 2024 Google, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// *****************************************************************************************************//
// Package main is the entry point for the media search backend server.
//
// This application sets up and runs a web server using the Gin framework. It provides a REST API
// for media search and file uploads. The server is instrumented with OpenTelemetry for logging,
// tracing, and metrics, providing observability into the application's performance.
//
// The main function initializes the application's configuration, sets up logging and telemetry,
// and initializes the application state, including clients for Google Cloud services. It defines
// API routes for searching media, retrieving media details, generating streaming URLs, and uploading files.
//
// The server also sets up and manages background listeners for Pub/Sub topics, which trigger
// workflows for processing media files (resizing, AI analysis, etc.) when new files are
// uploaded to Google Cloud Storage.
//
// Functions:
//   - main: The main entry point of the application. It sets up the server, configures routes,
//     initializes services, and handles graceful shutdown.
//   - MediaRouter: Sets up the API routes related to media, such as searching for media,
//     retrieving specific media items and scenes, and generating signed URLs for streaming.
//   - FileUpload: Configures the API endpoint for handling multipart/form-data file uploads,
//     saving the uploaded files to a Google Cloud Storage bucket.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"github.com/jaycherian/gcp-go-media-search/internal/telemetry"
)

// main is the primary entry point for the application.
// It orchestrates the setup of logging, telemetry, configuration, cloud services,
// the web server, API routes, and background listeners. It also handles graceful
// shutdown of the server upon receiving an interrupt signal.
func main() {
	// Initialize structured logging for the application.
	telemetry.SetupLogging()
	slog.Info("Logging initialized")

	// Create a new context that can be cancelled. This is the root context for the application.
	ctx, cancel := context.WithCancel(context.Background())
	// Defer the cancel function to be called when main exits, ensuring all child contexts are cancelled.
	defer cancel()

	// Load application configuration from TOML files.
	config := GetConfig()

	// Initialize OpenTelemetry for distributed tracing and metrics.
	_, err := telemetry.SetupOpenTelemetry(ctx, config)
	if err != nil {
		slog.Error("Failed to setup OpenTelemetry", "error", err)
		log.Fatal(err)
	}
	slog.Info("Tracing initialized")

	// Initialize the application's state, including all necessary service clients.
	InitState(ctx)
	slog.Info("Initialized State")

	// Set up the Gin web server with default middleware.
	r := gin.Default()

	// Add OpenTelemetry middleware to the Gin router to trace incoming requests.
	// This will automatically create spans for each request.
	r.Use(otelgin.Middleware("media-search-server"))

	// Configure Cross-Origin Resource Sharing (CORS) middleware.
	// Using cors.Default() provides a permissive configuration suitable for development,
	// allowing requests from any origin.
	r.Use(cors.Default())

	// Group routes under the "/api/v1" prefix.
	apiV1 := r.Group("/api/v1")
	{
		// Register the routes for media and file upload functionality within the API group.
		MediaRouter(apiV1)
		FileUpload(apiV1)
	}

	// Configure the HTTP server with the address and handler.
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	// Start the HTTP server in a separate goroutine so it doesn't block the main thread.
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("filed to listen: ", "error", err)
		}
	}()
	slog.Info("Server Ready on port 8080")

	// Set up a channel to listen for OS interrupt signals (e.g., Ctrl+C).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received on the quit channel.
	<-quit
	slog.Info("Shutdown Server ...")

	// Create a context with a timeout for the graceful shutdown.
	// This gives active requests 5 seconds to complete.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Attempt to gracefully shut down the server.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server Shutdown Failed:", "error", err)
	}

	log.Println("Server exiting")
}

// MediaRouter sets up the API routes for media-related actions.
//
// Inputs:
//   - r: A *gin.RouterGroup to which the media routes will be added. This allows
//     nesting routes under a common path prefix (e.g., "/api/v1").
//
// Outputs:
//   - This function does not return any values. It modifies the provided *gin.RouterGroup
//     by adding new route handlers.
//
// This function defines the following endpoints:
//   - GET /media: Searches for media scenes based on a query string 's'.
//   - GET /media/:id: Retrieves the full details of a specific media object by its ID.
//   - GET /media/:id/stream: Generates a time-limited, signed URL for securely streaming a media file.
//   - GET /media/:id/scenes/:scene_id: Fetches the details of a specific scene within a media object.
func MediaRouter(r *gin.RouterGroup) {
	// Group all media-related routes under the "/media" path.
	media := r.Group("/media")
	{
		// Handler for GET /media?s=<query>&count=<n>
		media.GET("", func(c *gin.Context) {
			// Get the search query 's' from the URL parameters.
			query := c.Query("s")
			// Get the 'count' parameter, defaulting to 5 if not provided or invalid.
			count, err := strconv.Atoi(c.DefaultQuery("count", "5"))
			if err != nil {
				count = 5
			}
			// If the query is empty, it's a bad request.
			if len(query) == 0 {
				c.Status(http.StatusBadRequest)
				return
			}
			// Call the search service to find scenes matching the query.
			sceneResults, err := state.searchService.FindScenes(c, query, count)
			if err != nil {
				log.Printf("Error finding scenes: %v\n", err)
				c.Status(http.StatusInternalServerError)
				return
			}

			// Use a map to aggregate scenes by their parent media ID to avoid duplicate media lookups.
			out := make(map[string]*model.Media)
			// Iterate over the search results.
			for _, r := range sceneResults {
				var med *model.Media
				// Check if we've already fetched this media object.
				if m, ok := out[r.MediaId]; !ok {
					// If not, fetch the full media details.
					m, err := state.mediaService.Get(c, r.MediaId)
					if err != nil {
						log.Printf("Error getting media %s: %v\n", r.MediaId, err)
						c.Status(http.StatusInternalServerError)
						return
					}
					// Initialize the scenes slice to ensure it's not nil.
					m.Scenes = make([]*model.Scene, 0)
					out[r.MediaId] = m
					med = m
				} else {
					// If we have, use the existing media object.
					med = m
				}

				// Get the specific scene details for the current search result.
				s, err := state.mediaService.GetScene(c, r.MediaId, r.SequenceNumber)
				if err != nil {
					log.Printf("Error getting scene %d for media %s: %v\n", r.SequenceNumber, r.MediaId, err)
					c.Status(http.StatusInternalServerError)
					return
				}
				// Append the found scene to the media object's scene list.
				med.Scenes = append(med.Scenes, s)
			}
			// Convert the map of media objects to a slice for the JSON response.
			results := make([]*model.Media, 0, len(out))
			for _, v := range out {
				results = append(results, v)
			}
			// Return the aggregated results as a JSON array.
			c.JSON(http.StatusOK, results)
		})

		// Handler for GET /media/:id
		media.GET("/:id", func(c *gin.Context) {
			// Get the media ID from the URL path.
			id := c.Param("id")
			// Fetch the media object by its ID.
			out, err := state.mediaService.Get(c, id)
			if err != nil {
				// If not found, return a 404 status.
				c.Status(http.StatusNotFound)
				return
			}
			// Return the media object as JSON.
			c.JSON(http.StatusOK, out)
		})

		// Handler for GET /media/:id/stream
		// This endpoint provides a secure, time-limited URL for clients to stream video content.
		media.GET("/:id/stream", func(c *gin.Context) {
			id := c.Param("id")
			// Fetch the media metadata to get the GCS URL.
			media, err := state.mediaService.Get(c, id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
				return
			}

			// Generate a signed URL valid for 15 minutes for the media file.
			signedURL, err := state.mediaService.GenerateSignedURL(c, media.MediaUrl, 15*time.Minute)
			if err != nil {
				log.Printf("Error generating signed URL: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate streaming URL"})
				return
			}
			// Return the signed URL in the JSON response.
			c.JSON(http.StatusOK, gin.H{"url": signedURL})
		})

		// Handler for GET /media/:id/scenes/:scene_id
		media.GET("/:id/scenes/:scene_id", func(c *gin.Context) {
			// Get media ID and scene ID from the URL path.
			id := c.Param("id")
			sceneID, err := strconv.Atoi(c.Param("scene_id"))
			if err != nil {
				// If scene_id is not a valid integer, return a bad request status.
				c.Status(http.StatusBadRequest)
				return
			}
			// Fetch the specific scene from the media service.
			out, err := state.mediaService.GetScene(c, id, sceneID)
			if err != nil {
				// If not found, return a 404 status.
				c.Status(http.StatusNotFound)
				return
			}
			// Return the scene object as JSON.
			c.JSON(http.StatusOK, out)
		})
	}
}

// FileUpload sets up the route for handling file uploads.
//
// Inputs:
//   - r: A *gin.RouterGroup to which the file upload route will be added.
//
// Outputs:
//   - This function does not return any values. It registers a POST route on the
//     provided router group.
//
// This function configures a POST endpoint at "/uploads" that accepts multipart/form-data.
// It processes one or more files sent under the "files" form field, saves them
// temporarily to the local disk, and then uploads them to a configured
// Google Cloud Storage bucket before deleting the local temporary file.
func FileUpload(r *gin.RouterGroup) {
	// Group the upload route under "/uploads".
	upload := r.Group("/uploads")
	{
		// Handler for POST /uploads
		upload.POST("", func(c *gin.Context) {
			// Parse the multipart form from the request.
			form, err := c.MultipartForm()
			if err != nil {
				c.String(http.StatusBadRequest, "get form err: %s", err.Error())
				return
			}
			// Get all files associated with the "files" field.
			files := form.File["files"]
			// Get a handle to the configured GCS bucket for high-resolution files.
			bucket := state.cloud.StorageClient.Bucket(state.config.Storage.HiResInputBucket)

			// Loop through all the uploaded files.
			for _, file := range files {
				// Define a temporary local path to save the file.
				localPath := filepath.Join(os.TempDir(), file.Filename)
				// Save the uploaded file to the local temporary path.
				if err := c.SaveUploadedFile(file, localPath); err != nil {
					c.String(http.StatusBadRequest, "upload file err: %s", err.Error())
					return
				}

				// Read the file content from the local path.
				content, err := os.ReadFile(localPath)
				if err != nil {
					log.Println(err)
					c.Status(http.StatusInternalServerError)
					return
				}
				// Get a writer for the new object in the GCS bucket.
				wc := bucket.Object(file.Filename).NewWriter(c)
				// Set the content type for the GCS object.
				wc.ContentType = "video/mp4"
				// Write the file content to the GCS object.
				if _, err = wc.Write(content); err != nil {
					c.String(http.StatusInternalServerError, "write file to bucket err: %s", err.Error())
					return
				}
				// Close the GCS writer to finalize the upload.
				if err := wc.Close(); err != nil {
					log.Printf("failed to close bucket handle: %v\n", err)
				}
				// Remove the temporary local file after successful upload.
				if err := os.Remove(localPath); err != nil {
					log.Printf("failed to remove file from server: %v\n", err)
				}
			}
			// Respond with a success message.
			c.String(http.StatusOK, "Uploaded successfully %d files.", len(files))
		})
	}
}
