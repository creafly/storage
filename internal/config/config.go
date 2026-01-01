package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/creafly/vault"
	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	MinIO     MinIOConfig
	Upload    UploadConfig
	Identity  IdentityConfig
	CORS      CORSConfig
	Tracing   TracingConfig
	RateLimit RateLimitConfig
}

type RateLimitConfig struct {
	Enabled           bool
	RequestsPerSecond float64
	BurstSize         int
}

type ServerConfig struct {
	Host    string
	Port    string
	GinMode string
}

type DatabaseConfig struct {
	URL         string
	AutoMigrate bool
}

type MinIOConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	UseSSL     bool
	BucketName string
}

type UploadConfig struct {
	MaxFileSize          int64
	AllowedImageTypes    []string
	AllowedDocumentTypes []string
	AllowedVideoTypes    []string
}

type IdentityConfig struct {
	ServiceURL string
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type TracingConfig struct {
	Enabled        bool
	OTLPEndpoint   string
	ServiceName    string
	ServiceVersion string
	Environment    string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	secrets := vault.NewSecretLoaderFromEnv("storage")

	corsOrigins := getEnv("CORS_ALLOWED_ORIGINS", "")
	ginMode := getEnv("GIN_MODE", "debug")

	databaseURL := buildDatabaseURL(secrets)

	minioAccessKey := secrets.GetSecret("minio_access_key", "MINIO_ACCESS_KEY", "")
	if minioAccessKey == "" {
		log.Fatal("MINIO_ACCESS_KEY is required")
	}

	minioSecretKey := secrets.GetSecret("minio_secret_key", "MINIO_SECRET_KEY", "")
	if minioSecretKey == "" {
		log.Fatal("MINIO_SECRET_KEY is required")
	}

	return &Config{
		Server: ServerConfig{
			Host:    getEnv("SERVER_HOST", "0.0.0.0"),
			Port:    getEnv("SERVER_PORT", "8083"),
			GinMode: ginMode,
		},
		Database: DatabaseConfig{
			URL:         databaseURL,
			AutoMigrate: getEnvBool("AUTO_MIGRATE", true),
		},
		MinIO: MinIOConfig{
			Endpoint:   getEnv("MINIO_ENDPOINT", "minio:9000"),
			AccessKey:  minioAccessKey,
			SecretKey:  minioSecretKey,
			UseSSL:     getEnvBool("MINIO_USE_SSL", false),
			BucketName: getEnv("MINIO_BUCKET_NAME", "creafly-storage"),
		},
		Upload: UploadConfig{
			MaxFileSize:          getEnvInt64("MAX_FILE_SIZE", 10*1024*1024),
			AllowedImageTypes:    splitNonEmpty(getEnv("ALLOWED_IMAGE_TYPES", "image/png,image/svg+xml,image/jpeg,image/webp,image/gif"), ","),
			AllowedDocumentTypes: splitNonEmpty(getEnv("ALLOWED_DOCUMENT_TYPES", "application/pdf,text/plain,text/markdown"), ","),
			AllowedVideoTypes:    splitNonEmpty(getEnv("ALLOWED_VIDEO_TYPES", "video/mp4,video/webm,video/quicktime"), ","),
		},
		Identity: IdentityConfig{
			ServiceURL: getEnv("IDENTITY_SERVICE_URL", "http://localhost:8080"),
		},
		CORS: CORSConfig{
			AllowedOrigins:   splitNonEmpty(corsOrigins, ","),
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowedHeaders:   []string{"Origin", "Content-Type", "Authorization", "Accept-Language", "X-Service-Name", "X-Tenant-ID"},
			AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           86400,
		},
		Tracing: TracingConfig{
			Enabled:        getEnvBool("TRACING_ENABLED", false),
			OTLPEndpoint:   getEnv("OTLP_ENDPOINT", "localhost:4317"),
			ServiceName:    "storage",
			ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
			Environment:    getEnv("ENVIRONMENT", "development"),
		},
		RateLimit: RateLimitConfig{
			Enabled:           getEnvBool("RATE_LIMIT_ENABLED", true),
			RequestsPerSecond: getEnvFloat("RATE_LIMIT_RPS", 100),
			BurstSize:         getEnvInt("RATE_LIMIT_BURST", 200),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return b
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return defaultValue
		}
		return i
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return i
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return defaultValue
		}
		return f
	}
	return defaultValue
}

func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func buildDatabaseURL(secrets *vault.SecretLoader) string {
	host := getEnv("DATABASE_HOST", "localhost")
	port := getEnv("DATABASE_PORT", "5432")
	name := getEnv("DATABASE_NAME", "storage")
	user := getEnv("DATABASE_USER", "postgres")
	sslMode := getEnv("DATABASE_SSL_MODE", "disable")

	password := secrets.GetSecret("database_password", "DATABASE_PASSWORD", "")
	if password == "" {
		log.Fatal("DATABASE_PASSWORD is required (from Vault or environment)")
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user,
		url.QueryEscape(password),
		host,
		port,
		name,
		sslMode,
	)
}
