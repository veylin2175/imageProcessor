package models

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type Image struct {
	ID                     uuid.UUID      `db:"id"`
	Filename               string         `db:"filename"`
	Status                 string         `db:"status"`
	OriginalPath           string         `db:"original_path"`
	ProcessedPathResize    sql.NullString `db:"processed_path_resize"`
	ProcessedPathThumbnail sql.NullString `db:"processed_path_thumbnail"`
	ProcessedPathWatermark sql.NullString `db:"processed_path_watermark"`
	CreatedAt              time.Time      `db:"created_at"`
	UpdatedAt              time.Time      `db:"updated_at"`
}
