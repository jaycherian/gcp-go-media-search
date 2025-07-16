#!/bin/bash
# 5-setup-sa.sh

# This script will create a service account for the Media Search system

# Inputs and defaults
SA_ID=${1:-"media-search-sa"}
PROJECT=${2:-$(gcloud config get project)}

SA_EMAIL="$SA_ID@$PROJECT.iam.gserviceaccount.com"

echo $SA_EMAIL

echo "Assigning roles to $SA_EMAIL on project $PROJECT"

# Array of roles to be assigned
ROLES=(
  "roles/bigquery.admin"
  "roles/telemetry.metricsWriter"
  "roles/cloudtrace.admin"
  "roles/cloudtrace.agent"
  "roles/cloudtrace.user"
  "roles/monitoring.metricWriter"
  "roles/monitoring.metricsScopesAdmin"
  "roles/pubsub.admin"
  "roles/storage.admin"
  "roles/storage.objectAdmin"
  "roles/viewer"
)

# Loop through the roles and assign them to the service account
for ROLE in "${ROLES[@]}"
do
  echo "Assigning $ROLE..."
  gcloud projects add-iam-policy-binding "$PROJECT" \
    --member="serviceAccount:$SA_EMAIL" \
    --role="$ROLE" \
    --condition=None > /dev/null
done

echo "All roles have been assigned successfully."