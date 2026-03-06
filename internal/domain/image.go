package domain

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
)

var ErrImageNotFound = errors.New("image not found")

// Image represents a domain entity for image processing
type Image struct {
	ID  uuid.UUID `json:"id"`
	URL string    `json:"url"`
}

// ImageRepository defines the interface for image storage
type ImageMetadataRepository interface {
	Save(ctx context.Context, image *Image) error
	FindByID(ctx context.Context, id uuid.UUID) (*Image, error)
}

type ImageStorageRepository interface {
	Upload(ctx context.Context, id uuid.UUID, file io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
