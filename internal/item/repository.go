package item

import (
	"context"
	"sync"
)

// Repository defines the persistence contract for items.
type Repository interface {
	GetAll(ctx context.Context) ([]Item, error)
}

// InMemoryRepository holds a fixed seed of items in memory.
// It is intended for development; no real database is used.
type InMemoryRepository struct {
	mu    sync.RWMutex
	items []Item
}

// NewInMemoryRepository returns a repository pre-seeded with luxury items.
// Seed contains cars and houses only; jewelry has zero entries so the
// "valid category / empty result" test path is exercisable without mocks.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items: []Item{
			{
				ID:       "car-1",
				Title:    "Ferrari 488 GTB",
				Price:    299000,
				ImageURL: "https://cdn.luxe.com/cars/ferrari-488.jpg",
				Category: "cars",
			},
			{
				ID:       "car-2",
				Title:    "Lamborghini Urus",
				Price:    229000,
				ImageURL: "https://cdn.luxe.com/cars/lamborghini-urus.jpg",
				Category: "cars",
			},
			{
				ID:       "house-1",
				Title:    "Beverly Hills Mansion",
				Price:    5500000,
				ImageURL: "https://cdn.luxe.com/houses/beverly-hills.jpg",
				Category: "houses",
			},
			{
				ID:       "house-2",
				Title:    "Miami Waterfront Penthouse",
				Price:    3200000,
				ImageURL: "https://cdn.luxe.com/houses/miami-penthouse.jpg",
				Category: "houses",
			},
			{
				ID:       "house-3",
				Title:    "Malibu Beach House",
				Price:    8900000,
				ImageURL: "https://cdn.luxe.com/houses/malibu-beach.jpg",
				Category: "houses",
			},
		},
	}
}

// GetAll returns a shallow copy of all items in the store.
func (r *InMemoryRepository) GetAll(ctx context.Context) ([]Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Item, len(r.items))
	copy(out, r.items)
	return out, nil
}
