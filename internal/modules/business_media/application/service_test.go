package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_media/domain"
)

func TestReplaceUsesCommandRepository(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	businessID := uuid.New()
	_, err := NewService(repository).Replace(context.Background(), "provider", "provider", businessID, domain.ReplaceInput{Items: []domain.Input{{MediaType: domain.MediaTypeLogo, StoragePath: "businesses/provider/" + businessID.String() + "/logo.png"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.replaced {
		t.Fatal("replace should use command repository")
	}
}

type fakeRepository struct{ replaced bool }

func (r *fakeRepository) ReplaceOwned(context.Context, string, uuid.UUID, domain.ReplaceInput) ([]domain.Media, error) {
	r.replaced = true
	return nil, nil
}
