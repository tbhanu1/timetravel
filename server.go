package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/temelpa/timetravel/api"
	"github.com/temelpa/timetravel/service"
)

// logError logs all non-nil errors
func logError(err error) {
	if err != nil {
		log.Printf("error: %v", err)
	}
}

func main() {
	router := mux.NewRouter()

	//TODO: Get properties from a config source
	//TODO: Setup v1 & v2 APIs to work simultaneously
	service1 := service.NewInMemoryRecordService()
	api1 := api.NewAPI(&service1)
	apiRoute := router.PathPrefix("/api/v1").Subrouter()
	apiRoute.Path("/health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		logError(err)
	})
	api1.CreateRoutes(apiRoute)

	sqliteDatabase, _ := sql.Open("sqlite3", "./records.db") // Open the created SQLite File
	defer sqliteDatabase.Close()                             // Defer Closing the database
	service2 := service.NewRepositoryRecordService(sqliteDatabase)
	api2 := api.NewAPI(&service2)
	apiRoute2 := router.PathPrefix("/api/v2").Subrouter()
	apiRoute2.Path("/health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		logError(err)
	})
	api2.CreateRoutes2(apiRoute2)

	address := "127.0.0.1:8000"
	srv := &http.Server{
		Handler:      router,
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("listening on %s", address)
	log.Fatal(srv.ListenAndServe())
}
