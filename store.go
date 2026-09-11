package main

import (
	"sync"
)

// Store is a simple in-memory, thread-safe data store.
// The assignment allows in-memory storage, so we keep this simple
// on purpose instead of wiring up a database.
type Store struct {
	mu           sync.RWMutex
	usersByID    map[string]*User
	usersByName  map[string]*User // for fast login lookup by username
	tickets      map[string]*Ticket
	nextUserID   int
	nextTicketID int
}

// NewStore creates an empty, ready-to-use store.
func NewStore() *Store {
	return &Store{
		usersByID:   make(map[string]*User),
		usersByName: make(map[string]*User),
		tickets:     make(map[string]*Ticket),
	}
}

// CreateUser adds a new user. Returns false if the username is taken.
func (s *Store) CreateUser(username, passwordHash string) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usersByName[username]; exists {
		return nil, false
	}

	s.nextUserID++
	user := &User{
		ID:           idFromCounter("usr", s.nextUserID),
		Username:     username,
		PasswordHash: passwordHash,
	}
	s.usersByID[user.ID] = user
	s.usersByName[username] = user
	return user, true
}

// GetUserByUsername looks up a user by their username.
func (s *Store) GetUserByUsername(username string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.usersByName[username]
	return u, ok
}

// CreateTicket adds a new ticket owned by userID.
func (s *Store) CreateTicket(userID, title, description string) *Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextTicketID++
	now := nowUTC()
	t := &Ticket{
		ID:          idFromCounter("tkt", s.nextTicketID),
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      StatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tickets[t.ID] = t
	return t
}

// ListTicketsByUser returns all tickets owned by userID.
func (s *Store) ListTicketsByUser(userID string) []*Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []*Ticket{}
	for _, t := range s.tickets {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result
}

// GetTicket returns a ticket by ID regardless of owner (ownership is
// checked by the caller/handler).
func (s *Store) GetTicket(id string) (*Ticket, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tickets[id]
	return t, ok
}

// UpdateTicketStatus overwrites the status/updated_at of an existing ticket.
func (s *Store) UpdateTicketStatus(id string, status TicketStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tickets[id]; ok {
		t.Status = status
		t.UpdatedAt = nowUTC()
	}
}
