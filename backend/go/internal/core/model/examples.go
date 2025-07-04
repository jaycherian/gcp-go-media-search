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

// Package model defines the data structures for the application. This file,
// `examples.go`, provides factory functions for creating hardcoded, example
// instances of the data models.
//
// These example objects are crucial for "few-shot" prompting with the
// generative AI models. By providing a concrete example of the desired JSON
// output structure within the prompt itself, we guide the AI to return data
// that is consistent, correctly formatted, and easily parsable.
package model

// GetExampleScene creates a sample Scene object. This is used to provide a
// "few-shot" learning example to the generative AI model when it is asked to
// extract scene-specific details. It helps the AI understand the expected
// JSON structure for a single scene, including fields like sequence number,
// start/end times, and the script content.
//
// Outputs:
//   - *Scene: A pointer to a hardcoded Scene object.
func GetExampleScene() *Scene {
	// Instantiate a Scene struct with example data.
	out := &Scene{
		SequenceNumber: 1,
		Start:          "00:00:00",
		End:            "00:01:00",
		Script: `
INT. BATTLEFIELD - DAY

A fierce battle is raging. Soldiers are fighting and dying all around.

VOICEOVER (V.O.) - (Nathan Fillion)
I aim to misbehave.

We see a young woman, RIVER TAM (16), running through the battlefield. She is terrified and covered in blood.

RIVER (V.O.) - (Summar Glau)
They were right. They were always right.

River stumbles and falls. She looks up to see a man standing over her. He is SIMON TAM (26), her older brother.

SIMON - (Sean Maher)
It's all right, River. I'm here.

Simon helps River to her feet. They run away together.`,
	}
	return out
}

// GetExampleSummary creates a sample MediaSummary object. This is used to
// provide a "few-shot" learning example to the generative AI model when it is
// asked to extract the overall summary and metadata for a media file. This
// example shows the AI the expected JSON structure for the entire media summary,
// including nested arrays for cast members and scene timestamps.
//
// Outputs:
//   - *MediaSummary: A pointer to a hardcoded MediaSummary object.
func GetExampleSummary() *MediaSummary {
	// Instantiate a MediaSummary struct with example data for the movie "Serenity".
	s := &MediaSummary{
		Title:           "Serenity",
		Category:        "trailer",
		Summary:         "The crew of the ship Serenity try to evade an assassin sent to recapture telepath River.",
		LengthInSeconds: 120,
		MediaUrl:        "https://storage.mtls.cloud.google.com/bucket_name/Serenity.mp4",
		Director:        "Joss Whedon",
		ReleaseYear:     2005,
		Genre:           "Science Fiction",
		Rating:          "PG-13",
		// Initialize the slices to be non-nil.
		SceneTimeStamps: make([]*TimeSpan, 0),
		Cast:            make([]*CastMember, 0),
	}
	// Append example time spans for scenes. The AI is expected to identify these.
	s.SceneTimeStamps = append(s.SceneTimeStamps, &TimeSpan{Start: "00:00:00", End: "00:00:05"}, &TimeSpan{Start: "00:00:06", End: "00:00:10"})
	// Append example cast members. The AI is expected to extract character and actor names.
	s.Cast = append(s.Cast, &CastMember{CharacterName: "Malcolm Reynolds", ActorName: "Nathan Fillion"})
	s.Cast = append(s.Cast, &CastMember{CharacterName: "River Tam", ActorName: "Summar Glau"})
	s.Cast = append(s.Cast, &CastMember{CharacterName: "Simon Tam", ActorName: "Sean Maher"})
	return s
}
