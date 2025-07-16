#!/bin/bash
# 4-create-sa.sh

# This script will create a service account for the Media Search system

# Inputs and defaults
SA_ID=${1:-"media-search-sa"}
SA_NAME=${2:-"Media Search service account"}
PROJECT=${3:-$(gcloud config get project)}

# --- 2. Create the service account ---
NEW_SA_EMAIL=$(gcloud iam service-accounts create "$SA_ID" --display-name="$SA_NAME" --format json | jq -r .email)
echo "✅ Service account created: "
echo "- Id: $SA_ID"
echo "- Email: $NEW_SA_EMAIL"


