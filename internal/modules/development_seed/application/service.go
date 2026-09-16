package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrForbidden = errors.New("forbidden")

type Repository interface {
	SeedStays(context.Context, string) (int, error)
	SeedServices(context.Context, string) (int, error)
}
type Service struct {
	repository Repository
	enabled    bool
}

func NewService(r Repository, appEnv string) *Service {
	return &Service{r, strings.EqualFold(appEnv, "development")}
}
func (s *Service) Stays(ctx context.Context, actor, role string) (int, error) {
	if !s.enabled || role != "provider" || strings.TrimSpace(actor) == "" {
		return 0, fmt.Errorf("%w: development provider access is required", ErrForbidden)
	}
	return s.repository.SeedStays(ctx, actor)
}
func (s *Service) Services(ctx context.Context, actor, role string) (int, error) {
	if !s.enabled || role != "provider" || strings.TrimSpace(actor) == "" {
		return 0, fmt.Errorf("%w: development provider access is required", ErrForbidden)
	}
	return s.repository.SeedServices(ctx, actor)
}
