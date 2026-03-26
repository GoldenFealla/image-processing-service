package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"github.com/GoldenFealla/image-processing-service/config"
	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/vips"
	"github.com/GoldenFealla/image-processing-service/internal/middleware"
	"github.com/GoldenFealla/image-processing-service/internal/presentation"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file, ignored")
	}

	// When want to change Tech, just implement those interface in domain
	// And Initialize the tech
	config.InitializePostgres()
	config.InitializeStore()
	config.InitializeSigningKey()
	config.InitializeR2()
	config.InitializeOAuthRepository()
}

func main() {
	mainMux := http.NewServeMux()

	imageProcessor := vips.NewVipsImageProcessor()

	// === service ==========
	processService := application.NewProcessImageService(
		config.ImageMetadataRepository,
		config.ImageStorageRepository,
		imageProcessor,
		config.ImageStore,
	)
	authService := application.NewAuthService(application.AuthServiceConfig{
		JWTSigningKeyConfig:    config.JWTSigningKeyConfig,
		UserRepository:         config.UserRepository,
		UserIdentityRepository: config.UserIdentityRepository,
		SessionStore:           config.SessionStore,
		GoogleOAuth:            config.GoogleOAuthRepository,
		GithubOAuth:            config.GithubOAuthRepository,
	})

	// === presentation =====
	imageHandler := presentation.NewImageHandler(processService)
	userHandler := presentation.NewAuthHandler(
		authService,
		config.OAuthStateStore,
		os.Getenv("REDIRECT_URL"),
	)

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}).Handler(mainMux)

	mainMux.Handle("/images/", middleware.Chain(
		http.StripPrefix("/images", imageHandler.Routes()),
		middleware.JWTMiddleware(authService),
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
