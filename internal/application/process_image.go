package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/google/uuid"
)

var (
	allowedImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/wepb": true,
	}
)

type ProcessImageUseCase interface {
	Retrieve(ctx context.Context, userID, id uuid.UUID) (*domain.Image, error)
	List(ctx context.Context, userID uuid.UUID) ([]*domain.Image, error)
	Upload(ctx context.Context, userID uuid.UUID, file multipart.File, name string) (*domain.Image, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	Save(ctx context.Context, userID, id uuid.UUID, opts domain.TransformOptions) (*domain.Image, error)
	Transform(ctx context.Context, userID, id uuid.UUID, opts domain.TransformOptions) ([]byte, error)
}

type ProcessImageService struct {
	cache     domain.ImageStore
	metadata  domain.ImageMetadataRepository
	storage   domain.ImageStorageRepository
	processor domain.ImageProcessor
}

func NewProcessImageService(
	metadata domain.ImageMetadataRepository,
	storage domain.ImageStorageRepository,
	processor domain.ImageProcessor,
	cache domain.ImageStore,
) *ProcessImageService {
	return &ProcessImageService{
		metadata:  metadata,
		storage:   storage,
		processor: processor,
		cache:     cache,
	}
}

func (pis *ProcessImageService) Retrieve(ctx context.Context, userid, id uuid.UUID) (*domain.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	image, err := pis.metadata.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func (pis *ProcessImageService) List(ctx context.Context, userID uuid.UUID) ([]*domain.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	images, err := pis.metadata.FindListByOwnerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (pis *ProcessImageService) Upload(ctx context.Context, userID uuid.UUID, file multipart.File, name string) (*domain.Image, error) {
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
		return nil, domain.ErrUnsupportedImage
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
	url, err := pis.storage.Upload(ctx, userID, newID, file, contentType)
	if err != nil {
		return nil, err
	}

	newImage := &domain.Image{
		ID:      newID,
		Name:    name,
		URL:     url,
		Version: 0,
		OwnerID: userID,
	}
	err = pis.metadata.Save(ctx, newImage)
	if err != nil {
		if delErr := pis.storage.Delete(context.WithoutCancel(ctx), userID, newID); delErr != nil {
			err = errors.Join(err, delErr)
		}
		return nil, err
	}

	return newImage, nil
}

func (pis *ProcessImageService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	err := pis.metadata.Delete(ctx, id)
	if err != nil {
		return err
	}

	go func() {
		if err := pis.storage.Delete(context.WithoutCancel(ctx), userID, id); err != nil {
			log.Println("failed to delete image from storage", "error", err, "id", id)
		}
	}()
	return nil
}

func (pis *ProcessImageService) Save(ctx context.Context, userID, id uuid.UUID, opts domain.TransformOptions) (*domain.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	existing, err := pis.metadata.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != userID {
		return nil, domain.ErrForbidden
	}

	original, err := pis.cache.GetOriginal(ctx, id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		original, err = pis.storage.Download(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		go pis.cache.SetOriginal(context.WithoutCancel(ctx), id, original)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	transformed, err := pis.processor.Transform(ctx, original, opts)
	if err != nil {
		return nil, err
	}

	contentType := http.DetectContentType(transformed)
	url, err := pis.storage.Replace(ctx, userID, id, bytes.NewReader(transformed), contentType, existing.Version+1)
	if err != nil {
		return nil, err
	}

	existing.URL = url
	existing.Version++
	if err := pis.metadata.Update(ctx, existing); err != nil {
		return nil, err
	}

	// Invalidate caches asynchronously
	go func() {
		bgCtx := context.WithoutCancel(ctx)
		pis.cache.DeleteOriginal(bgCtx, id)
		pis.cache.DeleteTransformed(bgCtx, id)
	}()

	return existing, nil
}

func (pis *ProcessImageService) Transform(ctx context.Context, userID, id uuid.UUID, opts domain.TransformOptions) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cachedTransform, err := pis.cache.GetTransformed(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	if cachedTransform != nil {
		return cachedTransform, nil
	}

	original, err := pis.cache.GetOriginal(ctx, id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		original, err = pis.storage.Download(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		go pis.cache.SetOriginal(context.WithoutCancel(ctx), id, original)
	}

	transformed, err := pis.processor.Transform(ctx, original, opts)
	if err != nil {
		return nil, err
	}

	go pis.cache.SetTransformed(context.WithoutCancel(ctx), id, opts, transformed)
	return transformed, nil
}
