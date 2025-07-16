#!/bin/bash
# 6-create-sa.sh

# This script will create a service account for you

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
print_header "Step 1: Service Account Name"
# Try to get the project from the current gcloud config
read -p "Enter your service account name [default: $1]: " SA_NAME

# --- 2. Create the service account ---
if [[ -z "$SA_NAME" ]]; then
    SA_NAME=$1
fi
NEW_SA_EMAIL=$(gcloud iam service-accounts create "$SA_NAME" --display-name="$SA_NAME" --format json | jq -r .email)
echo "✅ Service account created: "
echo "- Name: $SA_NAME"
echo "- Email: $NEW_SA_EMAIL"


