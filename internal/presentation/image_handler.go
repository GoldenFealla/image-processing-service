package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/GoldenFealla/image-processing-service/internal/application"
)

// ImageHandler handles HTTP requests for images
type ImageHandler struct {
	processUseCase *application.ProcessImageUseCase
}

// NewImageHandler creates a new ImageHandler
func NewImageHandler(processUseCase *application.ProcessImageUseCase) *ImageHandler {
	return &ImageHandler{
		processUseCase: processUseCase,
	}
}

// ProcessImage handles POST /process/{id}
func (h *ImageHandler) ProcessImage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/process/"):] // Simple path parsing

	err := h.processUseCase.Execute(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}
