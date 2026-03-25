package domain

import (
	"crypto/ecdsa"
	"time"
)

type JWTSigningKeyConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration // 15 min
	RefreshTokenTTL time.Duration // 7 days

	PublicKey  *ecdsa.PublicKey
	PrivateKey *ecdsa.PrivateKey
}
