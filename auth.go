package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleRegister creates a new user with a hashed password.
// POST /auth/register  { "username": "...", "password": "..." }
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not process password")
		return
	}

	user, ok := s.store.CreateUser(req.Username, hash)
	if !ok {
		writeError(w, http.StatusConflict, "username already exists")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":       user.ID,
		"username": user.Username,
	})
}

// handleLogin verifies credentials and returns a signed JWT.
// POST /auth/login  { "username": "...", "password": "..." }
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, ok := s.store.GetUserByUsername(req.Username)
	if !ok || !verifyPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := createToken(user.ID, s.jwtSecret, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
