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

// Package model defines the core data structures for the application.
// This file, `transient.go`, contains struct definitions for data models that
// are primarily used for in-memory operations during the execution of a workflow.
// These objects are considered "transient" because they are not intended to be
// directly persisted to the database in their current form. Instead, they serve
// as intermediate containers for data as it's being processed, transformed,
// and passed between different commands in a chain of responsibility.
package model

// These objects are used in memory via workflows, but are not persisted to the dataset

// MediaFormatFilter is a simple struct used to define the desired output format
// and dimensions for a media transcoding operation, such as resizing a video with FFmpeg.
type MediaFormatFilter struct {
	Format string // e.g., "mp4", "webm"
	Width  string // e.g., "240", "480"
}

// TimeSpan represents a simple time range with a start and end point.
// It is used within the MediaSummary struct to hold the scene timestamps
// extracted by the generative AI before they are processed into full Scene objects.
type TimeSpan struct {
	Start string `json:"start"` // The start time of the span, typically in "HH:MM:SS" format.
	End   string `json:"end"`   // The end time of the span, typically in "HH:MM:SS" format.
}

// MediaSummary is an intermediate data structure that holds the initial, high-level
// information extracted from a media file by the generative AI. It's the first
// structured representation of the AI's analysis. The data from this struct is
// later used to populate the more detailed, persistent `Media` struct.
type MediaSummary struct {
	Title           string        `json:"title"`                       // The main title of the media.
	Category        string        `json:"category"`                    // The category of the media (e.g., "trailer", "movie").
	Summary         string        `json:"summary"`                     // The AI-generated summary of the media's content.
	LengthInSeconds int           `json:"length_in_seconds"`           // The total duration of the media in seconds.
	MediaUrl        string        `json:"media_url,omitempty"`         // The GCS URL of the media file. The `omitempty` tag means this field will be omitted from the JSON if it's empty.
	Director        string        `json:"director,omitempty"`          // The director of the media.
	ReleaseYear     int           `json:"release_year,omitempty"`      // The year the media was released.
	Genre           string        `json:"genre,omitempty"`             // The genre of the media.
	Rating          string        `json:"rating,omitempty"`            // The content rating (e.g., "G", "PG-13").
	Cast            []*CastMember `json:"cast,omitempty"`              // A list of cast members (character and actor names).
	SceneTimeStamps []*TimeSpan   `json:"scene_time_stamps,omitempty"` // A list of time spans for each identified scene. This is used to guide the next step of scene-by-scene script extraction.
}

// SceneMatchResult is a lightweight struct used to hold the results from a
// BigQuery VECTOR_SEARCH. It contains the primary keys needed to retrieve the
// full media and scene data from the main `media` table.
type SceneMatchResult struct {
	MediaId        string `json:"media_id" bigquery:"media_id"`               // The unique ID of the media file that contains the matching scene.
	SequenceNumber int    `json:"sequence_number" bigquery:"sequence_number"` // The sequence number of the specific scene that matched the search query.
}
