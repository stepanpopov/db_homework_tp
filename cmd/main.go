package main

import (
	"fmt"
	"log"
	"net/http"


	db "github.com/stepanpopov/db_homework_tp/cmd/init/db"
	router "github.com/stepanpopov/db_homework_tp/cmd/init/router"
)

const (
	endpoint = ":5000"
	// maxHeaderBytesHTTP = 1 << 20
	// readTimeoutHTTP  = 10 * time.Second
	// writeTimeoutHTTP = 10 * time.Second
)

func main() {
	db, err := db.InitPostgresDB()
	if err != nil {
		fmt.Printf("error while connecting to database: %v\n", err)
		return
	}

	router := router.Init(db)

	httpServer := &http.Server{
		Handler: router,
		Addr:    endpoint,
		// MaxHeaderBytes: maxHeaderBytesHTTP,
		// ReadTimeout:  readTimeoutHTTP,
		// WriteTimeout: writeTimeoutHTTP,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("Unable to start http server: %v \n", err)
	}
}
