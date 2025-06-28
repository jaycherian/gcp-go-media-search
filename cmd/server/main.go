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

// // state manages the application's dependencies.
// var state = &StateManager{}

// // StateManager holds the shared components for the application.
// type StateManager struct {
// 	config        *cloud.Config
// 	cloud         *cloud.ServiceClients
// 	searchService *services.SearchService
// 	mediaService  *services.MediaService
// }

func main() {
	telemetry.SetupLogging()
	slog.Info("Logging initialized")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := GetConfig()

	_, err := telemetry.SetupOpenTelemetry(ctx, config)
	if err != nil {
		slog.Error("Failed to setup OpenTelemetry", "error", err)
		log.Fatal(err)
	}
	slog.Info("Tracing initialized")

	InitState(ctx)
	slog.Info("Initialized State")

	r := gin.Default()

	// Add OpenTelemetry middleware
	r.Use(otelgin.Middleware("media-search-server"))

	// **Use a default, more robust CORS configuration for development**
	// This allows all origins, methods, and headers, which is safe for local dev
	// and fixes potential communication issues between the frontend and backend.
	r.Use(cors.Default())

	// Create the "/api/v1" group
	apiV1 := r.Group("/api/v1")
	{
		// Register media and upload routes
		MediaRouter(apiV1)
		FileUpload(apiV1)
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r, // Use r as the handler, not r.Handler()
	}

	// Start the server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("filed to listen: ", "error", err)
		}
	}()
	slog.Info("Server Ready on port 8080")

	// Wait for an interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutdown Server ...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server Shutdown Failed:", "error", err)
	}

	log.Println("Server exiting")
}

// MediaRouter sets up the routes for media searching and retrieval.
func MediaRouter(r *gin.RouterGroup) {
	media := r.Group("/media")
	{
		media.GET("", func(c *gin.Context) {
			query := c.Query("s")
			count, err := strconv.Atoi(c.DefaultQuery("count", "5"))
			if err != nil {
				count = 5
			}
			if len(query) == 0 {
				c.Status(http.StatusBadRequest)
				return
			}
			sceneResults, err := state.searchService.FindScenes(c, query, count)
			if err != nil {
				log.Printf("Error finding scenes: %v\n", err)
				c.Status(http.StatusInternalServerError)
				return
			}

			out := make(map[string]*model.Media)
			for _, r := range sceneResults {
				var med *model.Media
				if m, ok := out[r.MediaId]; !ok {
					m, err := state.mediaService.Get(c, r.MediaId)
					if err != nil {
						log.Printf("Error getting media %s: %v\n", r.MediaId, err)
						c.Status(http.StatusInternalServerError)
						return
					}
					m.Scenes = make([]*model.Scene, 0)
					out[r.MediaId] = m
					med = m
				} else {
					med = m
				}

				s, err := state.mediaService.GetScene(c, r.MediaId, r.SequenceNumber)
				if err != nil {
					log.Printf("Error getting scene %d for media %s: %v\n", r.SequenceNumber, r.MediaId, err)
					c.Status(http.StatusInternalServerError)
					return
				}
				med.Scenes = append(med.Scenes, s)
			}
			results := make([]*model.Media, 0, len(out))
			for _, v := range out {
				results = append(results, v)
			}
			c.JSON(http.StatusOK, results)
		})

		media.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			out, err := state.mediaService.Get(c, id)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.JSON(http.StatusOK, out)
		})

		// New endpoint to generate a signed URL for streaming
		media.GET("/:id/stream", func(c *gin.Context) {
			id := c.Param("id")
			print("After assigning ID\n")
			media, err := state.mediaService.Get(c, id)
			print("after media\n")
			if err != nil {
				print("inside first error\n")
				c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
				return
			}

			// Generate a signed URL that is valid for 15 minutes.
			signedURL, err := state.mediaService.GenerateSignedURL(c, media.MediaUrl, 15*time.Minute)
			print("After getting signed URL\n")
			if err != nil {
				print("inside second error\n")
				print("The error is \n")
				print(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate streaming URL"})
				return
			}
			print("Before returning ok\n)")
			c.JSON(http.StatusOK, gin.H{"url": signedURL})
			print("the signed url is ", signedURL)
		})

		media.GET("/:id/scenes/:scene_id", func(c *gin.Context) {
			id := c.Param("id")
			sceneID, err := strconv.Atoi(c.Param("scene_id"))
			if err != nil {
				c.Status(http.StatusBadRequest)
				return
			}
			out, err := state.mediaService.GetScene(c, id, sceneID)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.JSON(http.StatusOK, out)
		})
	}
}

// FileUpload sets up the route for handling file uploads.
func FileUpload(r *gin.RouterGroup) {
	upload := r.Group("/uploads")
	{
		upload.POST("", func(c *gin.Context) {
			form, err := c.MultipartForm()
			if err != nil {
				c.String(http.StatusBadRequest, "get form err: %s", err.Error())
				return
			}
			files := form.File["files"]
			bucket := state.cloud.StorageClient.Bucket(state.config.Storage.HiResInputBucket)

			for _, file := range files {
				localPath := filepath.Join(os.TempDir(), file.Filename)
				if err := c.SaveUploadedFile(file, localPath); err != nil {
					c.String(http.StatusBadRequest, "upload file err: %s", err.Error())
					return
				}

				content, err := os.ReadFile(localPath)
				if err != nil {
					log.Println(err)
					c.Status(http.StatusInternalServerError)
					return
				}
				wc := bucket.Object(file.Filename).NewWriter(c)
				wc.ContentType = "video/mp4"
				if _, err = wc.Write(content); err != nil {
					c.String(http.StatusInternalServerError, "write file to bucket err: %s", err.Error())
					return
				}
				if err := wc.Close(); err != nil {
					log.Printf("failed to close bucket handle: %v\n", err)
				}
				if err := os.Remove(localPath); err != nil {
					log.Printf("failed to remove file from server: %v\n", err)
				}
			}
			c.String(http.StatusOK, "Uploaded successfully %d files.", len(files))
		})
	}
}

// // SetupListeners configures and starts the Pub/Sub listeners.
// func SetupListeners(config *cloud.Config, cloudClients *cloud.ServiceClients, ctx context.Context) {
// 	mediaResizeWorkflow := workflow.NewMediaResizeWorkflow(config, cloudClients, "ffmpeg", &model.MediaFormatFilter{Width: "240"})
// 	cloudClients.PubSubListeners["HiResTopic"].SetCommand(mediaResizeWorkflow)
// 	cloudClients.PubSubListeners["HiResTopic"].Listen(ctx)

// 	mediaIngestion := workflow.NewMediaReaderPipeline(config, cloudClients, "creative-flash")
// 	cloudClients.PubSubListeners["LowResTopic"].SetCommand(mediaIngestion)
// 	cloudClients.PubSubListeners["LowResTopic"].Listen(ctx)
// }

// // GetConfig loads the application configuration.
// func GetConfig() *cloud.Config {
// 	if state.config == nil {
// 		if err := os.Setenv(cloud.EnvConfigFilePrefix, "configs"); err != nil {
// 			log.Fatalf("failed to setup env: %v\n", err)
// 		}
// 		if err := os.Setenv(cloud.EnvConfigRuntime, "local"); err != nil {
// 			log.Fatalf("failed to setup env: %v\n", err)
// 		}
// 		config := cloud.NewConfig()
// 		cloud.LoadConfig(&config)
// 		state.config = config
// 	}
// 	return state.config
// }

// // InitState initializes the application state and dependencies.
// func InitState(ctx context.Context, config *cloud.Config) {
// 	cloudClients, err := cloud.NewCloudServiceClients(ctx, config)
// 	if err != nil {
// 		panic(err)
// 	}
// 	state.cloud = cloudClients

// 	datasetName := config.BigQueryDataSource.DatasetName
// 	mediaTableName := config.BigQueryDataSource.MediaTable
// 	embeddingTableName := config.BigQueryDataSource.EmbeddingTable

// 	state.searchService = &services.SearchService{
// 		BigqueryClient: cloudClients.BiqQueryClient,
// 		EmbeddingModel: cloudClients.EmbeddingModels["multi-lingual"],
// 		DatasetName:    datasetName,
// 		MediaTable:     mediaTableName,
// 		EmbeddingTable: embeddingTableName,
// 	}

// 	state.mediaService = &services.MediaService{
// 		BigqueryClient: cloudClients.BiqQueryClient,
// 		DatasetName:    datasetName,
// 		MediaTable:     mediaTableName,
// 	}

// 	embeddingGenerator := workflow.NewMediaEmbeddingGeneratorWorkflow(config, cloudClients)
// 	embeddingGenerator.StartTimer()

// 	SetupListeners(config, cloudClients, ctx)
// }
