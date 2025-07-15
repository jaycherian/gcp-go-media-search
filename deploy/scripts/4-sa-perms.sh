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

# Check if the Cloud Resource Manager API is enabled
echo "Checking if the Cloud Resource Manager API is enabled for project $PROJECT_ID..."
API_ENABLED=$(gcloud services list --enabled --project="$PROJECT_ID" --filter="config.name:cloudresourcemanager.googleapis.com" --format="value(config.name)")

if [[ -z "$API_ENABLED" ]]; then
  echo "Cloud Resource Manager API is not enabled. Enabling now..."
  gcloud services enable cloudresourcemanager.googleapis.com --project="$PROJECT_ID"
  if [ $? -ne 0 ]; then
    echo "Failed to enable Cloud Resource Manager API. Please check your permissions and try again."
    exit 1
  fi
  echo "API enabled successfully."
else
  echo "Cloud Resource Manager API is already enabled."
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