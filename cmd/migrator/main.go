package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/creafly/logger"
	"github.com/creafly/storage/internal/app"
	"github.com/xlab/closer"
)

func main() {
	defer logger.Shutdown()

	migrateUp := flag.Bool("migrate-up", false, "Run database migrations up")
	migrateDown := flag.Bool("migrate-down", false, "Run database migrations down")
	flag.Parse()

	app := app.NewApp()
	app.StartMigrator(*migrateUp, *migrateDown)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down...")
	closer.Close()
	logger.Info().Msg("Server exited")
}
