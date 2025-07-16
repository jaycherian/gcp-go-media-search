#!/bin/bash
# 5-setup-sa.sh

# This script will create a service account for the Media Search system

# Inputs and defaults
PROJECT=${1:-$(gcloud config get project)}

PROJECT_NUM=$(gcloud projects describe $PROJECT --format="value(projectNumber)")
SA_EMAIL="$PROJECT_NUM-compute@developer.gserviceaccount.com"

# --- 1. Assign roles to the Compute service account ---
echo "Assigning roles to Vertext AI Service Agent on project $PROJECT"

# Array of roles to be assigned
ROLES=(
  "roles/bigquery.admin"
  "roles/pubsub.admin"
  "roles/storage.admin"
  "roles/storage.objectAdmin"
  "roles/telemetry.metricsWriter"
  "roles/cloudtrace.admin"
  "roles/cloudtrace.agent"
  "roles/cloudtrace.user"
  "roles/monitoring.metricWriter"
  "roles/monitoring.metricsScopesAdmin"
)

# Loop through the roles and assign them to the service account
for ROLE in "${ROLES[@]}"
do
  echo "Granting $ROLE..."
  gcloud projects add-iam-policy-binding "$PROJECT" \
    --member=serviceAccount:$SA_EMAIL \
    --role="$ROLE" \
    --condition=None > /dev/null
done

echo "✅ Vertex AI Service Agent roles granted."


