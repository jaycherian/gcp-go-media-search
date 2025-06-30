```mermaid
sequenceDiagram
    actor User
    participant Go API Server
    participant GCS Hi-Res Bucket
    participant Pub/Sub Hi-Res Topic
    participant MediaResizeWorkflow
    participant GCS Low-Res Bucket
    participant Pub/Sub Low-Res Topic
    participant MediaReaderPipeline
    participant Vertex AI (Gemini)
    participant BigQuery

    %% =================================================================
    %% Flow 1: Video Ingestion & Resizing
    %% =================================================================

    alt Video Ingestion and Resizing

        User->>+Go API Server: POST /api/v1/uploads (upload high-res video)
        Go API Server->>+GCS Hi-Res Bucket: Uploads video file
        GCS Hi-Res Bucket-->>-Go API Server: Upload success
        Go API Server-->>-User: HTTP 200 OK

    end

    %% =================================================================
    %% Flow 2: AI Analysis & Persistence
    %% =================================================================
    
    alt AI Analysis and Persistence

        GCS Low-Res Bucket->>+Pub/Sub Low-Res Topic: Publishes notification (Object Finalized)
        Pub/Sub Low-Res Topic-->>+MediaReaderPipeline: Delivers message
        Note over MediaReaderPipeline: (Listens on LowResTopic subscription)

        MediaReaderPipeline->>+GCS Low-Res Bucket: Downloads low-res video
        GCS Low-Res Bucket-->>-MediaReaderPipeline: Returns video file

        MediaReaderPipeline->>+Vertex AI (Gemini): Uploads file to File Service for processing
        Vertex AI (Gemini)-->>-MediaReaderPipeline: Returns file handle

        MediaReaderPipeline->>+Vertex AI (Gemini): Generates summary (passes video handle & summary prompt)
        Vertex AI (Gemini)-->>-MediaReaderPipeline: Returns structured JSON with title, summary, cast, and scene timestamps

        Note over MediaReaderPipeline: Spawns parallel workers for scene extraction

        loop For Each Scene Timestamp
            MediaReaderPipeline->>+Vertex AI (Gemini): Extracts scene script (passes video handle, timestamp & scene prompt)
            Vertex AI (Gemini)-->>-MediaReaderPipeline: Returns scene script
        end

        MediaReaderPipeline->>MediaReaderPipeline: Assembles complete Media object (summary + all scenes)

        MediaReaderPipeline->>+BigQuery: Inserts complete Media object into 'media' table
        BigQuery-->>-MediaReaderPipeline: Insert success
        
        deactivate MediaReaderPipeline
    
    end

    %% =================================================================
    %% Flow 3: Semantic Search
    %% =================================================================

    alt Semantic Search

        User->>+Go API Server: GET /api/v1/media?s=... (search query)
        
        Go API Server->>+Vertex AI (Gemini): Generates embedding for search query text
        Vertex AI (Gemini)-->>-Go API Server: Returns query vector

        Note over Go API Server: Constructs VECTOR_SEARCH query
        Go API Server->>+BigQuery: Executes VECTOR_SEARCH on 'scene_embeddings' table
        BigQuery-->>-Go API Server: Returns list of matching media_id and scene_sequence

        loop For Each Match
            Go API Server->>+BigQuery: Fetches full Media & Scene details from 'media' table
            BigQuery-->>-Go API Server: Returns Media/Scene objects
        end

        Go API Server->>+GCS Low-Res Bucket: Generates a Signed URL for video streaming
        GCS Low-Res Bucket-->>-Go API Server: Returns secure, time-limited URL

        Go API Server-->>-User: HTTP 200 OK (with structured results and signed URL)

    end

