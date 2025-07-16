#!/bin/bash
# 5-setup-sa.sh

# This script will create a service account for the Media Search system

# Inputs and defaults
PROJECT=${1:-$(gcloud config get project)}

PROJECT_NUM=$(gcloud projects describe $PROJECT --format="value(projectNumber)")
SA_EMAIL="service-$PROJECT_NUM@gcp-sa-aiplatform.iam.gserviceaccount.com"

# --- 1. Create the Vertex AI service agent if it doesn't exist ---
echo "Creating Vertex AI service account if it doesn't exist..."
gcloud beta services identity create --service=aiplatform.googleapis.com 
echo "✅ Vertex AI Service Agent created."

# --- 2. Assign roles to the service agent ---
echo "Assigning roles to Vertext AI Service Agent on project $PROJECT"

# Array of roles to be assigned
ROLES=(
  "roles/aiplatform.serviceAgent"
  "roles/storage.objectAdmin"
  "roles/bigquery.dataViewer"
  "roles/bigquery.jobUser"
  "roles/pubsub.admin"
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


