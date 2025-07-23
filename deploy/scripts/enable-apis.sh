#!/bin/bash
# 1-user-perms.sh
 
PROJECT=${1:-`gcloud config get-value project`}

PROJECT_NUM=$(gcloud projects describe $PROJECT --format="value(projectNumber)")
VERTEX_SA_EMAIL="service-$PROJECT_NUM@gcp-sa-aiplatform.iam.gserviceaccount.com"

echo "Enabling APIs..."

declare -a apis=(
    "aiplatform.googleapis.com"
    "cloudresourcemanager.googleapis.com"
    "compute.googleapis.com"
    "iam.googleapis.com"
    "pubsub.googleapis.com"
    "storage.googleapis.com"
)

for api in "${apis[@]}"
do
    # enable the current API and output how long it took in milliseconds
    echo "Enabling: $api"

    # if on a mac, you need GNU time (`gtime`) installed with: brew install gtime
    # if on regular linux, just use regular `time` instead.
    #(gtime -f "%e" gcloud services enable $api --project $project) 2>&1 | xargs printf "Finished in %.0fs\n"

    # just run the command without timing as the above is not cross platform compatible
    gcloud services enable $api --project $PROJECT

done

echo "✅ APIs enabled."

# Create a service agent for Vertex AI api

echo "Creating Vertex AI service agent if it doesn't exist..."
gcloud beta services identity create --service=aiplatform.googleapis.com 
echo "✅ Vertex AI Service Agent created."

echo "Waiting 60 seconds for service agent to propagate before assigning IAM roles..."
sleep 60

echo "Assigning roles to Vertex AI Service Agent on project $PROJECT"

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
    --member=serviceAccount:$VERTEX_SA_EMAIL \
    --role="$ROLE" \
    --condition=None > /dev/null
done

echo "✅ Vertex AI Service Agent roles granted."
