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
// There are 12 items in total (7 cars + 5 houses) to support pagination tests
// that require more than the default limit of 10.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items: []Item{
			// ── Cars ──────────────────────────────────────────────────────────
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
				ID:       "car-3",
				Title:    "Bentley Continental GT",
				Price:    230000,
				ImageURL: "https://cdn.luxe.com/cars/bentley-continental.jpg",
				Category: "cars",
			},
			{
				ID:       "car-4",
				Title:    "Rolls-Royce Ghost",
				Price:    311900,
				ImageURL: "https://cdn.luxe.com/cars/rolls-royce-ghost.jpg",
				Category: "cars",
			},
			{
				ID:       "car-5",
				Title:    "McLaren 720S",
				Price:    299000,
				ImageURL: "https://cdn.luxe.com/cars/mclaren-720s.jpg",
				Category: "cars",
			},
			{
				ID:       "car-6",
				Title:    "Bugatti Chiron",
				Price:    3000000,
				ImageURL: "https://cdn.luxe.com/cars/bugatti-chiron.jpg",
				Category: "cars",
			},
			{
				ID:       "car-7",
				Title:    "Porsche 911 Turbo S",
				Price:    207900,
				ImageURL: "https://cdn.luxe.com/cars/porsche-911-turbo-s.jpg",
				Category: "cars",
			},
			// ── Houses ────────────────────────────────────────────────────────
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
			{
				ID:       "house-4",
				Title:    "Hamptons Estate",
				Price:    12000000,
				ImageURL: "https://cdn.luxe.com/houses/hamptons-estate.jpg",
				Category: "houses",
			},
			{
				ID:       "house-5",
				Title:    "Monaco Penthouse",
				Price:    25000000,
				ImageURL: "https://cdn.luxe.com/houses/monaco-penthouse.jpg",
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
