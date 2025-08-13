package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"imageProcessor/internal/config"
	"imageProcessor/internal/models"

	_ "github.com/lib/pq"
)

type Storage struct {
	DB *sql.DB
}

func InitDB(dbCfg *config.Database) (*Storage, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.User,
		dbCfg.Password,
		dbCfg.DBName,
		dbCfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	return &Storage{DB: db}, nil
}

func (s *Storage) SaveImage(ctx context.Context, filename string, originalPath string) (*models.Image, error) {
	const op = "storage.postgres.SaveImage"

	imageID := uuid.New()

	query := `
        INSERT INTO images (id, filename, status, original_path)
        VALUES ($1, $2, $3, $4)
        RETURNING id, filename, status, original_path, created_at, updated_at`

	var image models.Image

	err := s.DB.QueryRowContext(ctx, query, imageID, filename, "pending", originalPath).Scan(
		&image.ID,
		&image.Filename,
		&image.Status,
		&image.OriginalPath,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &image, nil
}

func (s *Storage) GetImage(ctx context.Context, id uuid.UUID) (*models.Image, error) {
	const op = "storage.postgres.GetImage"

	query := `
        SELECT id, filename, status, original_path, processed_path_resize, processed_path_thumbnail, processed_path_watermark, created_at, updated_at
        FROM images
        WHERE id = $1`

	image := &models.Image{}

	err := s.DB.QueryRowContext(ctx, query, id).Scan(
		&image.ID,
		&image.Filename,
		&image.Status,
		&image.OriginalPath,
		&image.ProcessedPathResize,
		&image.ProcessedPathThumbnail,
		&image.ProcessedPathWatermark,
		&image.CreatedAt,
		&image.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: image with ID %s not found: %w", op, id, sql.ErrNoRows)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return image, nil
}

func (s *Storage) UpdateImageStatus(ctx context.Context, id uuid.UUID, status string, processedPaths map[string]string) error {
	const op = "storage.postgres.UpdateImageStatus"

	query := `
        UPDATE images
        SET status = $1, processed_path_resize = $2, processed_path_thumbnail = $3, processed_path_watermark = $4, updated_at = NOW()
        WHERE id = $5`

	resizePath := sql.NullString{String: processedPaths["resize"], Valid: processedPaths["resize"] != ""}
	thumbnailPath := sql.NullString{String: processedPaths["thumbnail"], Valid: processedPaths["thumbnail"] != ""}
	watermarkPath := sql.NullString{String: processedPaths["watermark"], Valid: processedPaths["watermark"] != ""}

	_, err := s.DB.ExecContext(ctx, query, status, resizePath, thumbnailPath, watermarkPath, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeleteImage(ctx context.Context, id uuid.UUID) error {
	const op = "storage.postgres.DeleteImage"

	query := `
        DELETE FROM images
        WHERE id = $1`

	result, err := s.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: image with ID %s not found", op, id)
	}

	return nil
}

func (s *Storage) Close() error {
	return s.DB.Close()
}
