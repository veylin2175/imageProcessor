package saveImage

import (
	"context"
	"encoding/json"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"imageProcessor/internal/kafka/producer"
	"imageProcessor/internal/lib/api/response"
	"imageProcessor/internal/lib/logger/sl"
	"imageProcessor/internal/models"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type ImageResponse struct {
	response.Response
	ImageID uuid.UUID `json:"image_id"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=ImageSaver
type ImageSaver interface {
	SaveImage(ctx context.Context, filename string, originalPath string) (*models.Image, error)
}

// SaveImage uploads an image for processing.
// @Summary      Uploads an image
// @Description  Uploads an image file and returns its ID
// @Tags         images
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image file to upload"
// @Success      200  {object}  saveImage.ImageResponse
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /upload [post]
func New(log *slog.Logger, imageSaver ImageSaver, kafkaProducer producer.ProducerIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.image.saveImage.New"

		log = log.With(
			slog.String("op", op),
		)

		file, header, err := r.FormFile("image")
		if err != nil {
			log.Error("failed to get file from request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("failed to get file from request"))
			return
		}
		defer func(file multipart.File) {
			err = file.Close()
			if err != nil {
				return
			}
		}(file)

		if header.Size == 0 {
			log.Error("received empty file")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("received empty file"))
			return
		}

		uploadPath := "./uploads"
		if _, err = os.Stat(uploadPath); os.IsNotExist(err) {
			err = os.Mkdir(uploadPath, os.ModePerm)
			if err != nil {
				return
			}
		}

		filePath := filepath.Join(uploadPath, header.Filename)
		dst, err := os.Create(filePath)
		if err != nil {
			log.Error("failed to create file on disk", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to save file"))
			return
		}
		defer func(dst *os.File) {
			err = dst.Close()
			if err != nil {
				return
			}
		}(dst)

		if _, err = io.Copy(dst, file); err != nil {
			log.Error("failed to copy file content", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to save file"))
			return
		}

		image, err := imageSaver.SaveImage(r.Context(), header.Filename, filePath)
		if err != nil {
			log.Error("failed to save image metadata", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to save image metadata"))
			return
		}

		log.Info("image saved successfully", slog.String("image_id", image.ID.String()))

		kafkaMessage := struct {
			ImageID      uuid.UUID `json:"image_id"`
			OriginalPath string    `json:"original_path"`
		}{
			ImageID:      image.ID,
			OriginalPath: image.OriginalPath,
		}

		message, err := json.Marshal(kafkaMessage)
		if err != nil {
			log.Error("failed to marshal kafka message", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to prepare message"))
			return
		}

		err = kafkaProducer.SendMessage(r.Context(), message)
		if err != nil {
			log.Error("failed to publish message to kafka", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to start image processing"))
			return
		}

		log.Info("image saved successfully and message published to kafka", slog.String("image_id", image.ID.String()))

		render.JSON(w, r, ImageResponse{
			Response: response.OK(),
			ImageID:  image.ID,
		})
	}
}
