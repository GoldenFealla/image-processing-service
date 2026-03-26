package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"html"
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
			switch f.Name {
			case domain.FilterGrayscale:
				current, err = applyGrayscale(current, f.Intensity)

			case domain.FilterSepia:
				current, err = applySepia(current, f.Intensity)

			case domain.FilterVivid:
				current, err = applyVivid(current, f.Intensity)

			case domain.FilterDystopian:
				current, err = applyDystopian(current, f.Intensity)

			case domain.FilterFilm:
				current, err = applyFilm(current, f.Intensity)

			case domain.FilterWild:
				current, err = applyWild(current, f.Intensity)

			case domain.FilterNoir:
				current, err = applyNoir(current, f.Intensity)
			}

			if err != nil {
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
func matrixImageFromArray(arr []float64, size int) (*vips.Image, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%d %d\n", size, size)

	for i, v := range arr {
		if i%size == 0 && i != 0 {
			buf.WriteString("\n")
		} else if i%size != 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(strconv.FormatFloat(v, 'f', 4, 64))
	}
	buf.WriteString("\n")

	source := vips.NewSource(io.NopCloser(&buf))
	return vips.NewMatrixloadSource(source, vips.DefaultMatrixloadSourceOptions())
}

func blendImages(base, overlay *vips.Image, t float64) (*vips.Image, error) {
	if t <= 0 {
		return base.Copy(nil)
	}
	if t >= 1 {
		return overlay.Copy(nil)
	}

	b, _ := base.Copy(nil)
	o, _ := overlay.Copy(nil)

	// base * (1 - t)
	if err := b.Linear([]float64{1 - t}, []float64{0}, nil); err != nil {
		return nil, err
	}

	// overlay * t
	if err := o.Linear([]float64{t}, []float64{0}, nil); err != nil {
		return nil, err
	}

	// add
	if err := b.Add(o); err != nil {
		return nil, err
	}

	return b, nil
}

func applyGrayscale(img *vips.Image, intensity float64) (*vips.Image, error) {
	if intensity <= 0 {
		return img.Copy(nil)
	}

	base, _ := img.Copy(nil)
	gray, _ := img.Copy(nil)

	if err := gray.Colourspace(vips.InterpretationBW, nil); err != nil {
		return nil, err
	}
	if err := gray.Colourspace(vips.InterpretationSrgb, nil); err != nil {
		return nil, err
	}

	return blendImages(base, gray, intensity)
}

var sepiaMatrix = []float64{
	0.3588, 0.7044, 0.1368,
	0.2990, 0.5870, 0.1140,
	0.2392, 0.4696, 0.0912,
}

func applySepia(img *vips.Image, intensity float64) (*vips.Image, error) {
	if intensity <= 0 {
		return img.Copy(nil)
	}

	base, _ := img.Copy(nil)
	sepia, _ := img.Copy(nil)

	m, err := matrixImageFromArray(sepiaMatrix, 3)
	if err != nil {
		return nil, err
	}

	if err := sepia.Recomb(m); err != nil {
		return nil, err
	}

	return blendImages(base, sepia, intensity)
}

func applyVivid(img *vips.Image, intensity float64) (*vips.Image, error) {
	if intensity <= 0 {
		return img.Copy(nil)
	}

	base, _ := img.Copy(nil)
	out, _ := img.Copy(nil)

	if err := out.Linear(
		[]float64{1 + 0.2*intensity},
		[]float64{-10 * intensity},
		nil,
	); err != nil {
		return nil, err
	}

	matrix := []float64{
		1.2, 0, 0,
		0, 1.2, 0,
		0, 0, 1.2,
	}

	m, _ := matrixImageFromArray(matrix, 3)
	if err := out.Recomb(m); err != nil {
		return nil, err
	}

	return blendImages(base, out, intensity)
}

func applyDystopian(img *vips.Image, intensity float64) (*vips.Image, error) {
	if intensity <= 0 {
		return img.Copy(nil)
	}

	base, _ := img.Copy(nil)
	out, _ := img.Copy(nil)

	matrix := []float64{
		0.8, 0.1, 0.1,
		0.0, 1.0, 0.0,
		0.1, 0.1, 0.9,
	}

	m, _ := matrixImageFromArray(matrix, 3)
	if err := out.Recomb(m); err != nil {
		return nil, err
	}

	if err := out.Linear(
		[]float64{1 - 0.1*intensity},
		[]float64{-20 * intensity},
		nil,
	); err != nil {
		return nil, err
	}

	return blendImages(base, out, intensity)
}

func applyFilm(img *vips.Image, intensity float64) (*vips.Image, error) {
	if intensity <= 0 {
		return img.Copy(nil)
	}

	base, _ := img.Copy(nil)
	out, _ := img.Copy(nil)

	if err := out.Linear(
		[]float64{1 - 0.1*intensity},
		[]float64{10 * intensity},
		nil,
	); err != nil {
		return nil, err
	}

	matrix := []float64{
		1.1, 0.05, 0,
		0.05, 1.0, 0,
		0, 0.05, 0.9,
	}

	m, _ := matrixImageFromArray(matrix, 3)
	if err := out.Recomb(m); err != nil {
		return nil, err
	}

	return blendImages(base, out, intensity)
}

func applyWild(img *vips.Image, intensity float64) (*vips.Image, error) {
	if intensity <= 0 {
		return img.Copy(nil)
	}

	base, _ := img.Copy(nil)
	out, _ := img.Copy(nil)

	if err := out.Linear(
		[]float64{1 + 0.4*intensity},
		[]float64{-15 * intensity},
		nil,
	); err != nil {
		return nil, err
	}

	matrix := []float64{
		1.5, 0, 0,
		0, 1.5, 0,
		0, 0, 1.5,
	}

	m, _ := matrixImageFromArray(matrix, 3)
	if err := out.Recomb(m); err != nil {
		return nil, err
	}

	return blendImages(base, out, intensity)
}

func applyNoir(img *vips.Image, intensity float64) (*vips.Image, error) {
	if intensity <= 0 {
		return img.Copy(nil)
	}

	base, _ := img.Copy(nil)
	out, _ := img.Copy(nil)

	if err := out.Colourspace(vips.InterpretationBW, nil); err != nil {
		return nil, err
	}
	if err := out.Linear([]float64{1.3}, []float64{-20}, nil); err != nil {
		return nil, err
	}
	if err := out.Colourspace(vips.InterpretationSrgb, nil); err != nil {
		return nil, err
	}

	return blendImages(base, out, intensity)
}
