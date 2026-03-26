package interaction

import (
	"context"
	"errors"
)

// Interaction is the core domain model.
type Interaction struct {
	ItemID string
	Action string
}

// validActions is the exhaustive set of accepted action values.
var validActions = map[string]bool{
	"like": true,
	"skip": true,
}

// Typed errors for better error handling in handlers.
var (
	ErrInvalidItemID = errors.New("item_id is required")
	ErrInvalidAction = errors.New("action must be 'like' or 'skip'")
)

// Service defines the business-logic contract for interactions.
type Service interface {
	Create(ctx context.Context, itemID, action string) error
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Create validates the inputs and persists the interaction.
func (s *service) Create(ctx context.Context, itemID, action string) error {
	if itemID == "" {
		return ErrInvalidItemID
	}
	if !validActions[action] {
		return ErrInvalidAction
	}
	return s.repo.Save(ctx, Interaction{ItemID: itemID, Action: action})
}
