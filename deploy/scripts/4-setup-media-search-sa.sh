#!/bin/bash
# 4-setup-media-search-sa.sh

# This script will create a service account for the Media Search system and give it the roles it needs

# Inputs and defaults
SA_ID=${1:-"media-search-sa"}
SA_NAME=${2:-"Media Search service account"}
PROJECT=${3:-$(gcloud config get project)}

# --- 2. Create the service account ---
NEW_SA_EMAIL=$(gcloud iam service-accounts create "$SA_ID" --display-name="$SA_NAME" --format json | jq -r .email)
echo "✅ Media Search service account created: "
echo "- Id: $SA_ID"
echo "- Email: $NEW_SA_EMAIL"

# --- 3. Assign roles to the service account ---
echo "Granting roles to $NEW_SA_EMAIL on project $PROJECT"

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
    --member="serviceAccount:$SA_EMAIL" \
    --role="$ROLE" \
    --condition=None > /dev/null
done

echo "✅ All roles have been granted."