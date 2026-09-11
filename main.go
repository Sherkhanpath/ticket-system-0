package main

import (
	"embed"
	"log"
	"net/http"
	"os"
)

//go:embed static/index.html
var staticFiles embed.FS

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "frontend not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// Server holds shared dependencies used by all handlers.
type Server struct {
	store     *Store
	jwtSecret []byte
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// A sane default so the service still runs locally without any
		// .env file, but production deployments should always set
		// JWT_SECRET explicitly (see .env.example).
		secret = "dev-secret-change-me"
		log.Println("WARNING: JWT_SECRET not set, using an insecure default. Set JWT_SECRET in production.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s := &Server{
		store:     NewStore(),
		jwtSecret: []byte(secret),
	}

	mux := http.NewServeMux()

	// Frontend (simple login + ticket dashboard, served from the same binary)
	mux.HandleFunc("GET /{$}", handleIndex)

	// Public routes
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /auth/register", s.handleRegister)
	mux.HandleFunc("POST /auth/login", s.handleLogin)

	// Protected routes (require Authorization: Bearer <token>)
	mux.HandleFunc("POST /tickets", s.requireAuth(s.handleCreateTicket))
	mux.HandleFunc("GET /tickets", s.requireAuth(s.handleListTickets))
	mux.HandleFunc("GET /tickets/{id}", s.requireAuth(s.handleGetTicket))
	mux.HandleFunc("PATCH /tickets/{id}/status", s.requireAuth(s.handleUpdateTicketStatus))

	addr := ":" + port
	log.Printf("ticket-system listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}