package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/creafly/storage/internal/app"
	"github.com/creafly/storage/internal/logger"
	_ "github.com/lib/pq"
	"github.com/xlab/closer"
)

func main() {
	ctx := context.Background()

	migrateUp := flag.Bool("migrate-up", false, "Run database migrations up")
	migrateDown := flag.Bool("migrate-down", false, "Run database migrations down")
	flag.Parse()

	app := app.NewApp()
	app.StartMigrator(*migrateUp, *migrateDown)
	app.StartApp(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down...")
	closer.Close()
	logger.Info().Msg("Server exited")
}
