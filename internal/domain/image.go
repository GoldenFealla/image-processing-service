package domain

// Image represents a domain entity for image processing
type Image struct {
	ID   string
	Data []byte
}

// ImageRepository defines the interface for image storage
type ImageRepository interface {
	Save(image *Image) error
	FindByID(id string) (*Image, error)
}

// ImageProcessor defines the interface for image processing operations
type ImageProcessor interface {
	Process(image *Image) (*Image, error)
}
