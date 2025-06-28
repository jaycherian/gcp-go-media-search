#!/bin/bash

# ==============================================================================
# Consolidated Repository Reorganization Script
# ==============================================================================
# This script reorganizes the gcp-go-media-search repository from a Bazel-based
# structure to a more conventional Go and Node.js project structure. It handles
# both file movement and Go import path updates.
#
# WARNING: This script makes significant changes to the file system.
# It is highly recommended to run this on a clean git branch.
#
# USAGE: Run this script from the root of the 'gcp-go-media-search' repository.
#
# $ ./reorganize_consolidated.sh
# ==============================================================================

# Exit immediately if a command exits with a non-zero status.
set -e

echo "Starting repository reorganization..."
echo "Please ensure you are running this from the project root on a clean git branch."
echo "--------------------------------------------------------------------"

# === STEP 1: Create the new directory structure ===
echo "[1/8] Creating new directory structure..."
mkdir -p cmd/server
mkdir -p internal/api
mkdir -p internal/cloud
mkdir -p internal/core/commands
mkdir -p internal/core/cor
mkdir -p internal/core/model
mkdir -p internal/core/services
mkdir -p internal/core/workflow
mkdir -p internal/telemetry
mkdir -p internal/testutil
mkdir -p deployments
mkdir -p web/ui
echo "    -> Done."

# === STEP 2: Move the Go backend source files ===
echo "[2/8] Moving Go backend and library files..."
# Move main package and split the original api_server package
mv web/apps/api_server/api_server.go cmd/server/main.go
mv web/apps/api_server/setup.go cmd/server/
mv web/apps/api_server/listeners.go cmd/server/
mv web/apps/api_server/media.go internal/api/
mv web/apps/api_server/file_upload.go internal/api/
mv web/apps/api_server/dashboard.go internal/api/

# Move the 'pkg' contents into the new 'internal' structure
mv pkg/cloud/* internal/cloud/
mv pkg/cor/* internal/core/cor/
mv pkg/model/* internal/core/model/
mv pkg/telemetry/* internal/telemetry/

# Move commands, services, and workflow to their own packages to avoid collisions
mv pkg/commands/* internal/core/commands/
mv pkg/services/* internal/core/services/
mv pkg/workflow/* internal/core/workflow/
echo "    -> Done."


# === STEP 3: Move Terraform files ===
echo "[3/8] Moving Terraform deployment files..."
mv build/terraform deployments/
echo "    -> Done."

# === STEP 4: Move test files to match the new structure ===
echo "[4/8] Reorganizing test files..."
mkdir -p internal/cloud/test
mkdir -p internal/core/model/test
mkdir -p internal/core/services/test
mkdir -p internal/core/workflow/test

mv test/cloud/* internal/cloud/test/
mv test/model/* internal/core/model/test/
mv test/services/* internal/core/services/test/
mv test/workflow/* internal/core/workflow/test/
mv test/test.go internal/testutil/
echo "    -> Done."


# === STEP 5: Move the React frontend ===
echo "[5/8] Moving React frontend files..."
# Use rsync to robustly move files, including dotfiles
rsync -av --progress web/apps/media-search/ web/ui/ --remove-source-files
echo "    -> Done."


# === STEP 6: Clean up old, now-empty directories ===
echo "[6/8] Cleaning up old directories..."
rm -rf pkg
rm -rf web/apps
rm -rf build
rm -rf test
echo "    -> Done."

# === STEP 7: Delete all Bazel-related files ===
echo "[7/8] Deleting Bazel configuration files..."
find . -name "BUILD.bazel" -type f -delete
find . -name "*.bzl" -type f -delete
find . -name "MODULE.bazel" -type f -delete
find . -name "WORKSPACE" -type f -delete
find . -name ".bazelignore" -type f -delete
find . -name ".bazelrc" -type f -delete
echo "    -> Done."

# === STEP 8: Update Go import paths ===
echo "[8/8] Updating Go module and import paths..."
OLD_MODULE="github.com/GoogleCloudPlatform/solutions/media"
NEW_MODULE="github.com/jaycherian/gcp-go-media-search"

# Update go.mod file
sed -i.bak "s|${OLD_MODULE}|${NEW_MODULE}|g" go.mod

# Find all .go files in the new structure
GO_FILES=$(find cmd internal -name "*.go" -type f)

# Loop through all found .go files and replace old import paths.
for file in $GO_FILES; do
    sed -i.bak "s|${OLD_MODULE}/pkg/cloud|${NEW_MODULE}/internal/cloud|g" "$file"
    sed -i.bak "s|${OLD_MODULE}/pkg/cor|${NEW_MODULE}/internal/core/cor|g" "$file"
    sed -i.bak "s|${OLD_MODULE}/pkg/model|${NEW_MODULE}/internal/core/model|g" "$file"
    sed -i.bak "s|${OLD_MODULE}/pkg/telemetry|${NEW_MODULE}/internal/telemetry|g" "$file"
    sed -i.bak "s|${OLD_MODULE}/pkg/workflow|${NEW_MODULE}/internal/core/workflow|g" "$file"
    sed -i.bak "s|${OLD_MODULE}/pkg/commands|${NEW_MODULE}/internal/core/commands|g" "$file"
    sed -i.bak "s|${OLD_MODULE}/pkg/services|${NEW_MODULE}/internal/core/services|g" "$file"
    sed -i.bak "s|${OLD_MODULE}/test|${NEW_MODULE}/internal/testutil|g" "$file"
done

# Clean up backup files created by sed
find . -name "*.bak" -type f -delete
echo "    -> Done."

echo "--------------------------------------------------------------------"
echo "Reorganization complete!"
echo
echo "--- NEXT STEPS ---"
echo
echo "1. REVIEW GO CHANGES & DEPENDENCIES:"
echo "   - Run 'go mod tidy' to sync dependencies."
echo
echo "2. UPDATE FRONTEND SCRIPTS:"
echo "   - Edit 'web/ui/package.json' to use standard Vite scripts for 'dev', 'build', and 'preview'."
echo "   - Navigate to 'web/ui' and run 'pnpm install'."
echo
echo "3. UPDATE DOCUMENTATION:"
echo "   - Update README.md and other docs with the new, simplified build instructions."
echo
echo "4. REVIEW AND COMMIT:"
echo "   - Carefully review all changes with 'git status' and 'git diff'."
echo "   - Commit the changes to your git branch."


