package config

import (
	"log"
	"time"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/oauth"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/pg"
	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/valkey"
)

// =========== Postgres ===========
var (
	pgc *PostgresCloser
)

var (
	UserRepository         domain.UserRepository
	UserIdentityRepository domain.UserIdentityRepository

	ImageMetadataRepository domain.ImageMetadataRepository
)

func InitializePostgres() {
	pgc = newPostgresCloser(loadPostgresConfig())

	UserRepository = createPostgres(pgc, pg.NewPostgresUserRepository)
	UserIdentityRepository = createPostgres(pgc, pg.NewPostgresUserIdentityRepository)
	ImageMetadataRepository = createPostgres(pgc, pg.NewPostgresImageRepository)
}

func ClosePostgres() {
	pgc.Close()
}

// =========== Storage ===========
var (
	ImageStorageRepository domain.ImageStorageRepository
)

func InitializeR2() {
	var err error
	ImageStorageRepository, err = infrastructure.NewR2Storage(loadR2Config())
	if err != nil {
		log.Fatalf("failed to initialize R2 storage: %v", err)
	}
}

// =========== Valkey ===========
var (
	ImageStore   domain.ImageStore
	ImageStoreDB int = 0

	SessionStore   domain.SessionStore
	SessionStoreDB int = 1

	OAuthStateStore   domain.OAuthStateStore
	OAuthStateStoreDB int = 2
)

func InitializeStore() {
	ImageStore = createValkey(loadValkeyConfig(ImageStoreDB, 30*time.Minute), valkey.NewValkeyImageStore)
	SessionStore = createValkey(loadValkeyConfig(SessionStoreDB, time.Hour), valkey.NewValkeySessionStore)
	OAuthStateStore = createValkey(loadValkeyConfig(OAuthStateStoreDB, 5*time.Minute), valkey.NewValkeyOAuthStateStore)
}

// =========== JWT Keys ===========
var (
	JWTSigningKeyConfig domain.JWTSigningKeyConfig
)

func InitializeSigningKey() {
	JWTSigningKeyConfig = loadJWTSigningKeyConfig()
}

// =========== OAuth ===========
var (
	GoogleOAuthRepository domain.OAuthRepository
	GithubOAuthRepository domain.OAuthRepository
)

func InitializeOAuthRepository() {
	GoogleOAuthRepository = createOAuth(loadGoogleOAuthConfig(), oauth.NewGoogleOAuthRepository)
	GithubOAuthRepository = createOAuth(loadGithubOAuthConfig(), oauth.NewGithubOAuthRepository)
}
