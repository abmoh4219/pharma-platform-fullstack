package handler

import (
	"time"

	"pharma-platform/internal/service"
)

// ScoreCandidate keeps compatibility for existing unit tests while
// search/match logic is implemented in the recruitment service.
func ScoreCandidate(tokens []string, fullName, email, phone, idNumber string) (int, []string) {
	candidate := service.CandidateModel{
		FullName:     fullName,
		Email:        email,
		Phone:        phone,
		IDNumber:     idNumber,
		LastActiveAt: time.Now().UTC(),
	}
	return service.ScoreCandidate(tokens, candidate)
}
