package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/basecore-dev/basecore/internal/api"
	"github.com/basecore-dev/basecore/internal/config"
	"github.com/basecore-dev/basecore/internal/store"
)

func main() {

	apiPort := flag.Int("port", 8080, "port for start server")

	setJWTSecretFlag := flag.Bool("jwt-secret", false, "Set JWT Secret")
	flag.Parse()

	if *setJWTSecretFlag {
		var jwtSecret string
		fmt.Print("Enter your new jwt secret: ")
		if _, err := fmt.Scan(&jwtSecret); err != nil {
			log.Fatalf("read JWT secret: %v", err)
		}

		if err := config.SetJWTSecret(jwtSecret); err != nil {
			log.Fatal(err)
		}
	}

	if err := config.Init(); err != nil {
		log.Fatal(err)
	}

	if err := config.CheckJWTSecret(); err != nil {
		log.Fatal(err)
	}

	databasePath := filepath.Join(config.GetBaseCorePath(), "basecore.db")
	if err := store.ConnectDB(databasePath); err != nil {
		log.Fatalf("connect database: %v", err)
	}

	defer func() {
		if err := store.CloseDB(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := store.CreateTables(); err != nil {
		log.Printf("initialize database schema: %v", err)
	}

	apiServer := &http.Server{
		Addr: fmt.Sprintf(
			"0.0.0.0:%d", *apiPort,
		),

		Handler: api.Router(),
	}

	log.Printf("Server running on %d", *apiPort)

	if err := apiServer.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Printf("server error: %v", err)
	}
}
