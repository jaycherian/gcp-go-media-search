#!/bin/bash
# 4-create-sa.sh

# This script will create a service account for the Media Search system

# Inputs and defaults
SA_ID=${1:-"media-search-sa"}
SA_NAME=${2:-"Media Search service account"}
PROJECT=${3:-$(gcloud config get project)}

PROJECT_NUM=$(gcloud projects describe $PROJECT --format="value(projectNumber)")

# --- 2. Create the service account ---
NEW_SA_EMAIL=$(gcloud iam service-accounts create "$SA_ID" --display-name="$SA_NAME" --format json | jq -r .email)
echo "✅ Media Search service account created: "
echo "- Id: $SA_ID"
echo "- Email: $NEW_SA_EMAIL"

# --- 3. Create the Vertex AI service account if it doesn't exist ---
echo "Creating Vertex AI service account if it doesn't exist..."
gcloud beta services identity create --service=aiplatform.googleapis.com 
gcloud projects add-iam-policy-binding $PROJECT \
    --member=serviceAccount:service-$PROJECT_NUM@gcp-sa-aiplatform.iam.gserviceaccount.com \
    --role=roles/aiplatform.serviceAgent \
    --condition=None > /dev/null

echo "✅ Vertex AI Service Agent created."


