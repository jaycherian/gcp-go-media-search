#!/bin/bash
# 6-terraform.sh

# This script will create the tfvars file with inputs given by the user and then run terraform.

# --- Helper Functions for UI ---

# Prints a header for a section
print_header() {
    echo ""
    echo "======================================================================"
    echo "$1"
    echo "======================================================================"
}

# Prints an error message and exits
exit_with_error() {
    echo ""
    echo "❌ ERROR: $1" >&2
    exit 1
}

# --- 1. Get Project ID ---
print_header "Step 1: Project ID"
# Try to get the project from the current gcloud config
CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)
if [[ -n "$CURRENT_PROJECT" ]]; then
    read -p "Enter your Project ID [default: $CURRENT_PROJECT]: " PROJECT_ID
    PROJECT_ID=${PROJECT_ID:-$CURRENT_PROJECT}
else
    read -p "Enter your Project ID: " PROJECT_ID
    if [[ -z "$PROJECT_ID" ]]; then
        exit_with_error "Project ID cannot be empty."
    fi
fi
# Set the project for the rest of the script's gcloud commands
gcloud config set project "$PROJECT_ID" || exit_with_error "Failed to set project to '$PROJECT_ID'. Please check if the project exists and you have permissions."
echo "✅ Project set to: $PROJECT_ID"


# --- 2. Get Storage Buckets ---
print_header "Step 2: Create tfvars file for terraform"
read -p "Enter a name for the Low Res storage bucket: " LOW_RES_BUCKET
if [[ -z "$LOW_RES_BUCKET" ]]; then
    exit_with_error "Low Res bucket name cannot be empty."
fi
read -p "Enter a name for the High Res storage bucket: " HIGH_RES_BUCKET
if [[ -z "$HIGH_RES_BUCKET" ]]; then
    exit_with_error "High Res bucket name cannot be empty."
fi
# Output a tfvars file
cat <<EOF > ../terraform/terraform.tfvars
project_id = "$PROJECT_ID"
low_res_bucket = "$LOW_RES_BUCKET"
high_res_bucket = "$HIGH_RES_BUCKET"
EOF
echo "✅ .tfvars file created"

exit




# --- 3. Get Zone and Region ---
print_header "Step 3: Zone Selection"
echo "Fetching available zones... (this may take a moment)"
# Get a list of zones and let the user choose
ZONES=($(gcloud compute zones list --format="value(name)" | sort))
PS3="Please select a zone for your VM: "
select ZONE in "${ZONES[@]}"; do
    if [[ -n "$ZONE" ]]; then
        # Derive region from the zone (e.g., "us-central1-a" -> "us-central1")
        REGION=${ZONE%-*}
        echo "✅ Zone selected: $ZONE"
        echo "✅ Region automatically set to: $REGION"
        break
    else
        echo "Invalid selection. Please try again."
    fi
done

