package application

import (
	"github.com/GoldenFealla/image-processing-service/internal/domain"
)

// ProcessImageUseCase handles the business logic for processing images
type ProcessImageUseCase struct {
	repo      domain.ImageRepository
	processor domain.ImageProcessor
}

// NewProcessImageUseCase creates a new ProcessImageUseCase
func NewProcessImageUseCase(repo domain.ImageRepository, processor domain.ImageProcessor) *ProcessImageUseCase {
	return &ProcessImageUseCase{
		repo:      repo,
		processor: processor,
	}
}

// Execute processes an image and saves it
func (uc *ProcessImageUseCase) Execute(id string) error {
	image, err := uc.repo.FindByID(id)
	if err != nil {
		return err
	}

	processedImage, err := uc.processor.Process(image)
	if err != nil {
		return err
	}

	return uc.repo.Save(processedImage)
}
