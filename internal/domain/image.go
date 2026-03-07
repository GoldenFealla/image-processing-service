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
	ID      uuid.UUID `json:"id"`
	URL     string    `json:"url"`
	Version int       `json:"version"`
}

// ImageRepository defines the interface for image storage
type ImageMetadataRepository interface {
	Save(ctx context.Context, image *Image) error
	FindByID(ctx context.Context, id uuid.UUID) (*Image, error)
	Update(ctx context.Context, image *Image) error
}

type ImageStorageRepository interface {
	Upload(ctx context.Context, id uuid.UUID, file io.Reader, contentType string) (string, error)
	Replace(ctx context.Context, id uuid.UUID, file io.Reader, contentType string) (string, error)
	Download(ctx context.Context, id uuid.UUID) ([]byte, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ImageProcessor interface {
	Transform(ctx context.Context, data []byte, opts TransformOptions) ([]byte, error)
}

type ImageCache interface {
	GetOriginal(ctx context.Context, id uuid.UUID) ([]byte, error)
	SetOriginal(ctx context.Context, id uuid.UUID, data []byte) error
	GetTransformed(ctx context.Context, id uuid.UUID, opts TransformOptions) ([]byte, error)
	SetTransformed(ctx context.Context, id uuid.UUID, opts TransformOptions, data []byte) error
}

type Format string
type Filter string

const (
	FormatJPEG Format = "jpeg"
	FormatPNG  Format = "png"
	FormatWebP Format = "webp"
)

const (
	FilterGrayscale Filter = "grayscale"
	FilterSepia     Filter = "sepia"
)

type ResizeOptions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type CropOptions struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type RotateOptions struct {
	Angle float64 `json:"angle"` // any
}

type WatermarkOptions struct {
	Text     string  `json:"text"`
	Opacity  float64 `json:"opacity"`  // 0.0 - 1.0
	Position string  `json:"position"` // top-left, top-right, bottom-left, bottom-right, center
}

type CompressOptions struct {
	Quality int `json:"quality"` // 1 - 100
}

type TransformOptions struct {
	Resize    *ResizeOptions    `json:"resize,omitempty"`
	Crop      *CropOptions      `json:"crop,omitempty"`
	Rotate    *RotateOptions    `json:"rotate,omitempty"`
	Watermark *WatermarkOptions `json:"watermark,omitempty"`
	Compress  *CompressOptions  `json:"compress,omitempty"`
	Flip      bool              `json:"flip,omitempty"`   // vertical
	Mirror    bool              `json:"mirror,omitempty"` // horizontal
	Format    *Format           `json:"format,omitempty"`
	Filters   []Filter          `json:"filters,omitempty"`
}
