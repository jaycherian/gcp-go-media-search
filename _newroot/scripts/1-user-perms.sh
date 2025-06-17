#!/bin/bash
# 1-user-perms.sh
# ==============================================================================
# GCP IAM Role Granting Script
#
# Description:
#   This script grants a predefined list of IAM roles to a specified user
#   for a given GCP project. It is designed to be run from the GCP Cloud Shell.
#
# Author:
#   Gemini
#
# Version:
#   1.0
#
# Instructions:
#   1. Open Google Cloud Shell.
#   2. Create a new file named `grant_roles.sh`.
#   3. Copy and paste the content of this script into the file.
#   4. Make the script executable by running: chmod +x grant_roles.sh
#   5. Execute the script by running: ./grant_roles.sh
#   6. Follow the on-screen prompts.
# ==============================================================================


# --- Start of Configuration ---

# List of IAM roles to be granted.
# You can add or remove roles from this list as needed.
ROLES_TO_GRANT=(
   "roles/aiplatform.admin"
   "roles/aiplatform.user"
   "roles/aiplatform.viewer"
   "roles/automl.predictor"
   "roles/billing.projectManager"
   "roles/compute.admin"
   "roles/compute.instanceAdmin.v1"
   "roles/compute.osAdminLogin"
   "roles/compute.osLogin"
   "roles/compute.storageAdmin"
   "roles/editor"
   "roles/iam.serviceAccountUser"
   "roles/owner"
   "roles/pubsub.admin"
   "roles/pubsub.editor"
   "roles/resourcemanager.projectIamAdmin"
   "roles/resourcemanager.projectMover"
   "roles/storage.admin"
   "roles/storage.objectAdmin"
   "roles/storage.objectCreator"
   "roles/storage.objectUser"
   "roles/storage.objectViewer"
   "roles/visionai.admin"
   "roles/visionai.analysisEditor"
   "roles/visionai.analysisViewer"
   "roles/visionai.annotationViewer"
   "roles/visionai.applicationEditor"
   "roles/visionai.assetEditor"
   "roles/visionai.editor"
   "roles/visionai.viewer"
)

# --- End of Configuration ---

# --- Color Definitions for Enhanced Output ---
COLOR_RESET='\033[0m'
COLOR_RED='\033[0;31m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_BLUE='\033[0;34m'
COLOR_CYAN='\033[0;36m'

# --- Function Definitions ---

# Function to print a separator line for better readability.
print_separator() {
   printf "\n%s\n" "================================================================================"
}

# Function to log informational messages.
log_info() {
   echo -e "${COLOR_BLUE}[INFO]${COLOR_RESET} $1"
}

# Function to log success messages.
log_success() {
   echo -e "${COLOR_GREEN}[SUCCESS]${COLOR_RESET} $1"
}

# Function to log warning messages.
log_warning() {
   echo -e "${COLOR_YELLOW}[WARNING]${COLOR_RESET} $1"
}

# Function to log error messages and exit.
log_error_and_exit() {
   echo -e "${COLOR_RED}[ERROR]${COLOR_RESET} $1" >&2
   exit 1
}

# Function to check for the presence of the gcloud command-line tool.
check_gcloud_prerequisites() {
   log_info "Checking for necessary prerequisites..."
   if ! command -v gcloud &> /dev/null; then
       log_error_and_exit "gcloud command-line tool not found. Please ensure you are running this script in GCP Cloud Shell or have the Google Cloud SDK installed and configured."
   fi


   if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "."; then
       log_error_and_exit "You are not authenticated with gcloud. Please run 'gcloud auth login' first."
   fi
   log_success "Prerequisites check passed."
}

# Main function to orchestrate the script's execution.
main() {
   clear
   print_separator
   echo -e "${COLOR_CYAN}         GCP IAM Role Granting Utility${COLOR_RESET}"
   print_separator

   check_gcloud_prerequisites

   # --- User Input ---
   print_separator
   log_info "Please enter the GCP Project ID and the user's email address."

   read -p "Enter GCP Project ID: " PROJECT_ID
   # Validate that PROJECT_ID is not empty.
   if [[ -z "$PROJECT_ID" ]]; then
       log_error_and_exit "Project ID cannot be empty."
   fi

   read -p "Enter user email (e.g., user@example.com): " USER_EMAIL
   # Basic email format validation.
   if ! [[ "$USER_EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
       log_error_and_exit "Invalid email address format."
   fi

   # --- Project Validation ---
   print_separator
   log_info "Validating project '$PROJECT_ID'..."
   if ! gcloud projects describe "$PROJECT_ID" &> /dev/null; then
       log_error_and_exit "Project '$PROJECT_ID' not found or you do not have permission to access it."
   fi
   log_success "Project '$PROJECT_ID' is valid."

   # --- Confirmation ---
   print_separator
   log_warning "You are about to grant the following roles to '$USER_EMAIL' on project '$PROJECT_ID':"
   for role in "${ROLES_TO_GRANT[@]}"; do
       echo "  - $role"
   done
   print_separator

   read -p "Are you sure you want to proceed? (y/N): " CONFIRMATION
   if [[ ! "$CONFIRMATION" =~ ^[Yy]$ ]]; then
       log_info "Operation cancelled by user."
       exit 0
   fi

   # --- Granting Roles ---
   print_separator
   log_info "Setting current project to '$PROJECT_ID'..."
   gcloud config set project "$PROJECT_ID"

   log_info "Starting to grant roles to '$USER_EMAIL'..."
   for role in "${ROLES_TO_GRANT[@]}"; do
       echo -ne "  - Granting ${COLOR_YELLOW}${role}${COLOR_RESET}..."
       # The gcloud command is idempotent. If the binding already exists, it will not fail.
       # We capture the output to keep the script's output clean.
       if gcloud projects add-iam-policy-binding "$PROJECT_ID" \
           --member="user:$USER_EMAIL" \
           --role="$role" --condition=None &> /dev/null; then
           echo -e " ${COLOR_GREEN}Done.${COLOR_RESET}"
       else
           echo -e " ${COLOR_RED}Failed.${COLOR_RESET}"
           log_warning "Could not grant role '$role'. This might be due to permissions issues."
       fi
       sleep 0.2 # Small delay for visual effect
   done

   # --- Final Summary ---
   print_separator
   log_success "All specified roles have been processed for '$USER_EMAIL' on project '$PROJECT_ID'."
   log_info "To verify the new permissions, you can visit the IAM page in the GCP Console."
   print_separator
}

# --- Script Execution Start ---
# Encapsulate the main logic in a function and call it.
# This is a best practice for shell scripting.
main
