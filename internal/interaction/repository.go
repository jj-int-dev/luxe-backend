package interaction

import (
	"context"
	"sync"
)

// Repository defines the persistence contract for interactions.
type Repository interface {
	Save(ctx context.Context, i Interaction) error
}

// InMemoryRepository is a mock repository that stores interactions in memory.
// It is intended for testing and development; no real database is used.
type InMemoryRepository struct {
	mu sync.Mutex
	Interactions []Interaction
}

// NewInMemoryRepository returns a new empty InMemoryRepository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

// Save appends the interaction to the in-memory store.
func (r *InMemoryRepository) Save(ctx context.Context, i Interaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Interactions = append(r.Interactions, i)
	return nil
}
