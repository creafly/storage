package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	MinIO    MinIOConfig
	Upload   UploadConfig
	Identity IdentityConfig
	CORS     CORSConfig
	Tracing  TracingConfig
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
	corsOrigins := getEnv("CORS_ALLOWED_ORIGINS", "")
	ginMode := getEnv("GIN_MODE", "debug")

	return &Config{
		Server: ServerConfig{
			Host:    getEnv("SERVER_HOST", "0.0.0.0"),
			Port:    getEnv("SERVER_PORT", "8083"),
			GinMode: ginMode,
		},
		Database: DatabaseConfig{
			URL:         getEnvRequired("DATABASE_URL"),
			AutoMigrate: getEnvBool("AUTO_MIGRATE", true),
		},
		MinIO: MinIOConfig{
			Endpoint:   getEnvRequired("MINIO_ENDPOINT"),
			AccessKey:  getEnvRequired("MINIO_ACCESS_KEY"),
			SecretKey:  getEnvRequired("MINIO_SECRET_KEY"),
			UseSSL:     getEnvBool("MINIO_USE_SSL", false),
			BucketName: getEnv("MINIO_BUCKET_NAME", "hexaend-storage"),
		},
		Upload: UploadConfig{
			MaxFileSize:          getEnvInt64("MAX_FILE_SIZE", 10*1024*1024),
			AllowedImageTypes:    splitNonEmpty(getEnv("ALLOWED_IMAGE_TYPES", "image/png,image/svg+xml,image/jpeg,image/webp,image/gif"), ","),
			AllowedDocumentTypes: splitNonEmpty(getEnv("ALLOWED_DOCUMENT_TYPES", "application/pdf"), ","),
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
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
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
