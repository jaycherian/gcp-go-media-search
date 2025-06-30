#!/bin/bash

# ==============================================================================
# Project Reorganization Script
#
# Description:
#   This script reorganizes the project into a three-part structure:
#   1. deploy/   (Terraform)
#   2. backend/  (Go)
#   3. frontend/ (React UI)
#
# WARNING:
#   This script makes significant changes to the file system.
#   It is highly recommended to run this on a clean git branch.
#
# USAGE:
#   Run this script from the root of the project.
#   $ ./reorganize_project.sh
# ==============================================================================

# Exit immediately if a command exits with a non-zero status.
set -e

echo "Starting project reorganization..."
echo "------------------------------------------------"

# --- Step 1: Create new top-level directories ---
echo "[1/4] Creating new directory structure (deploy, backend, frontend)..."
mkdir -p deploy
mkdir -p backend/go
mkdir -p frontend/web
echo "    -> Done."

# --- Step 2: Move Terraform files ---
echo "[2/4] Moving Terraform files to deploy/..."
# Use rsync to move contents of the terraform directory
rsync -av --progress deployments/terraform/ deploy/ --remove-source-files
echo "    -> Done."

# --- Step 3: Move Backend Go files and module definitions ---
echo "[3/4] Moving Go source and module files to backend/go/..."
mv cmd backend/go/
mv internal backend/go/
mv go.mod backend/go/
mv go.sum backend/go/
echo "    -> Done."

# --- Step 4: Move Frontend React files ---
echo "[4/4] Moving React UI files to frontend/web/ui/..."
mv web/ui frontend/web/
echo "    -> Done."

# --- Step 5: Clean up old, now-empty directories ---
echo "[5/5] Cleaning up old directories..."
rm -rf deployments
rm -rf web
echo "    -> Done."


echo "------------------------------------------------"
echo "Reorganization complete!"
echo
echo "IMPORTANT: Several files now require content updates."
echo "Please see the 'File Content Updates' document for the exact changes needed."


