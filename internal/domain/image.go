package domain

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
)

var (
	ErrImageNotFound    = errors.New("image not found")
	ErrUnsupportedImage = errors.New("unsupported image type")
	ErrForbidden        = errors.New("forbidden")
)

// Image represents a domain entity for image processing
type Image struct {
	ID       uuid.UUID `db:"id"       json:"id"`
	Name     string    `db:"name"     json:"name"`
	URL      string    `db:"url"      json:"url"`
	Version  int       `db:"version"  json:"version"`
	OwnerID  uuid.UUID `db:"owner_id" json:"owner_id"`
	UpdateAt time.Time `db:"updated_at"  json:"updated_at"`
}

// ImageRepository defines the interface for image storage
type ImageMetadataRepository interface {
	Save(ctx context.Context, image *Image) error
	FindByID(ctx context.Context, id uuid.UUID) (*Image, error)
	FindListByOwnerID(ctx context.Context, userID uuid.UUID) ([]*Image, error)
	Update(ctx context.Context, image *Image) error
}

type ImageStorageRepository interface {
	Upload(ctx context.Context, userID, id uuid.UUID, file io.Reader, contentType string) (string, error)
	Replace(ctx context.Context, userID, id uuid.UUID, file io.Reader, contentType string, version int) (string, error)
	Download(ctx context.Context, userID, id uuid.UUID) ([]byte, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

type ImageProcessor interface {
	Transform(ctx context.Context, data []byte, opts TransformOptions) ([]byte, error)
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
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	KeepAspect bool `json:"keep_aspect"`
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
	Text    string  `json:"text"`
	Size    float64 `json:"size"`
	Opacity float64 `json:"opacity"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
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
