package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
)

type stub struct {
	created domain.CreateInput
	items   []domain.Review
	err     error
}

func (s *stub) Create(_ context.Context, _ string, _ uuid.UUID, input domain.CreateInput) (domain.Review, error) {
	s.created = input
	return domain.Review{ID: uuid.New()}, s.err
}
func (s *stub) HasReview(context.Context, string, uuid.UUID) (bool, error) { return true, s.err }
func (s *stub) List(context.Context, uuid.UUID, int, int) ([]domain.Review, error) {
	return s.items, s.err
}

func TestCreateValidatesAndNormalizes(t *testing.T) {
	repository := &stub{}
	service := NewService(repository, repository)
	comment := "  useful  "
	_, err := service.Create(context.Background(), Actor{ID: "customer", Role: "customer"}, uuid.New(), domain.CreateInput{
		SourceID: uuid.New(), Type: domain.SourceTypeStay, Rating: 5, Comment: &comment,
	})
	if err != nil || repository.created.Comment == nil || *repository.created.Comment != "useful" {
		t.Fatalf("Create() err=%v input=%#v", err, repository.created)
	}

	long := strings.Repeat("š", 1001)
	_, _ = service.Create(context.Background(), Actor{ID: "customer", Role: "customer"}, uuid.New(), domain.CreateInput{
		SourceID: uuid.New(), Type: domain.SourceTypeService, Rating: 1, Comment: &long,
	})
	if got := len([]rune(*repository.created.Comment)); got != 1000 {
		t.Fatalf("comment length = %d", got)
	}
}

func TestCreateRejectsInvalidOrProvider(t *testing.T) {
	repository := &stub{}
	service := NewService(repository, repository)
	valid := domain.CreateInput{SourceID: uuid.New(), Type: domain.SourceTypeStay, Rating: 5}
	if _, err := service.Create(context.Background(), Actor{ID: "provider", Role: "provider"}, uuid.New(), valid); !errors.Is(err, ErrForbidden) {
		t.Fatalf("provider error = %v", err)
	}
	valid.Rating = 0
	if _, err := service.Create(context.Background(), Actor{ID: "customer", Role: "customer"}, uuid.New(), valid); !errors.Is(err, ErrValidation) {
		t.Fatalf("rating error = %v", err)
	}
}

func TestListPaginates(t *testing.T) {
	repository := &stub{items: make([]domain.Review, 5)}
	page, err := NewService(repository, repository).List(context.Background(), Actor{ID: "user"}, uuid.New(), 4, 8)
	if err != nil || len(page.Items) != 4 || page.NextOffset == nil || *page.NextOffset != 12 {
		t.Fatalf("List() = %#v, %v", page, err)
	}
}
