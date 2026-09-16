package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type queriesStub struct {
	ownerID    string
	businessID uuid.UUID
	monthStart time.Time
	calls      int
}

func (s *queriesStub) GetDashboardMetrics(_ context.Context, ownerID string, businessID uuid.UUID, monthStart time.Time) (DashboardMetrics, error) {
	s.ownerID, s.businessID, s.monthStart = ownerID, businessID, monthStart
	s.calls++
	return DashboardMetrics{BusinessID: businessID, CurrentMonth: DashboardMonth{MonthKey: monthStart.Format("2006-01")}}, nil
}

func (s *queriesStub) GetEarningsMetrics(_ context.Context, _ string, businessID uuid.UUID, filter EarningsFilter) (EarningsMetrics, error) {
	return EarningsMetrics{
		BusinessID: businessID,
		StartDate:  filter.StartDate.Format(time.DateOnly),
		EndDate:    filter.EndDate.Format(time.DateOnly),
		StaffID:    filter.StaffID,
	}, nil
}

type updatesStub struct{ channel chan struct{} }

func (s *updatesStub) SubscribeChanges(uuid.UUID) (<-chan struct{}, func()) {
	return s.channel, func() {}
}

func TestDashboardMetricsAuthorizationScopingAndMonth(t *testing.T) {
	queries := &queriesStub{}
	service := NewService(queries, &updatesStub{channel: make(chan struct{})})
	service.now = func() time.Time {
		return time.Date(2026, time.September, 30, 23, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	}
	businessID := uuid.New()

	for _, actor := range []Actor{{}, {ID: "customer", Role: "customer"}, {ID: "admin", Role: "admin"}, {Role: "provider"}} {
		if _, err := service.GetDashboardMetrics(context.Background(), actor, businessID); !errors.Is(err, ErrForbidden) {
			t.Fatalf("actor %#v: error = %v", actor, err)
		}
	}
	if queries.calls != 0 {
		t.Fatal("unauthorized requests reached persistence")
	}
	provider := Actor{ID: "provider-a", Role: "provider"}
	if _, err := service.GetDashboardMetrics(context.Background(), provider, uuid.Nil); !errors.Is(err, ErrValidation) {
		t.Fatalf("nil business id error = %v", err)
	}
	if _, err := service.GetDashboardMetrics(context.Background(), provider, businessID); err != nil {
		t.Fatal(err)
	}
	if queries.ownerID != provider.ID || queries.businessID != businessID {
		t.Fatal("query was not scoped to the authenticated provider")
	}
	wantMonth := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	if !queries.monthStart.Equal(wantMonth) {
		t.Fatalf("month start = %v, want %v", queries.monthStart, wantMonth)
	}
}

func TestEarningsMetricsValidation(t *testing.T) {
	service := NewService(&queriesStub{}, &updatesStub{})
	provider := Actor{ID: "provider", Role: "provider"}
	businessID := uuid.New()
	start := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)

	if _, err := service.GetEarningsMetrics(context.Background(), provider, businessID, EarningsFilter{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing range error = %v", err)
	}
	if _, err := service.GetEarningsMetrics(context.Background(), provider, businessID, EarningsFilter{StartDate: start.AddDate(0, 0, 1), EndDate: start}); !errors.Is(err, ErrValidation) {
		t.Fatalf("reversed range error = %v", err)
	}
	metrics, err := service.GetEarningsMetrics(context.Background(), provider, businessID, EarningsFilter{StartDate: start, EndDate: start.AddDate(0, 0, 5)})
	if err != nil || metrics.StartDate != "2026-09-01" || metrics.EndDate != "2026-09-06" {
		t.Fatalf("metrics = %#v, error = %v", metrics, err)
	}
}
