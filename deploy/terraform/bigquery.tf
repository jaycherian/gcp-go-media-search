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

# TODO for production we may want to add customer managed encryption

resource "google_bigquery_dataset" "media_ds" {
  dataset_id                  = "media_ds"
  description                 = "Media data source for media file object"
  location                    = "US"
  delete_contents_on_destroy  = true
  max_time_travel_hours = 96
  labels = {
    env = "test"
  }
}

resource "google_bigquery_table" "media_ds_scene_embeddings" {
  dataset_id = google_bigquery_dataset.media_ds.dataset_id
  table_id   = "scene_embeddings"
  deletion_protection = false
  schema = <<EOF
[
    {
        "name": "media_id",
        "type": "STRING",
        "mode": "REQUIRED"
    },
    {
        "name": "model_name",
        "type": "STRING",
        "mode": "REQUIRED"
    },
    {
        "name": "sequence_number",
        "type": "INTEGER",
        "mode": "REQUIRED"
    },
    {
        "name": "embeddings",
        "type": "FLOAT64",
        "mode": "REPEATED"
    }
]
EOF
}

resource "google_bigquery_table" "media_ds_media" {
  dataset_id = google_bigquery_dataset.media_ds.dataset_id
  table_id   = "media"
  deletion_protection = false
  schema = <<EOF
[
    {
        "name": "id",
        "type": "STRING",
        "mode": "REQUIRED"
    },
    {
        "name": "create_date",
        "type": "TIMESTAMP",
        "mode": "REQUIRED"
    },
    {
        "name": "title",
        "type": "STRING",
        "mode": "REQUIRED"
    },
    {
        "name": "category",
        "type": "STRING",
        "mode": "REQUIRED"
    },
    {
        "name": "summary",
        "type": "STRING",
        "mode": "REQUIRED"
    },
    {
        "name": "media_url",
        "type": "STRING",
        "mode": "REQUIRED"
    },
    {
        "name": "length_in_seconds",
        "type": "INTEGER",
        "mode": "NULLABLE"
    },
    {
        "name": "director",
        "type": "STRING",
        "mode": "NULLABLE"
    },
    {
        "name": "release_year",
        "type": "INTEGER",
        "mode": "NULLABLE"
    },
    {
        "name": "genre",
        "type": "STRING",
        "mode": "NULLABLE"
    },
    {
        "name": "rating",
        "type": "STRING",
        "mode": "NULLABLE"
    },
    {
        "name": "cast",
        "type": "RECORD",
        "mode": "REPEATED",
        "fields": [
            {
                "name": "character_name",
                "type": "STRING",
                "mode": "NULLABLE"
            },
            {
                "name": "actor_name",
                "type": "STRING",
                "mode": "NULLABLE"
            }
        ]
    },
    {
        "name": "scenes",
        "type": "RECORD",
        "mode": "REPEATED",
        "fields": [
            {
                "name": "sequence",
                "type": "INTEGER",
                "mode": "REQUIRED"
            },
            {
                "name": "tokens_to_generate",
                "type": "INTEGER",
                "mode": "NULLABLE"
            },
            {
                "name": "tokens_generated",
                "type": "INTEGER",
                "mode": "NULLABLE"
            },
            {
                "name": "start",
                "type": "STRING",
                "mode": "NULLABLE"
            },
            {
                "name": "end",
                "type": "STRING",
                "mode": "NULLABLE"
            },
            {
                "name": "script",
                "type": "STRING",
                "mode": "NULLABLE"
            }
        ]
    }
]
EOF
}