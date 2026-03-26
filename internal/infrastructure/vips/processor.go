package vips

import (
	"context"
	"fmt"
	"html"
	"log"

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

	pipeline := []func(*vips.Image) error{
		v.crop(opts.Crop),
		v.resize(opts.Resize),
		v.rotate(opts.Rotate),
		v.flip(opts.Flip),
		v.mirror(opts.Mirror),
		v.watermark(opts.Watermark),
	}

	for _, fn := range pipeline {
		if err := fn(img); err != nil {
			return nil, err
		}
	}

	newImage, err := v.filters(opts.Filters)(img)
	if err != nil {
		return nil, err
	}

	return exportImage(newImage, opts)
}

func ensureExportable(img *vips.Image) error {
	// JPEG does NOT support alpha
	if img.HasAlpha() {
		if err := img.Flatten(nil); err != nil {
			return err
		}
	}

	// ensure correct colorspace
	if err := img.Colourspace(vips.InterpretationSrgb, nil); err != nil {
		return err
	}

	return nil
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

	if err := ensureExportable(img); err != nil {
		return nil, err
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

		hasWidth := opts.Width > 0
		hasHeight := opts.Height > 0

		if !hasWidth && !hasHeight {
			return nil
		}

		if opts.KeepAspect || (!hasWidth || !hasHeight) {
			// Scale by whichever dimension is provided
			// If both provided, scale by the constraining dimension
			var scale float64
			switch {
			case hasWidth && hasHeight:
				scaleX := float64(opts.Width) / float64(img.Width())
				scaleY := float64(opts.Height) / float64(img.Height())
				scale = min(scaleX, scaleY)
			case hasWidth:
				scale = float64(opts.Width) / float64(img.Width())
			case hasHeight:
				scale = float64(opts.Height) / float64(img.Height())
			}
			if err := img.Resize(scale, vips.DefaultResizeOptions()); err != nil {
				return fmt.Errorf("failed to resize: %w", err)
			}
		} else {
			// Exact resize, ignore aspect ratio
			hShrink := float64(img.Width()) / float64(opts.Width)
			vShrink := float64(img.Height()) / float64(opts.Height)
			if err := img.Reduceh(hShrink, nil); err != nil {
				return fmt.Errorf("failed to reduce width: %w", err)
			}
			if err := img.Reducev(vShrink, nil); err != nil {
				return fmt.Errorf("failed to reduce height: %w", err)
			}
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

func (v *VipsImageProcessor) filters(filters []domain.FilterOptions) func(*vips.Image) (*vips.Image, error) {
	return func(img *vips.Image) (*vips.Image, error) {
		current := img
		var err error

		for _, f := range filters {
			if f.Intensity <= 0 {
				continue
			}

			intensity := min(1, f.Intensity)

			switch f.Name {
			case domain.FilterGrayscale:
				current, err = applyGrayscale(current, intensity)
			case domain.FilterSepia:
				current, err = applySepia(current, intensity)
			case domain.FilterVivid:
				current, err = applyVivid(current, intensity)
			case domain.FilterDystopian:
				current, err = applyDystopian(current, intensity)
			case domain.FilterFilm:
				current, err = applyFilm(current, intensity)
			case domain.FilterWild:
				current, err = applyWild(current, intensity)
			case domain.FilterNoir:
				current, err = applyNoir(current, intensity)
			}

			if err != nil {
				log.Printf("error applying filter %v: err: %v", f, err)
				return nil, err
			}
		}

		return current, nil
	}
}

func (v *VipsImageProcessor) watermark(opts *domain.WatermarkOptions) func(*vips.Image) error {
	return func(img *vips.Image) error {
		if opts == nil {
			return nil
		}

		svg := fmt.Sprintf(
			`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">
                <text x="0" y="%g" font-size="%g" fill="rgba(255,255,255,%.2f)" font-family="sans-serif">%s</text>
            </svg>`,
			int(float64(len(opts.Text))*opts.Size*0.6),
			int(opts.Size),
			opts.Size,
			opts.Size,
			opts.Opacity,
			html.EscapeString(opts.Text),
		)

		watermark, err := vips.NewImageFromBuffer([]byte(svg), nil)
		if err != nil {
			return fmt.Errorf("create watermark: %w", err)
		}
		defer watermark.Close()

		return img.Composite2(watermark, vips.BlendModeOver, &vips.Composite2Options{
			X:                opts.X,
			Y:                opts.Y,
			CompositingSpace: vips.Interpretation(22),
		})
	}
}

// ========= helper =========
