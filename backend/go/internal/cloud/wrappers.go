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

// Package cloud provides components for interacting with Google Cloud services.
// This file implements a wrapper around the standard Generative AI client.
// This wrapper uses the Decorator design pattern to add extra functionality
// to an existing object without altering its code. Specifically, it adds
// rate limiting and a retry mechanism to the Generative AI model.
//
// Why this is important:
//   - Rate Limiting: Services like Vertex AI have quotas on how many requests
//     you can make per minute. This wrapper prevents the application from
//     exceeding those limits, which would otherwise result in errors.
//   - Retry Logic: Network requests can sometimes fail for transient reasons.
//     The wrapper automatically retries a failed request, making the application
//     more resilient and reliable.
//
// Structs:
//   - QuotaAwareGenerativeAIModel: A struct that wraps the base `genai.GenerativeModel`
//     and adds a rate limiter.
//
// Functions:
//   - NewQuotaAwareModel: A constructor to create a new instance of the wrapped model.
//   - GenerateContent: An overridden method that intercepts calls to the AI model
//     to enforce rate limiting and retries.
package cloud

import (
	"context"
	"errors"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/genai"
)

// QuotaAwareGenerativeAIModel is a decorator struct that wraps the standard
// `genai.GenerativeModel` to add rate-limiting capabilities. By embedding
// the original model, it inherits all its methods, but we can override specific
// ones, like `GenerateContent`, to add our custom logic.
type QuotaAwareGenerativeAIModel struct {
	GenerativeContentConfig *genai.GenerateContentConfig // The embedded base Vertex AI LLM. All its fields and methods are available here.
	ModelName               string
	ModelHandle             *genai.Models
	RateLimit               rate.Limiter // A rate limiter from Go's standard library to control request frequency.
}

// NewQuotaAwareModel is a constructor function that creates a new
// QuotaAwareGenerativeAIModel. It takes the base model and a rate limit
// (in requests per second) and returns our enhanced, quota-aware model.
//
// Inputs:
//   - wrapped: The original *genai.GenerativeModel to be wrapped.
//   - requestsPerSecond: An integer specifying the maximum number of API calls allowed per second.
//
// Outputs:
//   - *QuotaAwareGenerativeAIModel: A pointer to the newly created wrapper.
func NewQuotaAwareModel(wrapped *genai.GenerateContentConfig, name string, ModelHand *genai.Models, requestsPerSecond int) *QuotaAwareGenerativeAIModel {
	return &QuotaAwareGenerativeAIModel{
		GenerativeContentConfig: wrapped,
		ModelName:               name,
		ModelHandle:             ModelHand,
		// Creates a new rate limiter that allows a burst of `requestsPerSecond` events
		// and replenishes the "token bucket" at a rate of 1 token per second.
		RateLimit: *rate.NewLimiter(rate.Every(time.Second/1), requestsPerSecond),
	}
}

// GenerateContent overrides the original `GenerateContent` method of the embedded
// `genai.GenerativeModel`. This is where the rate-limiting and retry logic is implemented.
//
// Logic Flow:
//  1. Check the rate limiter.
//  2. If a request is allowed:
//     a. Call the original `GenerateContent` method.
//     b. If it fails, check the retry count.
//     c. If retries are available, wait and recursively call itself to try again.
//     d. If no retries are left, return the error.
//  3. If a request is NOT allowed (rate-limited):
//     a. Wait for a short period.
//     b. Recursively call itself to re-queue the request.
//
// Inputs:
//   - ctx: The context for the request. It's used here to manage retry state.
//   - parts: The parts of the multi-modal prompt (text, images, etc.).
//
// Outputs:
//   - *genai.GenerateContentResponse: The response from the AI model if successful.
//   - error: An error if the request fails after all retries or if another issue occurs.
func (q *QuotaAwareGenerativeAIModel) GenerateContent(ctx context.Context, content []*genai.Content) (resp *genai.GenerateContentResponse, err error) {
	// The `Allow()` method checks if an event can happen now. It's a non-blocking check.
	if q.RateLimit.Allow() {
		// If allowed, proceed to call the actual Generative AI model.
		resp, err = q.ModelHandle.GenerateContent(ctx, q.ModelName, content, q.GenerativeContentConfig)
		if err != nil {
			// If an error occurred during the API call, start the retry logic.
			// Get the current retry count from the context. `Value()` returns an interface{},
			// so we must type-assert it to an `int`.
			retryCount, ok := ctx.Value("retry").(int)
			if !ok {
				// This is the first attempt.
				retryCount = 0
			}
			if retryCount > 3 {
				// If we have exceeded the maximum number of retries, give up and return an error.
				return nil, errors.New("failed generation on max retries")
			}
			// If more retries are allowed, create a new context with an incremented retry count.
			errCtx := context.WithValue(ctx, "retry", retryCount+1)
			// Wait for one minute before retrying to give the service time to recover.
			time.Sleep(time.Minute * 1)
			// Recursively call this function to try again.
			return q.ModelHandle.GenerateContent(errCtx, q.ModelName, content, q.GenerativeContentConfig)
		}
		// If the API call was successful, return the response and a nil error.
		return resp, err
	} else {
		// If the rate limiter did not allow the request, wait for 5 seconds.
		// This pauses the execution of this specific request, effectively "queueing" it.
		time.Sleep(time.Second * 5)
		// After waiting, recursively call this function to try obtaining a token from the rate limiter again.
		return q.ModelHandle.GenerateContent(ctx, q.ModelName, content, q.GenerativeContentConfig)
	}
}
