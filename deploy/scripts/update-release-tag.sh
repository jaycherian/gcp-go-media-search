#!/bin/bash

# A script to update git release tags for both local and remote repositories.
# It deletes and then recreates 'release-latest' and a version-specific tag.
#
# Usage: ./update-release-tag.sh <version>
# Example: ./update-release-tag.sh 0.0.8

# --- Configuration ---
# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Icons for status messages
SUCCESS_ICON="✅"
FAILURE_ICON="❌"
INFO_ICON="ℹ️"
ROCKET_ICON="🚀"

# --- Argument Validation ---
if [ -z "$1" ]; then
    echo -e "${RED}${FAILURE_ICON} Error: Version argument not provided.${NC}"
    echo -e "Usage: $0 <version>"
    echo -e "Example: $0 0.0.8"
    exit 1
fi

VERSION=$1
TAG_VERSION="release-${VERSION}"
TAG_LATEST="release-latest"

# --- Helper Functions ---
print_success() {
    echo -e "${GREEN}${SUCCESS_ICON} $1${NC}"
}

print_info() {
    echo -e "${YELLOW}${INFO_ICON} $1${NC}"
}

print_step() {
    echo -e "\n${BLUE}--- Step $1 of 4: $2 ---${NC}"
}

# --- Main Execution ---
echo -e "${PURPLE}===============================================${NC}"
echo -e "${PURPLE}${ROCKET_ICON} Starting Release Tag Update for v${VERSION} ${ROCKET_ICON}${NC}"
echo -e "${PURPLE}===============================================${NC}"

# Step 1: Delete the 'release-latest' tag
print_step 1 4 "Deleting '${TAG_LATEST}' tag"
# Delete local tag, ignoring errors if it doesn't exist
if git tag -d "${TAG_LATEST}" >/dev/null 2>&1; then
    print_success "Deleted local tag: '${TAG_LATEST}'"
else
    print_info "Local tag '${TAG_LATEST}' not found. No action taken."
fi
# Delete remote tag, ignoring errors if it doesn't exist
if git push origin --delete "${TAG_LATEST}" >/dev/null 2>&1; then
    print_success "Deleted remote tag: '${TAG_LATEST}'"
else
    print_info "Remote tag '${TAG_LATEST}' not found on origin. No action taken."
fi

# Step 2: Delete the version-specific tag
print_step 2 4 "Deleting '${TAG_VERSION}' tag"
# Delete local tag, ignoring errors if it doesn't exist
if git tag -d "${TAG_VERSION}" >/dev/null 2>&1; then
    print_success "Deleted local tag: '${TAG_VERSION}'"
else
    print_info "Local tag '${TAG_VERSION}' not found. No action taken."
fi
# Delete remote tag, ignoring errors if it doesn't exist
if git push origin --delete "${TAG_VERSION}" >/dev/null 2>&1; then
    print_success "Deleted remote tag: '${TAG_VERSION}'"
else
    print_info "Remote tag '${TAG_VERSION}' not found on origin. No action taken."
fi

# From this point, we expect commands to succeed. Exit on any error.
set -e

# Step 3: Create new tags on the current HEAD
print_step 3 4 "Creating new tags"
echo "Tagging current HEAD with '${TAG_LATEST}' and '${TAG_VERSION}'..."
git tag "${TAG_LATEST}"
print_success "Created local tag: '${TAG_LATEST}'"
git tag "${TAG_VERSION}"
print_success "Created local tag: '${TAG_VERSION}'"

# Step 4: Push the new tags to the remote repository
print_step 4 4 "Pushing new tags to origin"
git push origin "${TAG_LATEST}" "${TAG_VERSION}"
print_success "Pushed new tags to remote."

echo -e "\n${GREEN}${ROCKET_ICON} All tasks completed successfully! ${ROCKET_ICON}${NC}"
echo -e "${GREEN}HEAD is now tagged as '${TAG_LATEST}' and '${TAG_VERSION}'.${NC}\n"
