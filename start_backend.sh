#!/bin/bash
# ==============================================================================
# Start Backend Script
#
# Description:
#   This script starts only the backend Go server for local development or API
#   testing without launching the frontend UI. It logs output to both the
#   console and to a file named 'backend.log' in the project root.
#   It also includes graceful shutdown handling.
#
# USAGE: Run this script from the root of the project.
# $ ./start_backend.sh
# ==============================================================================

# Exit the script if any command fails
set -e

# Define a cleanup function to be called on script exit.
# This function will be triggered by the 'trap' command below.
cleanup() {
    echo # Newline for cleaner output
    echo "Shutting down backend server (PID: $BACKEND_PID)..."
    # The 'kill' command sends a SIGTERM signal by default.
    # This allows the Go server's graceful shutdown logic to run.
    # The '|| true' ensures that if the process is already gone, the script doesn't error out.
    kill $BACKEND_PID || true
    echo "Backend server shut down."
}

# 'trap' catches signals. When this script receives an EXIT signal (e.g., from
# Ctrl+C or when it finishes normally), it will execute the 'cleanup' function.
trap cleanup EXIT

echo "Changing directory to backend/go..."
cd backend/go

echo "Starting Go backend server in the background..."
echo "The API server will be available at http://localhost:8080"
echo "Output will be displayed here and saved to backend.log in the project root."

# Run the Go server in the background (&).
# The `2>&1` redirects stderr to stdout.
# The `tee` command reads from stdin and writes to both stdout and the specified file.
go run ./server 2>&1 | tee ../../backend.log &

# Get the Process ID (PID) of the last background command.
BACKEND_PID=$!
echo "Backend server started with PID: $BACKEND_PID"
echo "Press Ctrl+C to stop the server gracefully."

# The 'wait' command pauses the script here and waits for the specified PID
# to finish. This keeps the script alive while the server is running.
wait $BACKEND_PID
