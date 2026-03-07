package presentation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/google/uuid"
)

const (
	maxUploadBytes = 10 << 20
)

type ImageHandler struct {
	imageProcessor application.ProcessImageUseCase
}

func NewImageHandler(imageProcessor application.ProcessImageUseCase) *ImageHandler {
	return &ImageHandler{
		imageProcessor: imageProcessor,
	}
}

func (h *ImageHandler) Routes() *http.ServeMux {
	r := http.NewServeMux()

	r.HandleFunc("GET /{id}", h.RetrieveImage)
	r.HandleFunc("POST /", h.UploadImage)
	r.HandleFunc("POST /{id}/transform", h.TransformImage)
	r.HandleFunc("PUT /{id}/save", h.SaveImage)

	return r
}

func (h *ImageHandler) RetrieveImage(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid image id", http.StatusBadRequest)
		return
	}

	image, err := h.imageProcessor.Retrieve(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrImageNotFound):
			http.Error(w, "image not found", http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(image)
}

func (h *ImageHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		http.Error(w, "request body too large or invalid form data", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing or invalid 'image' field in form", http.StatusBadRequest)
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
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(image)
}

func (h *ImageHandler) TransformImage(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid image id", http.StatusBadRequest)
		return
	}

	var opts domain.TransformOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		http.Error(w, fmt.Sprintf("decode body: %v", err), http.StatusBadRequest)
		return
	}

	image, err := h.imageProcessor.Transform(r.Context(), id, opts)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrImageNotFound):
			http.Error(w, "image not found", http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(image)
}

func (h *ImageHandler) SaveImage(w http.ResponseWriter, r *http.Request) {
}
