package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/domain"
)

func TestCreateWeeklyRejectsInvalidInterval(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	_, err := NewService(repository, repository).CreateWeekly(context.Background(), "provider", "provider", uuid.New(), uuid.New(), domain.CreateWeeklyInput{Weekday: 1, StartMinutes: 720, EndMinutes: 720})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
func TestUpdateWeeklyRequiresAChange(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	_, err := NewService(repository, repository).UpdateWeekly(context.Background(), "provider", "provider", uuid.New(), uuid.New(), uuid.New(), domain.UpdateWeeklyInput{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
func TestCreateBlockRejectsInvertedRange(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)
	repository := &fakeRepository{}
	_, err := NewService(repository, repository).CreateBlock(context.Background(), "provider", "provider", uuid.New(), uuid.New(), domain.CreateBlockInput{StartAt: start, EndAt: start.Add(-time.Minute)})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
func TestCreateBlockTrimsReason(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	start := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	reason := "  annual leave  "
	_, err := NewService(repository, repository).CreateBlock(context.Background(), "provider", "provider", uuid.New(), uuid.New(), domain.CreateBlockInput{StartAt: start, EndAt: start.Add(time.Hour), Reason: &reason})
	if err != nil {
		t.Fatal(err)
	}
	if repository.block.Reason == nil || *repository.block.Reason != "annual leave" {
		t.Fatalf("reason = %#v", repository.block.Reason)
	}
}
func TestAvailabilityRequiresProviderRole(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	_, err := NewService(repository, repository).ListWeekly(context.Background(), "customer", "customer", uuid.New(), uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestOperationsUseDedicatedPorts(t *testing.T) {
	t.Parallel()
	commands := &availabilityCommandsSpy{}
	queries := &availabilityQueriesSpy{}
	service := NewService(commands, queries)
	businessID, staffID, weeklyID, blockID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := service.ListWeekly(context.Background(), "provider", "provider", businessID, staffID); err != nil {
		t.Fatal(err)
	}
	if !queries.weeklyListed || queries.blocksListed {
		t.Fatal("ListWeekly() must use the query port")
	}
	if _, err := service.ListBlocks(context.Background(), "provider", "provider", businessID, staffID); err != nil {
		t.Fatal(err)
	}
	if !queries.blocksListed {
		t.Fatal("ListBlocks() must use the query port")
	}
	if _, err := service.CreateWeekly(context.Background(), "provider", "provider", businessID, staffID, domain.CreateWeeklyInput{Weekday: 1, StartMinutes: 540, EndMinutes: 600}); err != nil {
		t.Fatal(err)
	}
	weekday, start, end := int16(2), int16(600), int16(660)
	if _, err := service.UpdateWeekly(context.Background(), "provider", "provider", businessID, staffID, weeklyID, domain.UpdateWeeklyInput{Weekday: &weekday, StartMinutes: &start, EndMinutes: &end}); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteWeekly(context.Background(), "provider", "provider", businessID, staffID, weeklyID); err != nil {
		t.Fatal(err)
	}
	blockStart := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	blockEnd := blockStart.Add(time.Hour)
	if _, err := service.CreateBlock(context.Background(), "provider", "provider", businessID, staffID, domain.CreateBlockInput{StartAt: blockStart, EndAt: blockEnd}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateBlock(context.Background(), "provider", "provider", businessID, staffID, blockID, domain.UpdateBlockInput{StartAt: &blockStart, EndAt: &blockEnd}); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteBlock(context.Background(), "provider", "provider", businessID, staffID, blockID); err != nil {
		t.Fatal(err)
	}
	if !commands.weeklyCreated || !commands.weeklyUpdated || !commands.weeklyDeleted || !commands.blockCreated || !commands.blockUpdated || !commands.blockDeleted {
		t.Fatal("availability mutations must use the command repository")
	}
}

type fakeRepository struct {
	block         domain.CreateBlockInput
	weeklyCreated bool
}

func (r *fakeRepository) CreateWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.CreateWeeklyInput) (domain.WeeklyAvailability, error) {
	r.weeklyCreated = true
	return domain.WeeklyAvailability{}, nil
}
func (*fakeRepository) ListWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID) ([]domain.WeeklyAvailability, error) {
	return nil, nil
}
func (*fakeRepository) UpdateWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdateWeeklyInput) (domain.WeeklyAvailability, error) {
	return domain.WeeklyAvailability{}, nil
}
func (*fakeRepository) DeleteWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (r *fakeRepository) CreateBlockOwned(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID, input domain.CreateBlockInput) (domain.AvailabilityBlock, error) {
	r.block = input
	return domain.AvailabilityBlock{}, nil
}
func (*fakeRepository) ListBlocksOwned(context.Context, string, uuid.UUID, uuid.UUID) ([]domain.AvailabilityBlock, error) {
	return nil, nil
}
func (*fakeRepository) UpdateBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdateBlockInput) (domain.AvailabilityBlock, error) {
	return domain.AvailabilityBlock{}, nil
}
func (*fakeRepository) DeleteBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}

type availabilityCommandsSpy struct {
	weeklyCreated bool
	weeklyUpdated bool
	weeklyDeleted bool
	blockCreated  bool
	blockUpdated  bool
	blockDeleted  bool
}

func (r *availabilityCommandsSpy) CreateWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.CreateWeeklyInput) (domain.WeeklyAvailability, error) {
	r.weeklyCreated = true
	return domain.WeeklyAvailability{}, nil
}
func (r *availabilityCommandsSpy) UpdateWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdateWeeklyInput) (domain.WeeklyAvailability, error) {
	r.weeklyUpdated = true
	return domain.WeeklyAvailability{}, nil
}
func (r *availabilityCommandsSpy) DeleteWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID) error {
	r.weeklyDeleted = true
	return nil
}
func (r *availabilityCommandsSpy) CreateBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.CreateBlockInput) (domain.AvailabilityBlock, error) {
	r.blockCreated = true
	return domain.AvailabilityBlock{}, nil
}
func (r *availabilityCommandsSpy) UpdateBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdateBlockInput) (domain.AvailabilityBlock, error) {
	r.blockUpdated = true
	return domain.AvailabilityBlock{}, nil
}
func (r *availabilityCommandsSpy) DeleteBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID) error {
	r.blockDeleted = true
	return nil
}

type availabilityQueriesSpy struct {
	weeklyListed bool
	blocksListed bool
}

func (q *availabilityQueriesSpy) ListWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID) ([]domain.WeeklyAvailability, error) {
	q.weeklyListed = true
	return nil, nil
}

func (q *availabilityQueriesSpy) ListBlocksOwned(context.Context, string, uuid.UUID, uuid.UUID) ([]domain.AvailabilityBlock, error) {
	q.blocksListed = true
	return nil, nil
}
