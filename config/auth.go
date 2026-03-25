package config

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"time"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
)

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

func loadJWTSigningKeyConfig() domain.JWTSigningKeyConfig {
	privateKey, _ := loadECPrivateKey("ec256_private.pem")
	publicKey, _ := loadECPublicKey("ec256_public.pem")

	return domain.JWTSigningKeyConfig{
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		PrivateKey:      privateKey,
		PublicKey:       publicKey,
	}
}
