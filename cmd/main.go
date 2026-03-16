package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/pg"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/valkey"
	"github.com/GoldenFealla/image-processing-service/internal/middleware"
	"github.com/GoldenFealla/image-processing-service/internal/presentation"
)

var (
	ImageCacheDB   int = 0
	SessionStoreDB int = 1
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file, ignored")
	}

	mainMux := http.NewServeMux()

	metadataRepo, err := pg.NewPostgresImageRepository(loadPostgresConfig())
	if err != nil {
		log.Fatalf("failed to initialize image metadata repository: %v", err)
	}
	defer metadataRepo.Close()
	storageRepo, err := infrastructure.NewR2Storage(loadR2Config())
	if err != nil {
		log.Fatalf("failed to initialize R2 storage: %v", err)
	}
	userRepo, err := pg.NewPostgresUserRepository(loadPostgresConfig())
	if err != nil {
		log.Fatalf("failed to initialize user repository: %v", err)
	}

	imageCache, err := valkey.NewImageCache(loadCacheConfig(ImageCacheDB))
	if err != nil {
		log.Fatalf("failed to initialize image cache: %v", err)
	}
	sessionStore, err := valkey.NewSessionStore(loadCacheConfig(SessionStoreDB))
	if err != nil {
		log.Fatalf("failed to initialize session store: %v", err)
	}

	imageProcessor := infrastructure.NewVipsImageProcessor()

	processUseCase := application.NewProcessImageService(metadataRepo, storageRepo, imageProcessor, imageCache)
	authUseCase := application.NewAuthService(userRepo, sessionStore, loadAuthConfig())

	// === presentation =====
	imageHandler := presentation.NewImageHandler(processUseCase)
	userHandler := presentation.NewAuthHandler(authUseCase)

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}).Handler(mainMux)

	mainMux.Handle("/images/", middleware.Chain(
		http.StripPrefix("/images", imageHandler.Routes()),
		middleware.JWTMiddleware(authUseCase),
	))

	mainMux.Handle("/auth/", http.StripPrefix("/auth", userHandler.Routes()))

	server := &http.Server{
		Addr:    "0.0.0.0:8081",
		Handler: middleware.Chain(handler, middleware.LoggerMiddleware),
	}
	log.Println("Listening on port 8081")
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

func loadPostgresConfig() *pg.PostgresConfig {
	return &pg.PostgresConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}
}

func loadCacheConfig(DB int) valkey.ValkeyConfig {
	return valkey.ValkeyConfig{
		Addr:     os.Getenv("VALKEY_ADDR"),
		Password: os.Getenv("VALKEY_PASSWORD"),
		DB:       DB,
		TTL:      30 * time.Minute,
	}
}

func loadAuthConfig() application.AuthConfig {
	privateKey, _ := loadECPrivateKey("ec256_private.pem")
	publicKey, _ := loadECPublicKey("ec256_public.pem")

	return application.AuthConfig{
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		PrivateKey:      privateKey,
		PublicKey:       publicKey,
		// GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		// GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		// GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}
}

func loadECPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	return x509.ParseECPrivateKey(block.Bytes)
}

func loadECPublicKey(path string) (*ecdsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub.(*ecdsa.PublicKey), nil
}
