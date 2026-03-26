package vips

import "github.com/cshum/vipsgen/vips"

func blendImages(base, overlay *vips.Image, t float64) (*vips.Image, error) {
	if t <= 0 {
		return base, nil
	}
	if t >= 1 {
		return overlay, nil
	}

	b, err := base.Copy(nil)
	if err != nil {
		return nil, err
	}

	o, err := overlay.Copy(nil)
	if err != nil {
		return nil, err
	}

	if err := b.Linear([]float64{1 - t}, []float64{0}, nil); err != nil {
		return nil, err
	}

	if err := o.Linear([]float64{t}, []float64{0}, nil); err != nil {
		return nil, err
	}

	if err := b.Add(o); err != nil {
		return nil, err
	}

	return b, nil
}

func applyWithIntensity(
	img *vips.Image,
	intensity float64,
	full func(*vips.Image) error,
) (*vips.Image, error) {

	if intensity <= 0 {
		return img, nil
	}

	if intensity >= 1 {
		return img, full(img)
	}

	working, err := img.Copy(nil)
	if err != nil {
		return nil, err
	}

	if err := full(working); err != nil {
		return nil, err
	}

	return blendImages(img, working, intensity)
}

func applyGrayscale(img *vips.Image, intensity float64) (*vips.Image, error) {
	return applyWithIntensity(img, intensity, func(im *vips.Image) error {
		if err := im.Colourspace(vips.InterpretationBW, nil); err != nil {
			return err
		}
		return im.Colourspace(vips.InterpretationSrgb, nil)
	})
}

func applySepia(img *vips.Image, intensity float64) (*vips.Image, error) {
	return applyWithIntensity(img, intensity, func(im *vips.Image) error {
		return im.Recomb(sepiaMatrixImage)
	})
}

func applyVivid(img *vips.Image, intensity float64) (*vips.Image, error) {
	return applyWithIntensity(img, intensity, func(im *vips.Image) error {

		if err := im.Linear(
			[]float64{1 + 0.2*intensity},
			[]float64{-10 * intensity},
			nil,
		); err != nil {
			return err
		}

		return im.Recomb(vividMatrixImage)
	})
}

func applyDystopian(img *vips.Image, intensity float64) (*vips.Image, error) {
	return applyWithIntensity(img, intensity, func(im *vips.Image) error {

		if err := im.Recomb(dystopianMatrixImage); err != nil {
			return err
		}

		return im.Linear(
			[]float64{1 - 0.1*intensity},
			[]float64{-20 * intensity},
			nil,
		)
	})
}

func applyFilm(img *vips.Image, intensity float64) (*vips.Image, error) {
	return applyWithIntensity(img, intensity, func(im *vips.Image) error {

		if err := im.Linear(
			[]float64{1 - 0.1*intensity},
			[]float64{10 * intensity},
			nil,
		); err != nil {
			return err
		}

		return im.Recomb(filmMatrixImage)
	})
}

func applyWild(img *vips.Image, intensity float64) (*vips.Image, error) {
	return applyWithIntensity(img, intensity, func(im *vips.Image) error {

		if err := im.Linear(
			[]float64{1 + 0.4*intensity},
			[]float64{-15 * intensity},
			nil,
		); err != nil {
			return err
		}

		return im.Recomb(wildMatrixImage)
	})
}

func applyNoir(img *vips.Image, intensity float64) (*vips.Image, error) {
	return applyWithIntensity(img, intensity, func(im *vips.Image) error {

		if err := im.Colourspace(vips.InterpretationBW, nil); err != nil {
			return err
		}

		if err := im.Linear(
			[]float64{1.3},
			[]float64{-20},
			nil,
		); err != nil {
			return err
		}

		return im.Colourspace(vips.InterpretationSrgb, nil)
	})
}
