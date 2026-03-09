package valkey

import "time"

type ValkeyConfig struct {
	Addr     string
	Password string
	DB       int
	TTL      time.Duration
}
