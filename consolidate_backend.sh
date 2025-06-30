#!/bin/bash

# ==============================================================================
# Backend Consolidation Script
#
# Description:
#   This script further refines the backend structure by moving the server
#   package out of the `/cmd` directory.
#
# WARNING:
#   This script makes file system changes. Run it on a clean git branch.
#   It assumes the 'reorganize_project.sh' script has already been run.
#
# USAGE:
#   Run this script from the root of the project.
#   $ ./consolidate_backend.sh
# ==============================================================================

# Exit immediately if a command exits with a non-zero status.
set -e

echo "Starting backend consolidation..."
echo "------------------------------------------------"

# --- Step 1: Move the server package ---
echo "[1/2] Moving 'server' package up one level..."
mv backend/go/cmd/server backend/go/server
echo "    -> Done."

# --- Step 2: Clean up the old cmd directory ---
echo "[2/2] Removing empty 'cmd' directory..."
rm -rf backend/go/cmd
echo "    -> Done."


echo "------------------------------------------------"
echo "Backend consolidation complete!"
echo "IMPORTANT: Key files have been updated to reflect these changes."

