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
data "google_storage_project_service_account" "gcs_account" {
}

########
# This module can create the infra need for the high and low res buckets + pubsub
# var.type should be one of 'high_res' or 'low_res'
########

resource "google_pubsub_topic" "media_events" {
  name = "media_${var.type}_events"
  message_storage_policy {
    allowed_persistence_regions = [var.region]
  }
}

resource "google_pubsub_topic" "media_events_dead_letter" {
  name = "media_${var.type}_events_dead_letter"
  message_storage_policy {
    allowed_persistence_regions = [var.region]
  }
}

resource "google_storage_bucket" "media_bucket" {
  name          = var.bucket_name
  location      = var.region
  uniform_bucket_level_access = true
  force_destroy = true
  public_access_prevention = "enforced"
  versioning {
    enabled = true
  }
  logging {
    log_bucket = "media_logs"
    log_object_prefix = "media-logs"
  }
}

resource "google_pubsub_subscription" "media_bucket_subscription" {
  name  = "media_${var.type}_resources_subscription"
  topic = google_pubsub_topic.media_events.id

  # Enable exactly-once delivery by enabling message ordering
  enable_message_ordering = true

  # Configure retry policy for failed message delivery attempts
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  # Configure dead-letter policy to handle messages that cannot be delivered
 dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.media_events_dead_letter.id
    max_delivery_attempts = 10
  }
}

resource "google_storage_notification" "media_bucket_notifications" {
  bucket         = google_storage_bucket.media_bucket.name
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.media_events.id
  event_types    = ["OBJECT_FINALIZE", "OBJECT_METADATA_UPDATE"]
  custom_attributes = {
    new-attribute = "new-attribute-value"
  }
  depends_on = [google_pubsub_topic_iam_binding.topic_binding]
}

resource "google_pubsub_topic_iam_binding" "topic_binding" {
  topic   = google_pubsub_topic.media_events.id
  role    = "roles/pubsub.publisher"
  members = ["serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"]
}