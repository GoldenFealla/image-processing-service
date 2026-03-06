package infrastructure

import (
	"errors"
	"sync"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
)

// InMemoryImageRepository is an in-memory implementation of ImageRepository
type InMemoryImageRepository struct {
	mu     sync.RWMutex
	images map[string]*domain.Image
}

// NewInMemoryImageRepository creates a new InMemoryImageRepository
func NewInMemoryImageRepository() *InMemoryImageRepository {
	return &InMemoryImageRepository{
		images: make(map[string]*domain.Image),
	}
}

// Save saves an image to the repository
func (r *InMemoryImageRepository) Save(image *domain.Image) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.images[image.ID] = image
	return nil
}

// FindByID finds an image by ID
func (r *InMemoryImageRepository) FindByID(id string) (*domain.Image, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	image, exists := r.images[id]
	if !exists {
		return nil, errors.New("image not found")
	}
	return image, nil
}

// SimpleImageProcessor is a simple implementation of ImageProcessor
type SimpleImageProcessor struct{}

// NewSimpleImageProcessor creates a new SimpleImageProcessor
func NewSimpleImageProcessor() *SimpleImageProcessor {
	return &SimpleImageProcessor{}
}

// Process simulates image processing (e.g., resizing, filtering)
func (p *SimpleImageProcessor) Process(image *domain.Image) (*domain.Image, error) {
	// For simplicity, just return the same image
	// In a real implementation, this would modify the data
	processed := &domain.Image{
		ID:   image.ID + "_processed",
		Data: image.Data, // Simulate processing
	}
	return processed, nil
}
