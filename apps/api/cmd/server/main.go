package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bankpiggy/api/internal/server"
	"bankpiggy/api/internal/store"
)

func run() error {
	config, err := server.LoadConfig()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	database, err := store.Open(ctx, config.DatabaseURL)
	if err != nil {
		cancel()
		return err
	}
	defer database.Pool.Close()
	app, err := server.New(ctx, config, database)
	cancel()
	if err != nil {
		return err
	}
	listener := &http.Server{Addr: config.ListenAddr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopped := make(chan error, 1)
	go func() {
		slog.Info("Bank Piggy listening", "address", config.ListenAddr, "mode", config.Mode)
		stopped <- listener.ListenAndServe()
	}()
	select {
	case err := <-stopped:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return listener.Shutdown(shutdown)
	}
}

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
