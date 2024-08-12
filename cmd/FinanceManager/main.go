package main

import (
	"encoding/json"
	"errors"
	"fmt"
	database "github.com/sebuszqo/FinanceManager/internal/db"
	"log"
	"net/http"
	"os"
	"time"
)

type Response struct {
	Message string `json:"message"`
}

func jsonResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s in %v", r.URL.Path, time.Since(start))
	})
}

type Server struct {
	server http.Handler
}

func NewServer() *Server {
	mux := http.NewServeMux()
	dbService := database.New()
	defer dbService.Close()

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		log.Printf("GET REQUEST for ID: %s\n", id)
		fmt.Fprintf(w, "USER ID GET: %s", id)
	})

	mux.HandleFunc("POST /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		fmt.Fprintf(w, "USER ID POST: %s", id)
	})

	log.Println("Server started at :8080")
	loggedMux := loggingMiddleware(mux)
	return &Server{loggedMux}
}

func checkConfiguration() error {
	if os.Getenv("JWT_SECRET") == "" {
		return errors.New("no JWT_SECRET Provided")
	}
	return nil
}

func main() {
	if err := checkConfiguration(); err != nil {
		log.Fatalf("Missing configuration, update to start server")
	}

	srv := NewServer()
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", srv.server); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
