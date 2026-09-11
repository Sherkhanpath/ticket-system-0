package main

import "time"

// User represents a registered user of the system.
// PasswordHash is never sent back in any API response.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

// TicketStatus is a restricted set of allowed ticket states.
type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusClosed     TicketStatus = "closed"
)

// isValidStatus checks whether a string is one of the allowed statuses.
func isValidStatus(s string) bool {
	switch TicketStatus(s) {
	case StatusOpen, StatusInProgress, StatusClosed:
		return true
	}
	return false
}

// allowedTransition enforces the required status flow:
// open -> in_progress -> closed, and closed can never move back.
func allowedTransition(from, to TicketStatus) bool {
	switch from {
	case StatusOpen:
		return to == StatusInProgress || to == StatusOpen
	case StatusInProgress:
		return to == StatusClosed || to == StatusInProgress
	case StatusClosed:
		return false // closed tickets can never change again
	}
	return false
}

// Ticket represents a single support ticket owned by exactly one user.
type Ticket struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
