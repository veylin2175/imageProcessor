package getImage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"imageProcessor/internal/lib/api/response"
	"imageProcessor/internal/lib/logger/sl"
	"imageProcessor/internal/models"
	"log/slog"
	"net/http"
)

type Response struct {
	response.Response
	Image models.Image `json:"image"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=ImageGetter
type ImageGetter interface {
	GetImage(ctx context.Context, id uuid.UUID) (*models.Image, error)
}

func New(log *slog.Logger, imageGetter ImageGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.image.getImage.New"

		log = log.With(slog.String("op", op))

		idStr := chi.URLParam(r, "id")
		imageID, err := uuid.Parse(idStr)
		if err != nil {
			log.Error("failed to parse image ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid image ID"))
			return
		}

		image, err := imageGetter.GetImage(r.Context(), imageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Warn("image not found", slog.String("image_id", imageID.String()))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("image not found"))
				return
			}

			log.Error("failed to get image from storage", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get image"))
			return
		}

		log.Info("image retrieved successfully", slog.String("image_id", imageID.String()))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Image:    *image,
		})
	}
}
