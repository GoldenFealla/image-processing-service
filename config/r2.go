package config

import (
	"os"

	"github.com/GoldenFealla/image-processing-service/internal/infrastructure"
)

func loadR2Config() *infrastructure.R2StorageConfig {
	return &infrastructure.R2StorageConfig{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("R2_BUCKET"),
		PublicURL:       os.Getenv("R2_PUBLIC_URL"),
	}
}
