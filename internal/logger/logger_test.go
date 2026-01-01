package logger

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInit(t *testing.T) {
	Init("test-service")

	if Log.GetLevel() == 0 && os.Getenv("GIN_MODE") != "release" {
		t.Log("Logger initialized successfully")
	}
}

func TestWithRequestID(t *testing.T) {
	Init("test-service")

	requestID := "test-request-123"
	logger := WithRequestID(requestID)

	logger.Info().Msg("test message")
}

func TestFromContext(t *testing.T) {
	Init("test-service")

	c := &gin.Context{}
	logger := FromContext(c)

	logger.Info().Msg("test message")
}

func TestFromContextWithRequestID(t *testing.T) {
	Init("test-service")

	c := &gin.Context{}
	c.Set("request_id", "test-123")

	logger := FromContext(c)

	logger.Info().Msg("test message")
}

func TestLoggerFunctions(t *testing.T) {
	Init("test-service")

	Info().Msg("test info")
	Debug().Msg("test debug")
	Warn().Msg("test warn")
	Error().Msg("test error")
}
