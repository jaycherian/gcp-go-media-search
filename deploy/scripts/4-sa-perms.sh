#!/bin/bash
# 4-sa-perms.sh

# Check if a service account email is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <service_account_email>"
  exit 1
fi

SERVICE_ACCOUNT=$1
PROJECT_ID=$(gcloud config get-value project)

if [ -z "$PROJECT_ID" ]; then
  echo "GCP project not set. Please run 'gcloud config set project <your-project-id>'"
  exit 1
fi

echo "Assigning roles to $SERVICE_ACCOUNT on project $PROJECT_ID"

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
  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="$ROLE" \
    --condition=None > /dev/null
done

echo "All roles have been assigned successfully."