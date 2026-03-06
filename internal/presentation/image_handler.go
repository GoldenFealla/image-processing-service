package presentation

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/GoldenFealla/image-processing-service/internal/application"
)

const (
	MaxByte = 10 << 20
)

type ImageHandler struct {
	imageProcessor application.ImageProcessor
}

func NewImageHandler(imageProcessor application.ImageProcessor) *ImageHandler {
	return &ImageHandler{
		imageProcessor: imageProcessor,
	}
}

func (h *ImageHandler) Routes() *http.ServeMux {
	r := http.NewServeMux()

	r.HandleFunc("GET /:id", h.RetreiveImage)
	r.HandleFunc("POST /", h.UploadImage)
	r.HandleFunc("POST /:id/transform", h.TransformImage)

	return r
}

func (h *ImageHandler) RetreiveImage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Retreive"))
}

func (h *ImageHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxByte)

	if err := r.ParseMultipartForm(MaxByte); err != nil {
		http.Error(w, "file too large or invalid form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing 'image' field form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	image, err := h.imageProcessor.Upload(r.Context(), file)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrUnsupportedImage):
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(image)
}

func (h *ImageHandler) TransformImage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Transform"))
}
