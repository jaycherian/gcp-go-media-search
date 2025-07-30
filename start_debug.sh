#!/bin/bash
# ==============================================================================
# Start Script for Media Search Application
# ==============================================================================
# This script starts both the backend Go server and the frontend Vite server
# for local development.
#
# USAGE: Run this script from the root of the project.
# $ ./start_servers.sh
# ==============================================================================

# Exit the script if any command fails
set -e

# Define PIDs for cleanup
BACKEND_PID=""
FRONTEND_PID=""

# Define a cleanup function to be called on script exit.
cleanup() {
    echo
    echo "Shutting down servers..."
    if [ -n "$BACKEND_PID" ]; then
        echo "Shutting down backend server (PID: $BACKEND_PID)..."
        # The 'kill' command sends a SIGTERM signal by default, allowing graceful shutdown.
        kill $BACKEND_PID || true
    fi
    if [ -n "$FRONTEND_PID" ]; then
        echo "Shutting down frontend server (PID: $FRONTEND_PID)..."
        kill $FRONTEND_PID || true
    fi
    echo "All servers shut down."
}

# 'trap' catches signals. When this script receives an EXIT signal (e.g., from
# Ctrl+C or when it finishes), it will run the 'cleanup' function.
trap cleanup EXIT

# --- Start Backend ---
echo "Starting Go backend server in the background..."
echo "Backend Server Environment:"
echo "GOOGLE_APPLICATION_CREDENTIALS: ${GOOGLE_APPLICATION_CREDENTIALS}"
echo "GOOGLE_GENAI_USE_VERTEXAI: ${GOOGLE_GENAI_USE_VERTEXAI}"
echo "GOOGLE_CLOUD_PROJECT: ${GOOGLE_CLOUD_PROJECT}"
echo "GOOGLE_CLOUD_LOCATION: ${GOOGLE_CLOUD_LOCATION}"
echo "Backend logs will be written to backend.log"
(cd backend/go && dlv debug ./server --headless --listen=:2345 --api-version=2 --accept-multiclient) > /var/log/media-search-backend.log 2>&1 &
BACKEND_PID=$!
echo "Backend server started with PID: $BACKEND_PID"

# --- Start Frontend ---
echo "Installing frontend dependencies (if needed)..."
(cd frontend/web/ui && pnpm install)

echo "Starting Vite frontend server in the background..."
echo "Frontend logs will be written to frontend.log"
(cd frontend/web/ui && pnpm dev -- --host) > /var/log/media-search-frontend.log 2>&1 &
FRONTEND_PID=$!
echo "Frontend server started with PID: $FRONTEND_PID"

# --- Wait for servers ---
echo
echo "Both servers are running in the background."
echo "Backend API: http://localhost:8080"
echo "Frontend UI will be available shortly (check frontend.log for URL)."
echo "Press Ctrl+C to stop both servers."

# The 'wait' command pauses the script here and waits for all background
# child processes to finish. When Ctrl+C is pressed, the 'trap' runs,
# kills the children, and 'wait' then exits.
wait
