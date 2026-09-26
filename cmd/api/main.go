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

	"agnos-assignment/internal/config"
	"agnos-assignment/internal/database"
	"agnos-assignment/internal/his"
	"agnos-assignment/internal/repository"
	"agnos-assignment/internal/router"
	"agnos-assignment/internal/security"
	"agnos-assignment/internal/service"
)

func main() {
	cfg := config.Load()

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()

	databasePool, err := database.Connect(startupContext, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer databasePool.Close()

	hospitalRepository := repository.NewHospitalRepository(databasePool)
	staffRepository := repository.NewStaffRepository(databasePool)
	patientRepository := repository.NewPatientRepository(databasePool)
	passwordHasher := security.NewBcryptHasher()
	jwtManager := security.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry)
	staffService := service.NewStaffService(hospitalRepository, staffRepository, passwordHasher, jwtManager)
	hisRegistry := his.NewRegistry(map[string]his.Client{
		"hospital-a": his.NewHospitalAClient(&http.Client{Timeout: 5 * time.Second}),
	})
	patientService := service.NewPatientService(hospitalRepository, patientRepository, hisRegistry)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(databasePool, staffService, patientService, jwtManager),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("API server started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("API server failed", "error", err)
			os.Exit(1)
		}
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-shutdownSignal.Done()

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("API server stopped")
}
