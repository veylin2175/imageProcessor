package saveImage_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"imageProcessor/internal/http-server/handlers/image/saveImage"
	saverMocks "imageProcessor/internal/http-server/handlers/image/saveImage/mocks"
	kafkaMocks "imageProcessor/internal/kafka/producer/mocks"
	"imageProcessor/internal/models"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveImage(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(bytes.NewBuffer(nil), nil))

	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "uploads")
	os.Mkdir(uploadsDir, os.ModePerm)

	testUUID, _ := uuid.NewRandom()

	tests := []struct {
		name           string
		fileContent    []byte
		fileName       string
		mockImage      *models.Image
		mockSaveErr    error
		mockKafkaErr   error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			fileContent:    []byte("test file content"),
			fileName:       "test.jpg",
			mockImage:      &models.Image{ID: testUUID, Filename: "test.jpg", OriginalPath: "uploads/test.jpg"},
			mockSaveErr:    nil,
			mockKafkaErr:   nil,
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"status":"OK","image_id":"%s"}`, testUUID),
		},
		{
			name:           "Empty File",
			fileContent:    []byte(""),
			fileName:       "empty.jpg",
			mockImage:      nil,
			mockSaveErr:    nil,
			mockKafkaErr:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"received empty file"}`,
		},
		{
			name:           "Failed to Save Metadata",
			fileContent:    []byte("test file content"),
			fileName:       "test.jpg",
			mockImage:      nil,
			mockSaveErr:    errors.New("db error"),
			mockKafkaErr:   nil,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to save image metadata"}`,
		},
		{
			name:           "Failed to Publish to Kafka",
			fileContent:    []byte("test file content"),
			fileName:       "test.jpg",
			mockImage:      &models.Image{ID: testUUID, Filename: "test.jpg", OriginalPath: "uploads/test.jpg"},
			mockSaveErr:    nil,
			mockKafkaErr:   errors.New("kafka error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to start image processing"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageSaverMock := saverMocks.NewImageSaver(t)
			kafkaProducerMock := kafkaMocks.NewProducerIface(t)

			if tt.name == "Success" || tt.name == "Failed to Publish to Kafka" || tt.name == "Failed to Save Metadata" {
				imageSaverMock.On("SaveImage", mock.Anything, mock.Anything, mock.Anything).Return(tt.mockImage, tt.mockSaveErr).Once()
			}
			if tt.mockSaveErr == nil && tt.name != "Empty File" {
				kafkaProducerMock.On("SendMessage", mock.Anything, mock.Anything).Return(tt.mockKafkaErr).Once()
			}

			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("image", tt.fileName)
			require.NoError(t, err)
			part.Write(tt.fileContent)
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()

			originalCreate := osCreate
			osCreate = func(path string) (*os.File, error) {
				return originalCreate(filepath.Join(uploadsDir, filepath.Base(path)))
			}
			defer func() { osCreate = originalCreate }()

			handler := saveImage.New(log, imageSaverMock, kafkaProducerMock)
			handler.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedStatus, rr.Code)

			actualBody := rr.Body.String()
			var actualMap, expectedMap map[string]interface{}
			err = json.Unmarshal([]byte(actualBody), &actualMap)
			require.NoError(t, err)
			err = json.Unmarshal([]byte(tt.expectedBody), &expectedMap)
			require.NoError(t, err)
			require.Equal(t, expectedMap, actualMap)
		})
	}
}

var osCreate = os.Create
