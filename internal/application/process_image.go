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
	// allowedImageTypes is an immutable set of MIME types we process
	allowedImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/wepb": true,
	}
)

var (
	ErrUnsupportedImage = errors.New("unsupported image type")
)

type ProcessImageUseCase interface {
	Retrieve(ctx context.Context, id uuid.UUID) (*domain.Image, error)
	Upload(ctx context.Context, file multipart.File) (*domain.Image, error)
	Save(ctx context.Context, id uuid.UUID, file multipart.File) error
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
	if !allowedImageTypes[contentType] {
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
		if delErr := pis.storage.Delete(context.WithoutCancel(ctx), newID); delErr != nil {
			err = errors.Join(err, delErr)
		}
		return nil, err
	}

	return newImage, nil
}

func (pis *ProcessImageService) Retrieve(ctx context.Context, id uuid.UUID) (*domain.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	image, err := pis.metadata.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func (pis *ProcessImageService) Save(ctx context.Context, id uuid.UUID, file multipart.File) error {
	return nil
}
