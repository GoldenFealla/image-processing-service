package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/cshum/vipsgen/vips"
)

// infrastructure/vips_image_processor.go
type VipsImageProcessor struct{}

func NewVipsImageProcessor() *VipsImageProcessor {
	return &VipsImageProcessor{}
}

func (v *VipsImageProcessor) Transform(ctx context.Context, data []byte, opts domain.TransformOptions) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	loadOpts := vips.DefaultLoadOptions()
	img, err := vips.NewImageFromBuffer(data, loadOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %w", err)
	}
	defer img.Close()

	fmt.Printf("DEBUG loaded: width=%d height=%d bands=%d colorspace=%v hasAlpha=%v\n",
		img.Width(), img.Height(), img.Bands(), img.Interpretation(), img.HasAlpha())
	fmt.Printf("DEBUG format: %s\n", img.Format())
	fmt.Printf("DEBUG filter: %s\n", opts.Filters)

	pipeline := []func(*vips.Image) error{
		v.crop(opts.Crop),
		v.resize(opts.Resize),
		v.rotate(opts.Rotate),
		v.flip(opts.Flip),
		v.mirror(opts.Mirror),
		v.filters(opts.Filters),
		v.watermark(opts.Watermark),
	}

	for _, fn := range pipeline {
		if err := fn(img); err != nil {
			return nil, err
		}
	}

	return exportImage(img, opts)
}

func exportImage(img *vips.Image, opts domain.TransformOptions) ([]byte, error) {
	quality := 85 // default
	if opts.Compress != nil {
		quality = opts.Compress.Quality
	}

	format := domain.FormatJPEG // default
	if opts.Format != nil {
		format = *opts.Format
	}

	switch format {
	case domain.FormatPNG:
		params := vips.DefaultPngsaveBufferOptions()
		params.Q = quality
		buf, err := img.PngsaveBuffer(params)
		return buf, err
	case domain.FormatWebP:
		params := vips.DefaultWebpsaveBufferOptions()
		params.Q = quality
		buf, err := img.WebpsaveBuffer(params)
		return buf, err
	default:
		params := vips.DefaultJpegsaveBufferOptions()
		params.Q = quality
		buf, err := img.JpegsaveBuffer(params)
		return buf, err
	}
}

// ========= operation =========
func (v *VipsImageProcessor) crop(opts *domain.CropOptions) func(*vips.Image) error {
	return func(img *vips.Image) error {
		if opts == nil {
			return nil
		}
		if err := img.ExtractArea(opts.X, opts.Y, opts.Width, opts.Height); err != nil {
			return fmt.Errorf("failed to crop: %w", err)
		}
		return nil
	}
}

func (v *VipsImageProcessor) resize(opts *domain.ResizeOptions) func(*vips.Image) error {
	return func(img *vips.Image) error {
		if opts == nil {
			return nil
		}
		scaleX := float64(opts.Width) / float64(img.Width())
		scaleY := float64(opts.Height) / float64(img.Height())
		scale := min(scaleX, scaleY)
		if err := img.Resize(scale, vips.DefaultResizeOptions()); err != nil {
			return fmt.Errorf("failed to resize: %w", err)
		}
		return nil
	}
}

func (v *VipsImageProcessor) rotate(opts *domain.RotateOptions) func(*vips.Image) error {
	return func(img *vips.Image) error {
		if opts == nil {
			return nil
		}
		if err := img.Rotate(opts.Angle, vips.DefaultRotateOptions()); err != nil {
			return fmt.Errorf("failed to rotate: %w", err)
		}
		return nil
	}
}

func (v *VipsImageProcessor) flip(enabled bool) func(*vips.Image) error {
	return func(img *vips.Image) error {
		if !enabled {
			return nil
		}
		if err := img.Flip(vips.DirectionVertical); err != nil {
			return fmt.Errorf("failed to flip: %w", err)
		}
		return nil
	}
}

func (v *VipsImageProcessor) mirror(enabled bool) func(*vips.Image) error {
	return func(img *vips.Image) error {
		if !enabled {
			return nil
		}
		if err := img.Flip(vips.DirectionHorizontal); err != nil {
			return fmt.Errorf("failed to mirror: %w", err)
		}
		return nil
	}
}

func (v *VipsImageProcessor) filters(filters []domain.Filter) func(*vips.Image) error {
	return func(img *vips.Image) error {
		for _, filter := range filters {
			switch filter {
			case domain.FilterGrayscale:
				{
					if err := img.Colourspace(vips.InterpretationBW, vips.DefaultColourspaceOptions()); err != nil {
						return fmt.Errorf("failed to apply grayscale: %w", err)
					}
					if err := img.Colourspace(vips.InterpretationSrgb, vips.DefaultColourspaceOptions()); err != nil {
						return fmt.Errorf("failed to apply grayscale: %w", err)
					}
				}
			case domain.FilterSepia:
				{
					if err := applySepia(img); err != nil {
						return fmt.Errorf("failed to apply sepia: %w", err)
					}
				}
			}
			// TODO: Add more filter
		}
		return nil
	}
}

func (v *VipsImageProcessor) watermark(opts *domain.WatermarkOptions) func(*vips.Image) error {
	return func(img *vips.Image) error {
		if opts == nil {
			return nil
		}
		watermark, err := vips.NewImageFromBuffer([]byte(fmt.Sprintf(
			`<svg><text opacity="%f">%s</text></svg>`,
			opts.Opacity, opts.Text,
		)), nil)
		if err != nil {
			return fmt.Errorf("failed to create watermark: %w", err)
		}
		defer watermark.Close()

		x, y := resolveWatermarkPosition(img, watermark, opts.Position)
		if err := img.Composite2(
			watermark,
			vips.BlendModeOver,
			&vips.Composite2Options{X: x, Y: y, CompositingSpace: vips.Interpretation(22)},
		); err != nil {
			return fmt.Errorf("failed to apply watermark: %w", err)
		}
		return nil
	}
}

// ========= helper =========
func resolveWatermarkPosition(img *vips.Image, watermark *vips.Image, position string) (int, int) {
	switch position {
	case "top-left":
		return 10, 10
	case "top-right":
		return img.Width() - watermark.Width() - 10, 10
	case "bottom-left":
		return 10, img.Height() - watermark.Height() - 10
	case "bottom-right":
		return img.Width() - watermark.Width() - 10, img.Height() - watermark.Height() - 10
	default: // center
		return (img.Width() - watermark.Width()) / 2, (img.Height() - watermark.Height()) / 2
	}
}

var sepiaMatrix = []float64{
	0.3588, 0.7044, 0.1368,
	0.2990, 0.5870, 0.1140,
	0.2392, 0.4696, 0.0912,
}

func sepiaMatrixReader() io.ReadCloser {
	var buf bytes.Buffer
	// VIPS matrix format: first line is "width height"
	// then rows of space-separated floats
	buf.WriteString("3 3\n")
	for i, v := range sepiaMatrix {
		if i%3 == 0 && i != 0 {
			buf.WriteString("\n")
		} else if i%3 != 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(strconv.FormatFloat(v, 'f', 4, 64))
	}
	buf.WriteString("\n")
	return io.NopCloser(&buf)
}

func applySepia(img *vips.Image) error {
	if img.HasAlpha() {
		if err := img.Flatten(nil); err != nil {
			return err
		}
	}

	matrixSource := vips.NewSource(sepiaMatrixReader())
	matrixImage, err := vips.NewMatrixloadSource(matrixSource, vips.DefaultMatrixloadSourceOptions())
	if err != nil {
		return err
	}
	return img.Recomb(matrixImage)
}
