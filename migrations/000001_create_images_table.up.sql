CREATE TABLE IF NOT EXISTS images
(
    id                       UUID PRIMARY KEY,
    filename                 TEXT        NOT NULL,
    status                   VARCHAR(50) NOT NULL     DEFAULT 'pending',
    original_path            TEXT        NOT NULL,
    processed_path_resize    TEXT,
    processed_path_thumbnail TEXT,
    processed_path_watermark TEXT,
    created_at               TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);