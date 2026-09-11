package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// handleCreateTicket creates a new ticket owned by the authenticated user.
// POST /tickets  { "title": "...", "description": "..." }
func (s *Server) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r)

	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	ticket := s.store.CreateTicket(userID, req.Title, req.Description)
	writeJSON(w, http.StatusCreated, ticket)
}

// handleListTickets returns only the tickets owned by the authenticated user.
// GET /tickets
func (s *Server) handleListTickets(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r)
	tickets := s.store.ListTicketsByUser(userID)
	writeJSON(w, http.StatusOK, tickets)
}

// handleGetTicket returns a single ticket if it belongs to the caller.
// GET /tickets/{id}
func (s *Server) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r)
	id := r.PathValue("id")

	ticket, ok := s.store.GetTicket(id)
	if !ok {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	if ticket.UserID != userID {
		// Do not leak existence of another user's ticket.
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

// handleUpdateTicketStatus updates the status of a ticket owned by the caller,
// enforcing the required status flow (open -> in_progress -> closed, and a
// closed ticket can never be reopened).
// PATCH /tickets/{id}/status  { "status": "in_progress" }
func (s *Server) handleUpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r)
	id := r.PathValue("id")

	ticket, ok := s.store.GetTicket(id)
	if !ok {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	if ticket.UserID != userID {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if !isValidStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "status must be one of: open, in_progress, closed")
		return
	}

	newStatus := TicketStatus(req.Status)
	if !allowedTransition(ticket.Status, newStatus) {
		writeError(w, http.StatusConflict, "invalid status transition: "+string(ticket.Status)+" -> "+req.Status)
		return
	}

	s.store.UpdateTicketStatus(id, newStatus)
	updated, _ := s.store.GetTicket(id)
	writeJSON(w, http.StatusOK, updated)
}
