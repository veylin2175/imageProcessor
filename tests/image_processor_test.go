package tests

import (
	"bytes"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	host = "0.0.0.0:8082"
)

func TestFullImageProcessingCycle(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	t.Run("Upload Image", func(t *testing.T) {
		filePath := "test_image.jpg"

		file, err := os.Open(filePath)
		require.NoError(t, err)
		defer file.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", filepath.Base(filePath))
		require.NoError(t, err)
		_, err = io.Copy(part, file)
		require.NoError(t, err)
		writer.Close()

		resp := e.POST("/upload").
			WithHeader("Content-Type", writer.FormDataContentType()).
			WithBytes(body.Bytes()).
			Expect().
			Status(http.StatusOK).
			JSON().Object().
			Value("image_id").String().NotEmpty()

		imageID := resp.Raw()

		t.Run("Get Image", func(t *testing.T) {
			time.Sleep(5 * time.Second)

			resp := e.GET("/image/" + imageID).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			resp.Value("image").Object().
				Value("ID").String().IsEqual(imageID)
			resp.Value("image").Object().
				Value("Status").String().IsEqual("processed")

			processedPath := resp.Value("image").Object().Value("ProcessedPathResize").Object().Value("String").String().Raw()
			e.GET("/" + processedPath).
				Expect().
				Status(http.StatusOK)
		})

		t.Run("Delete Image", func(t *testing.T) {
			e.DELETE("/image/" + imageID).
				Expect().
				Status(http.StatusOK).
				JSON().Object().
				Value("status").String().IsEqual("OK")

			e.GET("/image/" + imageID).
				Expect().
				Status(http.StatusNotFound)
		})
	})
}

func TestInvalidUpload(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	e.POST("/upload").
		Expect().
		Status(http.StatusBadRequest).
		JSON().Object().
		Value("error").String().Contains("file from request")
}

func TestGetImageNotFound(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := httpexpect.Default(t, u.String())

	nonExistentID := "00000000-0000-0000-0000-000000000000"

	e.GET("/image/" + nonExistentID).
		Expect().
		Status(http.StatusNotFound).
		JSON().Object().
		Value("error").String().Contains("not found")
}
