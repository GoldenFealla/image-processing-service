package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure"
	"github.com/GoldenFealla/image-processing-service/internal/presentation"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}

	r2Config := &infrastructure.R2StorageConfig{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("R2_BUCKET"),
		PublicURL:       os.Getenv("R2_PUBLIC_URL"),
	}

	pgConfig := &infrastructure.PostgresConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}

	mainMux := http.NewServeMux()

	// === infrastructure ===
	metadataRepo, err := infrastructure.NewPostgresImageRepository(pgConfig)
	if err != nil {
		log.Fatalf("failed to initialize postgres repository: %v", err)
	}
	defer metadataRepo.Close()

	storageRepo, err := infrastructure.NewR2Storage(r2Config)
	if err != nil {
		log.Fatalf("failed to initialize R2 storage: %v", err)
	}

	// === application ======
	processUseCase := application.NewProcessImageService(metadataRepo, storageRepo)

	// === presentation =====
	imageHandler := presentation.NewImageHandler(processUseCase)

	mainMux.Handle("/images/", http.StripPrefix("/images", imageHandler.Routes()))

	server := &http.Server{Addr: "localhost:8080", Handler: mainMux}
	log.Println("Listening on port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
