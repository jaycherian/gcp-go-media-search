flowchart TD
 subgraph subGraph0["Flow 1: File Upload & Transcoding"]
    direction LR
        B("<b>FileUpload</b> in <i>cmd/server/main.go</i>")
        A["User uploads video via React UI"]
        C("<b>bucket.Object().NewWriter()</b><br>in <i>internal/cloud/state.go</i>")
        D["GCS Hi-Res Bucket"]
        E["Pub/Sub Hi-Res Topic"]
        F("<b>SetupListeners</b> for HiResTopic<br>in <i>cmd/server/listeners.go</i>")
        G("<b>MediaResizeWorkflow.Execute</b><br>in <i>internal/core/workflow/media_resize_workflow.go</i>")
        H("<b>GCSToTempFile.Execute</b><br>in <i>internal/core/commands/gcs_to_temp_file.go</i>")
        I{"Downloads from GCS Hi-Res"}
        J("<b>FFMpegCommand.Execute</b><br>in <i>internal/core/commands/ffmpeg.go</i>")
        K{"Transcodes video locally using FFmpeg"}
        L("<b>GCSFileUpload.Execute</b><br>in <i>internal/core/commands/gcs_file_upload.go</i>")
        M["GCS Low-Res Bucket"]
  end
 subgraph subGraph1["Flow 2: AI Analysis & Persistence"]
    direction LR
        N["Pub/Sub Low-Res Topic"]
        O("<b>SetupListeners</b> for LowResTopic<br>in <i>cmd/server/listeners.go</i>")
        P("<b>MediaReaderPipeline.Execute</b><br>in <i>internal/core/workflow/media_reader_workflow.go</i>")
        Q("<b>MediaUpload.Execute</b><br>in <i>internal/core/commands/media_upload.go</i>")
        R{"Uploads low-res video to Vertex AI File Service"}
        S("<b>MediaSummaryCreator.Execute</b><br>in <i>internal/core/commands/media_summary_creator.go</i>")
        T("<b>GenerateMultiModalResponse</b><br>in <i>internal/cloud/utils.go</i>")
        U["Vertex AI (Gemini)"]
        V("<b>MediaSummaryJsonToStruct.Execute</b><br>in <i>internal/core/commands/media_summary_json_to_struct.go</i>")
        W("<b>SceneExtractor.Execute</b><br>in <i>internal/core/commands/scene_extractor.go</i>")
        X{"<b>sceneWorker</b> calls <b>GenerateMultiModalResponse</b>"}
        Y["Vertex AI (Gemini)"]
        Z("<b>MediaAssembly.Execute</b><br>in <i>internal/core/commands/media_assembly.go</i>")
        AA("<b>MediaPersistToBigQuery.Execute</b><br>in <i>internal/core/commands/media_persist_to_big_query.go</i>")
        BB@{ label: "BigQuery 'media' Table" }
        CC("<b>MediaEmbeddingGeneratorWorkflow.Execute</b><br>in <i>internal/core/workflow/media_embedding_generator_workflow.go</i>")
        DD["Vertex AI (Gemini)"]
        EE@{ label: "Saves to BigQuery 'scene_embeddings' Table" }
  end
 subgraph subGraph2["Flow 3: Search & Retrieval"]
    direction LR
        GG("<b>MediaRouter</b> GET /api/v1/media<br>in <i>cmd/server/main.go</i>")
        FF["User searches via React UI"]
        HH("<b>SearchService.FindScenes</b><br>in <i>internal/core/services/search.go</i>")
        II["Vertex AI (Gemini)"]
        JJ@{ label: "BigQuery 'scene_embeddings' Table" }
        KK("<b>MediaService.Get</b> &amp; <b>GetScene</b><br>in <i>internal/core/services/media.go</i>")
        LL@{ label: "BigQuery 'media' Table" }
        MM("<b>MediaService.GenerateSignedURL</b><br>in <i>internal/core/services/media.go</i>")
        NN["GCS Low-Res Bucket"]
  end
    A --> B
    B --> C
    C --> D
    D -- GCS Notification --> E
    E -- Delivers Message --> F
    F --> G
    G --> H
    H --> I
    I --> J
    J --> K
    K --> L
    L --> M
    M -- GCS Notification --> N
    N -- Delivers Message --> O
    O --> P
    P --> Q
    Q --> R
    R --> S
    S -- Calls --> T
    T --> U & V
    U -- Returns Summary JSON --> T
    V --> W
    W -- In Parallel --> X
    X --> Y & W
    Y -- Returns Scene JSON --> X
    W --> Z
    Z --> AA
    AA --> BB
    BB -- Periodically Polled by --> CC
    CC -- Embeds each scene's script --> DD
    DD -- Returns Embedding Vector --> CC
    CC --> EE
    FF --> GG
    GG --> HH
    HH -- Gets query embedding --> II
    II --> HH
    HH -- VECTOR_SEARCH --> JJ
    JJ -- Returns matching scene IDs --> HH
    HH --> GG
    GG -- Gets full media details --> KK
    KK --> LL & MM
    LL --> KK
    MM -- Generates secure URL --> NN
    NN --> MM
    MM --> GG
    GG -- Returns full results with signed URL --> FF
    subGraph1 --> n1["Untitled Node"]

    BB@{ shape: rect}
    EE@{ shape: diamond}
    JJ@{ shape: rect}
    LL@{ shape: rect}


