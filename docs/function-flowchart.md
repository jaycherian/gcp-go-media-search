```mermaid
graph TD
    subgraph "Flow 1: File Upload & Transcoding"
        direction LR
        A["User uploads video via React UI"] --> B("<b>FileUpload</b> in <i>backend/go/server/main.go</i>");
        B --> C("<b>bucket.Object().NewWriter()</b><br>in <i>backend/go/internal/cloud/state.go</i>");
        C --> D["GCS Hi-Res Bucket"];
        D -- "GCS Notification" --> E["Pub/Sub Hi-Res Topic"];
        E -- "Delivers Message" --> F("<b>SetupListeners</b> for HiResTopic<br>in <i>backend/go/server/listeners.go</i>");
        F --> G("<b>MediaResizeWorkflow.Execute</b><br>in <i>backend/go/internal/core/workflow/media_resize_workflow.go</i>");
        G --> H("<b>GCSToTempFile.Execute</b><br>in <i>backend/go/internal/core/commands/gcs_to_temp_file.go</i>");
        H --> I{"Downloads from GCS Hi-Res"};
        I --> J("<b>FFMpegCommand.Execute</b><br>in <i>backend/go/internal/core/commands/ffmpeg.go</i>");
        J --> K{"Transcodes video locally using FFmpeg"};
        K --> L("<b>GCSFileUpload.Execute</b><br>in <i>backend/go/internal/core/commands/gcs_file_upload.go</i>");
        L --> M["GCS Low-Res Bucket"];
    end

    subgraph "Flow 2: AI Analysis & Persistence"
        direction LR
        M -- "GCS Notification" --> N["Pub/Sub Low-Res Topic"];
        N -- "Delivers Message" --> O("<b>SetupListeners</b> for LowResTopic<br>in <i>backend/go/server/listeners.go</i>");
        O --> P("<b>MediaReaderPipeline.Execute</b><br>in <i>backend/go/internal/core/workflow/media_reader_workflow.go</i>");
        P --> Q("<b>MediaUpload.Execute</b><br>in <i>backend/go/internal/core/commands/media_upload.go</i>");
        Q --> R{"Uploads low-res video to Vertex AI File Service"};
        R --> S("<b>MediaSummaryCreator.Execute</b><br>in <i>backend/go/internal/core/commands/media_summary_creator.go</i>");
        S -- "Calls" --> T("<b>GenerateMultiModalResponse</b><br>in <i>backend/go/internal/cloud/utils.go</i>");
        T --> U["Vertex AI (Gemini)"];
        U -- "Returns Summary JSON" --> T;
        T --> V("<b>MediaSummaryJsonToStruct.Execute</b><br>in <i>backend/go/internal/core/commands/media_summary_json_to_struct.go</i>");
        V --> W("<b>SceneExtractor.Execute</b><br>in <i>backend/go/internal/core/commands/scene_extractor.go</i>");
        W -- "In Parallel" --> X{"<b>sceneWorker</b> calls <b>GenerateMultiModalResponse</b>"};
        X --> Y["Vertex AI (Gemini)"];
        Y -- "Returns Scene JSON" --> X;
        X --> W;
        W --> Z("<b>MediaAssembly.Execute</b><br>in <i>backend/go/internal/core/commands/media_assembly.go</i>");
        Z --> AA("<b>MediaPersistToBigQuery.Execute</b><br>in <i>backend/go/internal/core/commands/media_persist_to_big_query.go</i>");
        AA --> BB["BigQuery 'media' Table"];
        BB -- "Periodically Polled by" --> CC("<b>MediaEmbeddingGeneratorWorkflow.Execute</b><br>in <i>backend/go/internal/core/workflow/media_embedding_generator_workflow.go</i>");
        CC -- "Embeds each scene's script" --> DD["Vertex AI (Gemini)"];
        DD -- "Returns Embedding Vector" --> CC;
        CC --> EE{"Saves to BigQuery 'scene_embeddings' Table"};
    end

    subgraph "Flow 3: Search & Retrieval"
        direction LR
        FF["User searches via React UI"] --> GG("<b>MediaRouter</b> GET /api/v1/media<br>in <i>backend/go/server/main.go</i>");
        GG --> HH("<b>SearchService.FindScenes</b><br>in <i>backend/go/internal/core/services/search.go</i>");
        HH -- "Gets query embedding" --> II["Vertex AI (Gemini)"];
        II --> HH;
        HH -- "VECTOR_SEARCH" --> JJ["BigQuery 'scene_embeddings' Table"];
        JJ -- "Returns matching scene IDs" --> HH;
        HH --> GG;
        GG -- "Gets full media details" --> KK("<b>MediaService.Get</b> & <b>GetScene</b><br>in <i>backend/go/internal/core/services/media.go</i>");
        KK --> LL["BigQuery 'media' Table"];
        LL --> KK;
        KK --> MM("<b>MediaService.GenerateSignedURL</b><br>in <i>backend/go/internal/core/services/media.go</i>");
        MM -- "Generates secure URL" --> NN["GCS Low-Res Bucket"];
        NN --> MM;
        MM --> GG;
        GG -- "Returns full results with signed URL" --> FF;
    end
