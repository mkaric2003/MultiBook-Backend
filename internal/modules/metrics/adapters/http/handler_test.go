package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	"github.com/mkaric2003/multibook-backend/internal/modules/metrics/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

type handlerQueriesStub struct {
	metrics         application.DashboardMetrics
	earningsMetrics application.EarningsMetrics
	earningsFilter  application.EarningsFilter
}

func (s *handlerQueriesStub) GetDashboardMetrics(context.Context, string, uuid.UUID, time.Time) (application.DashboardMetrics, error) {
	return s.metrics, nil
}

func (s *handlerQueriesStub) GetEarningsMetrics(_ context.Context, _ string, _ uuid.UUID, filter application.EarningsFilter) (application.EarningsMetrics, error) {
	s.earningsFilter = filter
	return s.earningsMetrics, nil
}

func TestGetEarningsMetricsContract(t *testing.T) {
	businessID, staffID := uuid.New(), uuid.New()
	queries := &handlerQueriesStub{earningsMetrics: application.EarningsMetrics{
		BusinessID: businessID, BusinessType: domain.BusinessTypeService, Currency: "BAM",
		StartDate: "2026-09-01", EndDate: "2026-09-06", StaffID: &staffID,
		RevenueMinor: 33091, ReservationCount: 2, StaffEarningsMinor: 25000,
		DailyRevenueMinor: map[string]int64{"2026-09-06": 33091},
	}}
	handler := NewHandler(application.NewService(queries, &handlerUpdatesStub{updates: make(chan struct{})}))
	role := usersdomain.UserRoleProvider
	request := requestWithBusinessID(businessID, usersdomain.User{ID: "owner", Role: &role})
	request.URL.RawQuery = "startDate=2026-09-01&endDate=2026-09-06&utcOffsetMinutes=120&staffId=" + staffID.String()
	recorder := httptest.NewRecorder()
	handler.GetEarningsMetrics(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["revenueMinor"] != float64(33091) || body["staffEarningsMinor"] != float64(25000) || body["staffId"] != staffID.String() {
		t.Fatalf("response = %#v", body)
	}
	if queries.earningsFilter.StaffID == nil || *queries.earningsFilter.StaffID != staffID {
		t.Fatalf("filter = %#v", queries.earningsFilter)
	}
	if queries.earningsFilter.UTCOffsetMinutes != 120 || queries.earningsFilter.StartDate.UTC().Format(time.RFC3339) != "2026-08-31T22:00:00Z" {
		t.Fatalf("timezone filter = %#v", queries.earningsFilter)
	}
}

func TestGetEarningsMetricsRejectsInvalidUTCOffset(t *testing.T) {
	handler := NewHandler(application.NewService(&handlerQueriesStub{}, &handlerUpdatesStub{updates: make(chan struct{})}))
	role := usersdomain.UserRoleProvider
	request := requestWithBusinessID(uuid.New(), usersdomain.User{ID: "owner", Role: &role})
	request.URL.RawQuery = "startDate=2026-09-01&endDate=2026-09-06&utcOffsetMinutes=900"
	recorder := httptest.NewRecorder()
	handler.GetEarningsMetrics(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestGetEarningsMetricsRequiresDateRange(t *testing.T) {
	handler := NewHandler(application.NewService(&handlerQueriesStub{}, &handlerUpdatesStub{updates: make(chan struct{})}))
	role := usersdomain.UserRoleProvider
	recorder := httptest.NewRecorder()
	handler.GetEarningsMetrics(recorder, requestWithBusinessID(uuid.New(), usersdomain.User{ID: "owner", Role: &role}))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", recorder.Code)
	}
}

type handlerUpdatesStub struct{ updates chan struct{} }

func (s *handlerUpdatesStub) SubscribeChanges(uuid.UUID) (<-chan struct{}, func()) {
	return s.updates, func() {}
}

func TestGetDashboardMetricsContract(t *testing.T) {
	businessID := uuid.New()
	queries := &handlerQueriesStub{metrics: application.DashboardMetrics{
		BusinessID: businessID, BusinessType: domain.BusinessTypeStay, Currency: "BAM", ActiveReservationCount: 3,
		CurrentMonth: application.DashboardMonth{
			MonthKey: "2026-09", RevenueMinor: 12500, ReservationCount: 2,
			DailyRevenueMinor: map[string]int64{"2026-09-06": 12500}, DailyReservationCount: map[string]int{"2026-09-06": 2},
		},
	}}
	handler := NewHandler(application.NewService(queries, &handlerUpdatesStub{updates: make(chan struct{})}))
	role := usersdomain.UserRoleProvider
	request := requestWithBusinessID(businessID, usersdomain.User{ID: "owner", Role: &role})
	recorder := httptest.NewRecorder()
	handler.GetDashboardMetrics(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	month := body["currentMonth"].(map[string]any)
	if body["businessType"] != "stay" || body["activeReservationCount"] != float64(3) || month["revenueMinor"] != float64(12500) {
		t.Fatalf("response = %#v", body)
	}
}

func TestDashboardMetricsRequireProvider(t *testing.T) {
	handler := NewHandler(application.NewService(&handlerQueriesStub{}, &handlerUpdatesStub{updates: make(chan struct{})}))
	role := usersdomain.UserRoleCustomer
	recorder := httptest.NewRecorder()
	handler.GetDashboardMetrics(recorder, requestWithBusinessID(uuid.New(), usersdomain.User{ID: "customer", Role: &role}))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func requestWithBusinessID(id uuid.UUID, user usersdomain.User) *http.Request {
	request := usershttp.WithCurrentUser(httptest.NewRequest(http.MethodGet, "/businesses/"+id.String()+"/dashboard-metrics", nil), user)
	route := chi.NewRouteContext()
	route.URLParams.Add("businessID", id.String())
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
}
