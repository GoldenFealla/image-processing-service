package main

import (
	"log"
	"net/http"
	"os"

	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure"
	"github.com/GoldenFealla/image-processing-service/internal/presentation"
)

func main() {
	imageRepo := infrastructure.NewInMemoryImageRepository()

	// simple processor stays the same for now; you could replace it with a
	// more advanced implementation later
	processor := infrastructure.NewSimpleImageProcessor()

	// Application layer
	processUseCase := application.NewProcessImageUseCase(imageRepo, processor)

	// Presentation layer
	imageHandler := presentation.NewImageHandler(processUseCase)

	// Routes
	http.HandleFunc("/process/", imageHandler.ProcessImage)

	// Start server
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
