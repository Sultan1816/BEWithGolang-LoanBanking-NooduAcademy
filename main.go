package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"noodu-login-service/internal/auth"
	"noodu-login-service/internal/config"
	"noodu-login-service/internal/customerprofile"
	"noodu-login-service/internal/customers"
	"noodu-login-service/internal/httpapi"
	"noodu-login-service/internal/imgbb"
	"noodu-login-service/internal/users"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("gagal menyiapkan database: ", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(startupContext)
	cancelStartup()
	if err != nil {
		log.Fatal("gagal terhubung ke PostgreSQL: ", err)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTTTL)
	profiles := customerprofile.NewService(customers.NewStore(db), imgbb.NewClient(cfg.ImgBBAPIKey))
	api := httpapi.NewWithCustomerProfiles(users.NewStore(db), db, tokens, cfg.CORSOrigin, profiles)
	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("login service berjalan di %s", cfg.HTTPAddress)
		serverErrors <- server.ListenAndServe()
	}()

	stopContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server berhenti: ", err)
		}
	case <-stopContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("shutdown tidak bersih: %v", err)
		}
	}
}
