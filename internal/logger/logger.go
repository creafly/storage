package logger

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func Init(serviceName string) {
	zerolog.TimeFieldFormat = time.RFC3339

	Log = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	if os.Getenv("GIN_MODE") == "release" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func WithRequestID(requestID string) zerolog.Logger {
	return Log.With().Str("requestId", requestID).Logger()
}

func FromContext(c *gin.Context) zerolog.Logger {
	if requestID, exists := c.Get("request_id"); exists {
		return WithRequestID(requestID.(string))
	}
	return Log
}

func Info() *zerolog.Event {
	return Log.Info()
}

func Error() *zerolog.Event {
	return Log.Error()
}

func Debug() *zerolog.Event {
	return Log.Debug()
}

func Warn() *zerolog.Event {
	return Log.Warn()
}

func Fatal() *zerolog.Event {
	return Log.Fatal()
}
