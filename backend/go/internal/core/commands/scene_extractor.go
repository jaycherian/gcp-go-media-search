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

// Package commands provides the concrete implementations of the Chain of
// Responsibility (COR) pattern's Command interface. This file defines a
// command that extracts detailed scene-by-scene descriptions from a media file.
//
// Logic Flow:
// This command is a performance-critical part of the media analysis pipeline.
// After the `MediaSummaryCreator` has identified the timestamps for each scene,
// this `SceneExtractor` processes each of those scenes in parallel to generate
// a detailed script or description.
//
//  1. It receives the `MediaSummary` (which contains the scene timestamps) and
//     the `genai.File` handle as input from the context.
//  2. **Worker Pool Pattern**: To speed up the process, it creates a "worker pool"
//     using Go's concurrency features (goroutines and channels).
//     - It sets up a `jobs` channel to send scene-processing tasks to workers.
//     - It sets up a `results` channel to receive the generated scripts from workers.
//     - It launches a configurable number of `sceneWorker` goroutines. A goroutine
//     is a lightweight thread managed by the Go runtime.
//  3. **Distributing Work**: The main `Execute` function loops through all scene
//     timestamps and creates a `SceneJob` for each one. Each job contains all the
//     necessary data (prompt, file handle, etc.) for a worker to process one scene.
//     These jobs are then sent into the `jobs` channel.
//  4. **Concurrent Processing**: Each `sceneWorker` goroutine pulls a job from the
//     `jobs` channel, makes a call to the Gemini model to generate the scene script,
//     and sends the result back on the `results` channel. Because these workers
//     run concurrently, multiple scenes can be processed at the same time,
//     significantly reducing the total processing time.
//  5. **Aggregating Results**: The `Execute` function waits for all workers to
//     finish (using a `sync.WaitGroup`) and then collects all the generated scene
//     scripts from the `results` channel into a single slice.
//  6. This slice of scene scripts is then placed back into the context for the
//     final `MediaAssembly` command to use.
package commands

import (
	"bytes"
	goctx "context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"go.opentelemetry.io/otel/metric"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genai"
)

// SceneExtractor is a command that processes scene timestamps in parallel to generate detailed descriptions.
type SceneExtractor struct {
	cor.BaseCommand
	generativeAIModel        *cloud.QuotaAwareGenerativeAIModel // The rate-limited generative model client.
	promptTemplate           *template.Template                 // The Go template for generating the scene-specific prompt.
	numberOfWorkers          int                                // The number of concurrent workers to spawn.
	geminiInputTokenCounter  metric.Int64Counter                // OTel counter for input tokens.
	geminiOutputTokenCounter metric.Int64Counter                // OTel counter for output tokens.
	geminiRetryCounter       metric.Int64Counter                // OTel counter for retries.
}

// NewSceneExtractor is the constructor for the SceneExtractor command.
//
// Inputs:
//   - name: A string name for this command instance.
//   - model: The client for the generative AI model.
//   - prompt: The parsed Go template for the prompt.
//   - numberOfWorkers: The size of the worker pool for concurrent processing.
//
// Outputs:
//   - *SceneExtractor: A pointer to the newly instantiated command.
func NewSceneExtractor(
	name string,
	model *cloud.QuotaAwareGenerativeAIModel,
	prompt *template.Template,
	numberOfWorkers int) *SceneExtractor {
	out := &SceneExtractor{
		BaseCommand:       *cor.NewBaseCommand(name),
		generativeAIModel: model,
		promptTemplate:    prompt,
		numberOfWorkers:   numberOfWorkers}

	// Initialize OpenTelemetry metrics specific to this command.
	out.geminiInputTokenCounter, _ = out.GetMeter().Int64Counter(fmt.Sprintf("%s.gemini.token.input", out.GetName()))
	out.geminiOutputTokenCounter, _ = out.GetMeter().Int64Counter(fmt.Sprintf("%s.gemini.token.output", out.GetName()))
	out.geminiRetryCounter, _ = out.GetMeter().Int64Counter(fmt.Sprintf("%s.gemini.retry", out.GetName()))

	return out
}

// IsExecutable checks if the required data (media summary and video file handle) is present in the context.
func (s *SceneExtractor) IsExecutable(context cor.Context) bool {
	return context != nil &&
		context.Get(s.GetInputParam()) != nil &&
		context.Get(GetVideoUploadFileParameterName()) != nil
}

// Execute orchestrates the parallel processing of scene extractions.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (s *SceneExtractor) Execute(context cor.Context) {
	// Retrieve necessary data from the context.
	summary := context.Get(s.GetInputParam()).(*model.MediaSummary)
	videoFile := context.Get(GetVideoUploadFileParameterName()).(*genai.File)

	// --- Prepare data for the prompt template ---
	exampleScene := model.GetExampleScene()
	exampleJson, _ := json.Marshal(exampleScene)
	exampleText := string(exampleJson)

	// Create a human-readable string of the cast for the prompt context.
	var castBuilder strings.Builder
	for _, cast := range summary.Cast {
		fmt.Fprintf(&castBuilder, "%s - %s\n", cast.CharacterName, cast.ActorName)
	}
	summaryText := fmt.Sprintf("Title:%s\nSummary:\n\n%s\nCast:\n\n%s", summary.Title, summary.Summary, castBuilder.String())

	// --- Setup for Concurrent Processing ---
	// A WaitGroup is used to wait for a collection of goroutines to finish.
	var wg sync.WaitGroup

	// `jobs` is a buffered channel that will be used to send work to the workers.
	// The buffer size is the number of scenes, so we can send all jobs without blocking.
	jobs := make(chan *SceneJob, len(summary.SceneTimeStamps))

	// `results` is a channel where workers will send back their results (or errors).
	results := make(chan *SceneResponse, len(summary.SceneTimeStamps))

	// --- Launch Worker Goroutines ---
	for w := 1; w <= s.numberOfWorkers; w++ {
		wg.Add(1) // Increment the WaitGroup counter for each worker.
		// Launch a new goroutine. The `go` keyword starts a new concurrent execution path.
		go sceneWorker(jobs, results, &wg)
	}

	// --- Distribute Jobs to Workers ---
	for i, ts := range summary.SceneTimeStamps {
		// Create a job package for each scene.
		job := CreateJob(context.GetContext(), s.Tracer, s.geminiInputTokenCounter, s.geminiOutputTokenCounter, s.geminiRetryCounter, i, s.GetName(), summaryText, exampleText, *s.promptTemplate, videoFile, s.generativeAIModel, ts)
		// Send the job into the jobs channel. One of the available workers will pick it up.
		jobs <- job
	}

	// After sending all jobs, close the jobs channel. This signals to the workers
	// that no more work is coming, and they can exit their `for range` loop.
	close(jobs)

	// Block and wait for all goroutines in the WaitGroup to call `wg.Done()`.
	wg.Wait()

	// Once all workers are done, close the results channel.
	close(results)

	// --- Aggregate Results ---
	sceneData := make([]string, 0, len(summary.SceneTimeStamps))
	// Range over the results channel to collect all the responses.
	for r := range results {
		if r.err != nil {
			s.GetErrorCounter().Add(context.GetContext(), 1)
			context.AddError(s.GetName(), r.err)
		} else {
			// Append the successful scene data to the list.
			sceneData = append(sceneData, r.value)
		}
	}

	if !context.HasErrors() {
		s.GetSuccessCounter().Add(context.GetContext(), 1)
	}

	// Place the final list of scene strings into the context.
	context.Add(s.GetOutputParam(), sceneData)
	context.Add(cor.CtxOut, sceneData)
}

// SceneResponse is a simple struct to pass results or errors back from a worker.
type SceneResponse struct {
	value string
	err   error
}

// SceneJob encapsulates all the data needed for a single worker to process one scene.
type SceneJob struct {
	workerId                 int
	ctx                      goctx.Context
	geminiInputTokenCounter  metric.Int64Counter
	geminiOutputTokenCounter metric.Int64Counter
	geminiRetryCounter       metric.Int64Counter
	timeSpan                 *model.TimeSpan
	span                     trace.Span
	contents                 []*genai.Content
	model                    *cloud.QuotaAwareGenerativeAIModel
	err                      error
}

// Close ends the OpenTelemetry span associated with this job.
func (s *SceneJob) Close(status codes.Code, description string) {
	s.span.SetStatus(status, description)
	s.span.End()
}

// CreateJob is a helper function to construct a SceneJob.
func CreateJob(
	ctx goctx.Context,
	tracer trace.Tracer,
	geminiInputTokenCounter metric.Int64Counter,
	geminiOutputTokenCounter metric.Int64Counter,
	geminiRetryCounter metric.Int64Counter,
	workerId int,
	commandName string,
	summaryText string,
	exampleText string,
	template template.Template,
	videoFile *genai.File,
	model *cloud.QuotaAwareGenerativeAIModel,
	timeSpan *model.TimeSpan,
) *SceneJob {
	// Start a new OTel span for this specific scene processing task.
	sceneCtx, sceneSpan := tracer.Start(ctx, fmt.Sprintf("%s_genai_scene_%d", commandName, workerId))
	sceneSpan.SetAttributes(
		attribute.Int("sequence", workerId),
		attribute.String("start", timeSpan.Start),
		attribute.String("end", timeSpan.End),
	)

	// Prepare the data for the prompt template.
	vocabulary := make(map[string]string)
	vocabulary["SEQUENCE"] = fmt.Sprintf("%d", workerId+1)
	vocabulary["SUMMARY_DOCUMENT"] = summaryText
	vocabulary["TIME_START"] = timeSpan.Start
	vocabulary["TIME_END"] = timeSpan.End
	vocabulary["EXAMPLE_JSON"] = exampleText

	// Execute the template to generate the final prompt string.
	var doc bytes.Buffer
	err := template.Execute(&doc, vocabulary)
	if err != nil {
		return &SceneJob{err: err}
	}
	tsPrompt := doc.String()

	//Muziris Change: To accomodate how the new genai golan libaries work

	// // Assemble the multimodal parts for the Gemini API call.
	// parts := []*genai.Part {
	// 	cloud.NewFileData(videoFile.URI, videoFile.MIMEType),
	// 	cloud.NewTextPart(tsPrompt),
	// }

	//Muziris Change
	// Prepare the parts for the multi-modal request to Gemini.
	contents := []*genai.Content{
		{Parts: []*genai.Part{
			{Text: tsPrompt},
			{FileData: &genai.FileData{
				FileURI:  videoFile.URI,
				MIMEType: videoFile.MIMEType,
			}},
		},
			Role: "user"},
	}

	return &SceneJob{
		workerId:                 workerId,
		ctx:                      sceneCtx,
		geminiInputTokenCounter:  geminiInputTokenCounter,
		geminiOutputTokenCounter: geminiOutputTokenCounter,
		geminiRetryCounter:       geminiRetryCounter,
		timeSpan:                 timeSpan,
		span:                     sceneSpan,
		contents:                 contents,
		model:                    model,
	}
}

// sceneWorker is the function that each concurrent goroutine runs.
// It receives jobs from the `jobs` channel and sends results to the `results` channel.
//
// Inputs:
//   - jobs: A <-chan (receive-only) channel for getting `SceneJob` pointers.
//   - results: A chan<- (send-only) channel for sending `SceneResponse` pointers.
//   - wg: A pointer to the WaitGroup to signal when this worker is finished.
func sceneWorker(jobs <-chan *SceneJob, results chan<- *SceneResponse, wg *sync.WaitGroup) {
	// Defer `wg.Done()` to ensure it's called when the function exits, decrementing the WaitGroup counter.
	defer wg.Done()

	// This `for range` loop will automatically wait for a job to become available
	// on the `jobs` channel. When the channel is closed and empty, the loop terminates.
	for j := range jobs {
		if j.err != nil {
			results <- &SceneResponse{value: "", err: j.err}
			continue // Skip to the next job.
		}

		// Call the generative model to get the scene description.
		out, err := cloud.GenerateMultiModalResponse(j.ctx, j.geminiInputTokenCounter, j.geminiOutputTokenCounter, j.geminiRetryCounter, 0, j.model, j.contents)
		if err != nil {
			j.Close(codes.Error, "scene extract failed")
			results <- &SceneResponse{err: err}
			continue
		}

		// Only add the result if the model returned non-empty content.
		if len(strings.TrimSpace(out)) > 0 && out != "{}" {
			results <- &SceneResponse{value: out, err: nil}
		}

		j.Close(codes.Ok, "completed scene")
	}
}
