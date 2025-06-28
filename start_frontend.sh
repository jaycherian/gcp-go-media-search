#!/bin/bash
# ==============================================================================
# Start Script for Media Search Application
# ==============================================================================
# This script starts both the backend Go server and the frontend Vite server
# for local development.
#
# USAGE: Run this script from the root of the project.
# $ ./start.sh
# ==============================================================================

# Exit the script if any command fails
set -e



# 'trap' catches signals. When this script receives an EXIT signal (e.g., from
# Ctrl+C or when it finishes), it will run the 'cleanup' function.
trap cleanup EXIT

# Navigate to the UI directory to run the frontend commands.
cd web/ui

echo "Installing frontend dependencies (if needed)..."
pnpm install

echo "Starting Vite frontend server in the foreground..."
echo "Press Ctrl+C to stop both servers."

# Start the Vite dev server in the foreground. The script will pause here
# until this command is terminated by the user.
pnpm dev

# When `pnpm dev` is stopped (Ctrl+C), the script will exit,
# triggering the `trap` and running the `cleanup` function.

