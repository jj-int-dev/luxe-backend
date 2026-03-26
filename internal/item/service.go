package item

import "context"

// Service defines the business-logic contract for items.
type Service interface {
	GetItems(ctx context.Context, category *string) ([]Item, error)
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// GetItems returns all items, optionally filtered by category.
// If category is non-nil and not a recognised value, ErrInvalidCategory
// is returned. A nil category means "no filter".
func (s *service) GetItems(ctx context.Context, category *string) ([]Item, error) {
	if category != nil && !IsValidCategory(*category) {
		return nil, ErrInvalidCategory
	}

	all, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	if category == nil {
		return all, nil
	}

	filtered := make([]Item, 0)
	for _, it := range all {
		if it.Category == *category {
			filtered = append(filtered, it)
		}
	}
	return filtered, nil
}
