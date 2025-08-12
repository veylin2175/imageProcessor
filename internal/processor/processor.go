package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"image"
	"imageProcessor/internal/lib/logger/sl"
	"imageProcessor/internal/storage/postgres"
	"log/slog"
	"os"
	"path/filepath"
)

type ImageProcessor struct {
	storage *postgres.Storage
	log     *slog.Logger
}

func NewImageProcessor(log *slog.Logger, storage *postgres.Storage) *ImageProcessor {
	return &ImageProcessor{
		log:     log,
		storage: storage,
	}
}

func (p *ImageProcessor) ProcessMessage(ctx context.Context, message []byte) error {
	const op = "processor.ProcessMessage"

	var kafkaMessage struct {
		ImageID      uuid.UUID `json:"image_id"`
		OriginalPath string    `json:"original_path"`
	}

	if err := json.Unmarshal(message, &kafkaMessage); err != nil {
		p.log.Error("failed to unmarshal kafka message", slog.String("op", op), slog.String("error", err.Error()))
		return err
	}

	p.log.Info("processing image", slog.String("op", op), slog.String("image_id", kafkaMessage.ImageID.String()))

	src, err := imaging.Open(kafkaMessage.OriginalPath)
	if err != nil {
		p.log.Error("failed to open image", slog.String("op", op), slog.String("path", kafkaMessage.OriginalPath), slog.String("error", err.Error()))
		return err
	}

	processedPaths := make(map[string]string)
	outputDir := "./processed"
	if _, err = os.Stat(outputDir); os.IsNotExist(err) {
		err = os.Mkdir(outputDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	resizedImage := imaging.Resize(src, 800, 0, imaging.Lanczos)
	resizedPath := filepath.Join(outputDir, fmt.Sprintf("%s_resized.jpg", kafkaMessage.ImageID))
	err = imaging.Save(resizedImage, resizedPath)
	if err != nil {
		p.log.Error("failed to save resized image", slog.String("op", op), sl.Err(err))
		return err
	}
	processedPaths["resize"] = resizedPath

	thumbnailImage := imaging.Thumbnail(src, 150, 150, imaging.CatmullRom)
	thumbnailPath := filepath.Join(outputDir, fmt.Sprintf("%s_thumbnail.jpg", kafkaMessage.ImageID))
	err = imaging.Save(thumbnailImage, thumbnailPath)
	if err != nil {
		p.log.Error("failed to save thumbnail image", slog.String("op", op), sl.Err(err))
		return err
	}
	processedPaths["thumbnail"] = thumbnailPath

	watermark, err := imaging.Open("watermark.png")
	if err == nil {
		bounds := src.Bounds()
		watermarkBounds := watermark.Bounds()

		x := bounds.Dx()/2 - watermarkBounds.Dx()/2
		y := bounds.Dy()/2 - watermarkBounds.Dy()/2

		watermarkedImage := imaging.Overlay(src, watermark, image.Pt(x, y), 1.0)
		watermarkedPath := filepath.Join(outputDir, fmt.Sprintf("%s_watermarked.jpg", kafkaMessage.ImageID))
		err = imaging.Save(watermarkedImage, watermarkedPath)
		if err != nil {
			p.log.Error("failed to save watermarked image", slog.String("op", op), sl.Err(err))
			return err
		}
		processedPaths["watermark"] = watermarkedPath
	} else {
		p.log.Warn("watermark file not found, skipping watermark processing", slog.String("op", op), sl.Err(err))
	}

	err = p.storage.UpdateImageStatus(ctx, kafkaMessage.ImageID, "processed", processedPaths)
	if err != nil {
		p.log.Error("failed to update image status in storage", slog.String("op", op), slog.String("image_id", kafkaMessage.ImageID.String()), slog.String("error", err.Error()))
		return err
	}

	p.log.Info("image processed successfully and status updated", slog.String("op", op), slog.String("image_id", kafkaMessage.ImageID.String()))

	return nil
}
