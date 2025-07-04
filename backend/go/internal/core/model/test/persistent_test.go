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

// Package model_test contains unit tests for the data models defined in the
// model package. This file specifically tests the constructors and initial
// state of the persistent data models (`Media` and `SceneEmbedding`).
package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaycherian/gcp-go-media-search/internal/core/model"
	"github.com/stretchr/testify/assert"
)

// TestNewMedia tests the constructor for the Media struct.
// It verifies that the ID is generated correctly using a UUIDv5 hash of the
// file name, that the creation timestamp is set to the current time, and that
// the slice fields (Cast, Scenes) are initialized as empty slices.
func TestNewMedia(t *testing.T) {
	// Define a sample file name to be used for ID generation.
	fileName := "test-file.mp4"
	// Call the constructor to create a new Media object.
	media := model.NewMedia(fileName)

	// To verify the ID, we must generate the same UUIDv5 hash that the
	// constructor is expected to create. This uses the URL namespace and the
	// file name as the input byte slice.
	generatedID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fileName))

	// Assert that the generated ID in the Media object matches our expected ID.
	assert.Equal(t, generatedID.String(), media.Id)
	// Assert that the creation date is very recent (within one second of now).
	assert.WithinDuration(t, time.Now(), media.CreateDate, time.Second)
	// Assert that the Cast slice is initialized and empty.
	assert.Equal(t, 0, len(media.Cast))
	// Assert that the Scenes slice is initialized and empty.
	assert.Equal(t, 0, len(media.Scenes))
}

// TestNewSceneEmbedding tests the constructor for the SceneEmbedding struct.
// It ensures that the provided media ID, sequence number, and model name are
// correctly assigned to the struct's fields and that the Embeddings slice is
// properly initialized as an empty slice.
func TestNewSceneEmbedding(t *testing.T) {
	// Define sample input data for the constructor.
	mediaId := "test-media-id"
	sequenceNumber := 1
	modelName := "test-model"

	// Call the constructor with the test data.
	embedding := model.NewSceneEmbedding(mediaId, sequenceNumber, modelName)

	// Assert that each field was assigned the correct value.
	assert.Equal(t, mediaId, embedding.Id)
	assert.Equal(t, sequenceNumber, embedding.SequenceNumber)
	assert.Equal(t, modelName, embedding.ModelName)
	// Assert that the Embeddings slice is initialized and empty.
	assert.Equal(t, 0, len(embedding.Embeddings))
}
