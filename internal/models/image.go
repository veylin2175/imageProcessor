package models

import (
	"github.com/google/uuid"
	"time"
)

type Image struct {
	ID                     uuid.UUID `db:"id" json:"ID"`
	Filename               string    `db:"filename" json:"Filename"`
	Status                 string    `db:"status" json:"Status"`
	OriginalPath           string    `db:"original_path" json:"OriginalPath"`
	ProcessedPathResize    *string   `db:"processed_path_resize" json:"ProcessedPathResize"`       // <-- Изменили
	ProcessedPathThumbnail *string   `db:"processed_path_thumbnail" json:"ProcessedPathThumbnail"` // <-- Изменили
	ProcessedPathWatermark *string   `db:"processed_path_watermark" json:"ProcessedPathWatermark"` // <-- Изменили
	CreatedAt              time.Time `db:"created_at" json:"CreatedAt"`
	UpdatedAt              time.Time `db:"updated_at" json:"UpdatedAt"`
}
