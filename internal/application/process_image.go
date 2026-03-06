package application

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/google/uuid"
)

var (
	AllowedImageType = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/wepb": true,
	}
)

var (
	ErrUnsupportedImage = errors.New("unsupported image type")
)

type ImageProcessor interface {
	Upload(ctx context.Context, file multipart.File) (*domain.Image, error)
}

type ProcessImageService struct {
	metadata domain.ImageMetadataRepository
	storage  domain.ImageStorageRepository
}

func NewProcessImageService(
	metadata domain.ImageMetadataRepository,
	storage domain.ImageStorageRepository,
) *ProcessImageService {
	return &ProcessImageService{
		metadata: metadata,
		storage:  storage,
	}
}

func (pis *ProcessImageService) Upload(ctx context.Context, file multipart.File) (*domain.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	buffer := make([]byte, 512)
	_, err := io.ReadFull(file, buffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	contentType := http.DetectContentType(buffer)
	if !AllowedImageType[contentType] {
		return nil, ErrUnsupportedImage
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	// Save image
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	newID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	url, err := pis.storage.Upload(ctx, newID, file, contentType)
	if err != nil {
		return nil, err
	}

	newImage := &domain.Image{
		ID:  newID,
		URL: url,
	}
	err = pis.metadata.Save(ctx, newImage)
	if err != nil {
		pis.storage.Delete(context.WithoutCancel(ctx), newID)
		return nil, err
	}

	return newImage, nil
}
