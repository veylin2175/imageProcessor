package getImage_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"imageProcessor/internal/http-server/handlers/image/getImage"
	"imageProcessor/internal/http-server/handlers/image/getImage/mocks"
	"imageProcessor/internal/models"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetImage(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(bytes.NewBuffer(nil), nil))

	testUUID, _ := uuid.NewRandom()

	testImage := &models.Image{
		ID:                     testUUID,
		Filename:               "test.jpg",
		Status:                 "processed",
		OriginalPath:           "uploads/test.jpg",
		ProcessedPathResize:    sql.NullString{String: "processed/test_resized.jpg", Valid: true},
		ProcessedPathThumbnail: sql.NullString{String: "processed/test_thumbnail.jpg", Valid: true},
		ProcessedPathWatermark: sql.NullString{String: "processed/test_watermarked.jpg", Valid: true},
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	tests := []struct {
		name           string
		imageID        string
		mockImage      *models.Image
		mockErr        error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			imageID:        testUUID.String(),
			mockImage:      testImage,
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"status":"OK","image":{"ID":"%s","Filename":"test.jpg","Status":"processed","OriginalPath":"uploads/test.jpg","ProcessedPathResize":{"String":"processed/test_resized.jpg","Valid":true},"ProcessedPathThumbnail":{"String":"processed/test_thumbnail.jpg","Valid":true},"ProcessedPathWatermark":{"String":"processed/test_watermarked.jpg","Valid":true},"CreatedAt":"%s","UpdatedAt":"%s"}}`, testUUID, testImage.CreatedAt.Format(time.RFC3339Nano), testImage.UpdatedAt.Format(time.RFC3339Nano)),
		},
		{
			name:           "Invalid UUID",
			imageID:        "invalid-uuid",
			mockImage:      nil,
			mockErr:        nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid image ID"}`,
		},
		{
			name:           "Not Found",
			imageID:        testUUID.String(),
			mockImage:      nil,
			mockErr:        sql.ErrNoRows,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"Error","error":"image not found"}`,
		},
		{
			name:           "Internal Error",
			imageID:        testUUID.String(),
			mockImage:      nil,
			mockErr:        errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get image"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageGetterMock := mocks.NewImageGetter(t)

			if tt.name == "Success" {
				imageGetterMock.On("GetImage", mock.Anything, testUUID).Return(tt.mockImage, tt.mockErr).Once()
			} else if tt.name == "Not Found" {
				imageGetterMock.On("GetImage", mock.Anything, testUUID).Return(tt.mockImage, tt.mockErr).Once()
			} else if tt.name == "Internal Error" {
				imageGetterMock.On("GetImage", mock.Anything, testUUID).Return(tt.mockImage, tt.mockErr).Once()
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/image/%s", tt.imageID), nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.imageID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			handler := getImage.New(log, imageGetterMock)
			handler.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedStatus, rr.Code)

			actualBody := rr.Body.String()
			var actualMap, expectedMap map[string]interface{}
			err := json.Unmarshal([]byte(actualBody), &actualMap)
			require.NoError(t, err)
			err = json.Unmarshal([]byte(tt.expectedBody), &expectedMap)
			require.NoError(t, err)
			require.Equal(t, expectedMap, actualMap)
		})
	}
}
