package deleteImage_test

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
	"imageProcessor/internal/http-server/handlers/image/deleteImage"
	"imageProcessor/internal/http-server/handlers/image/deleteImage/mocks"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteImage(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(bytes.NewBuffer(nil), nil))

	testUUID, _ := uuid.NewRandom()

	tests := []struct {
		name           string
		imageID        string
		mockErr        error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			imageID:        testUUID.String(),
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK"}`,
		},
		{
			name:           "Invalid UUID",
			imageID:        "invalid-uuid",
			mockErr:        nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid image ID"}`,
		},
		{
			name:           "Not Found",
			imageID:        testUUID.String(),
			mockErr:        sql.ErrNoRows,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"Error","error":"image not found"}`,
		},
		{
			name:           "Internal Error",
			imageID:        testUUID.String(),
			mockErr:        errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to delete image"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageDeleterMock := mocks.NewImageDeleter(t)

			if tt.mockErr != nil {
				if tt.name == "Invalid UUID" {
				} else {
					imageDeleterMock.On("DeleteImage", mock.Anything, testUUID).Return(tt.mockErr).Once()
				}
			} else if tt.name == "Success" {
				imageDeleterMock.On("DeleteImage", mock.Anything, testUUID).Return(nil).Once()
			}

			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/image/%s", tt.imageID), nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.imageID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			handler := deleteImage.New(log, imageDeleterMock)
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
