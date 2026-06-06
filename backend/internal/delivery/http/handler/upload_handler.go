package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	uploadsDir    = "/uploads"
	maxUploadSize = 10 << 20 // 10 MB
)

var allowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".pdf":  true,
	".webp": true,
}

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		panic(fmt.Sprintf("upload handler: mkdir %s: %v", uploadsDir, err))
	}
	return &UploadHandler{}
}

// Upload handles POST /api/v1/upload.
// Accepts multipart field "file", saves to /uploads, returns JSON with the URL.
func (h *UploadHandler) Upload(c echo.Context) error {
	if err := c.Request().ParseMultipartForm(maxUploadSize); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file too large or invalid multipart"})
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "field 'file' missing"})
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file type not allowed; use jpg, png, pdf, or webp"})
	}

	filename := uuid.New().String() + ext
	dst, err := os.Create(filepath.Join(uploadsDir, filename))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not save file"})
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not write file"})
	}

	url := "/api/v1/uploads/" + filename
	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

// ServeFile handles GET /api/v1/uploads/:filename.
func (h *UploadHandler) ServeFile(c echo.Context) error {
	name := filepath.Base(c.Param("filename"))
	path := filepath.Join(uploadsDir, name)
	return c.File(path)
}
