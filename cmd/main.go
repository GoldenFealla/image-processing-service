package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure"
	"github.com/GoldenFealla/image-processing-service/internal/middleware"
	"github.com/GoldenFealla/image-processing-service/internal/presentation"
)

var (
	ImageCacheDB int = 0
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}

	mainMux := http.NewServeMux()

	metadataRepo, err := infrastructure.NewPostgresImageRepository(loadPostgresConfig())
	if err != nil {
		log.Fatalf("failed to initialize postgres repository: %v", err)
	}
	defer metadataRepo.Close()
	storageRepo, err := infrastructure.NewR2Storage(loadR2Config())
	if err != nil {
		log.Fatalf("failed to initialize R2 storage: %v", err)
	}
	imageCache, err := infrastructure.NewImageCache(loadCacheConfig(ImageCacheDB))
	if err != nil {
		log.Fatalf("failed to initialize image cache: %v", err)
	}
	imageProcessor := infrastructure.NewVipsImageProcessor()

	processUseCase := application.NewProcessImageService(metadataRepo, storageRepo, imageProcessor, imageCache)

	// === presentation =====
	imageHandler := presentation.NewImageHandler(processUseCase)

	mainMux.Handle("/images/", http.StripPrefix("/images", imageHandler.Routes()))

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: middleware.Chain(mainMux, middleware.LoggerMiddleware),
	}
	log.Println("Listening on port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func loadR2Config() *infrastructure.R2StorageConfig {
	return &infrastructure.R2StorageConfig{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("R2_BUCKET"),
		PublicURL:       os.Getenv("R2_PUBLIC_URL"),
	}
}

func loadPostgresConfig() *infrastructure.PostgresConfig {
	return &infrastructure.PostgresConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}
}

func loadCacheConfig(DB int) infrastructure.ImageCacheConfig {
	return infrastructure.ImageCacheConfig{
		Addr:     os.Getenv("VALKEY_ADDR"),
		Password: os.Getenv("VALKEY_PASSWORD"),
		DB:       DB,
		TTL:      30 * time.Minute,
	}
}
