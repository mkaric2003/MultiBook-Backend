package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_media/domain"
	"strings"
)

var (
	ErrNotFound   = errors.New("business not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type MediaRepository interface {
	ReplaceOwned(context.Context, string, uuid.UUID, domain.ReplaceInput) ([]domain.Media, error)
}

type Service struct{ repository MediaRepository }

func NewService(repository MediaRepository) *Service { return &Service{repository: repository} }
func (s *Service) Replace(ctx context.Context, actorID, role string, id uuid.UUID, input domain.ReplaceInput) ([]domain.Media, error) {
	if role != "provider" {
		return nil, ErrForbidden
	}
	if len(input.Items) > 9 {
		return nil, fmt.Errorf("%w: maximum is nine media items", ErrValidation)
	}
	prefix := "businesses/" + actorID + "/" + id.String() + "/"
	seen := map[domain.MediaType]bool{}
	positions := map[int16]bool{}
	for _, i := range input.Items {
		if !strings.HasPrefix(i.StoragePath, prefix) {
			return nil, fmt.Errorf("%w: storage_path does not belong to this business", ErrValidation)
		}
		if i.MediaType == domain.MediaTypeGallery {
			if i.Position == nil || *i.Position < 0 || *i.Position > 6 || positions[*i.Position] {
				return nil, fmt.Errorf("%w: invalid gallery position", ErrValidation)
			}
			positions[*i.Position] = true
		} else if (i.MediaType != domain.MediaTypeLogo && i.MediaType != domain.MediaTypeCover) || i.Position != nil || seen[i.MediaType] {
			return nil, fmt.Errorf("%w: invalid logo or cover", ErrValidation)
		} else {
			seen[i.MediaType] = true
		}
	}
	return s.repository.ReplaceOwned(ctx, actorID, id, input)
}
