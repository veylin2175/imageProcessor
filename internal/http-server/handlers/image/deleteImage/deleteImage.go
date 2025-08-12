package deleteImage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"imageProcessor/internal/lib/api/response"
	"imageProcessor/internal/lib/logger/sl"
	"log/slog"
	"net/http"
)

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=ImageDeleter
type ImageDeleter interface {
	DeleteImage(ctx context.Context, id uuid.UUID) error
}

type Response struct {
	response.Response
}

func New(log *slog.Logger, imageDeleter ImageDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.image.deleteImage.New"

		log = log.With(slog.String("op", op))

		idStr := chi.URLParam(r, "id")
		imageID, err := uuid.Parse(idStr)
		if err != nil {
			log.Error("failed to parse image ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid image ID"))
			return
		}

		log.Info("attempting to delete image", slog.String("image_id", imageID.String()))

		err = imageDeleter.DeleteImage(r.Context(), imageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Warn("image not found for deletion", slog.String("image_id", imageID.String()))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("image not found"))
				return
			}

			log.Error("failed to delete image from storage", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to delete image"))
			return
		}

		log.Info("image deleted successfully", slog.String("image_id", imageID.String()))

		render.JSON(w, r, Response{
			Response: response.OK(),
		})
	}
}
