# image-processing-service

## Description
A backend service for image processing built with Go, using a minimal (vanilla) approach and high-performance libraries.

The system is designed to handle image transformations efficiently and scale as part of a microservices architecture.

Tech stack:
- Go (vanilla, minimal framework usage)
- PostgreSQL (metadata storage)
- Libvips (fast image processing)
- Valkey (caching layer)
- R2 (object storage)

Key features:
- Image upload and storage
- High-performance processing via Libvips
- Metadata management with PostgreSQL
- Caching with Valkey
- Object storage integration (R2)

## Installation

### Requirements
- Go (1.20+ recommended)
- PostgreSQL
- Libvips
- Valkey
- R2 account (or S3 compatible object storage)

### Platforms
- Linux
- Windows

### Install
git clone https://github.com/GoldenFealla/image-processing-service  
cd image-processing-service  
go mod tidy  

## Instruction

### Run the service
go run main.go

### Usage
- Start required services (PostgreSQL, Valkey, storage)
- Configure environment variables
- Send HTTP requests to API:
  - Upload images
  - Process images (resize, transform, etc.)
  - Retrieve processed results

### Notes
- Designed for scalability and performance
- Libvips provides significantly faster processing than traditional libraries
- Suitable for integration with frontend apps like imery
