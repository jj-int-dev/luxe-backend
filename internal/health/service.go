package health

// Service defines the health check contract.
type Service interface {
	Health() error
}

type service struct{}

// NewService returns a new health Service.
func NewService() Service {
	return &service{}
}

func (s *service) Health() error {
	return nil
}
