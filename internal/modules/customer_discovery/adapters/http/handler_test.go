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
	"github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

type httpQueriesStub struct {
	popular         []application.CustomerBusinessSummary
	popularType     domain.BusinessType
	popularCity     string
	popularLimit    int
	popularOffset   int
	cities          []string
	search          []application.CustomerBusinessSummary
	searchQuery     string
	recommended     []application.RecommendedStay
	recommendedCity string
}

func (*httpQueriesStub) GetActiveDetail(context.Context, uuid.UUID) (application.CustomerBusinessDetail, error) {
	return application.CustomerBusinessDetail{}, nil
}
func (q *httpQueriesStub) ListPopular(_ context.Context, kind domain.BusinessType, city string, limit, offset int) ([]application.CustomerBusinessSummary, error) {
	q.popularType, q.popularCity, q.popularLimit, q.popularOffset = kind, city, limit, offset
	return q.popular, nil
}
func (q *httpQueriesStub) ListCities(context.Context) ([]string, error) { return q.cities, nil }
func (*httpQueriesStub) ListFeaturedCollections(context.Context, domain.BusinessType) ([]application.FeaturedCollection, error) {
	return nil, nil
}
func (q *httpQueriesStub) Search(_ context.Context, _ domain.BusinessType, query string, _ int) ([]application.CustomerBusinessSummary, error) {
	q.searchQuery = query
	return q.search, nil
}
func (q *httpQueriesStub) ListRecommendedStays(_ context.Context, city string, _ int) ([]application.RecommendedStay, error) {
	q.recommendedCity = city
	return q.recommended, nil
}

func TestDiscoveryHTTPContract(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	createdAt := time.Date(2026, time.August, 30, 10, 0, 0, 0, time.UTC)
	price := int64(12000)
	base := application.CustomerBusinessBase{ID: id, OwnerID: "provider", Type: domain.BusinessTypeStay, Status: domain.BusinessStatusActive, Name: "Hotel", CategoryID: "hotel", Currency: "BAM", Location: application.LocationSummary{City: "Sarajevo"}, PhotoPaths: []string{}, FeaturedCollectionIDs: []string{}, CreatedAt: createdAt, UpdatedAt: createdAt}
	queries := &httpQueriesStub{
		popular:     []application.CustomerBusinessSummary{{CustomerBusinessBase: base, Stay: &application.StaySummary{PricePerNight: &price, InventoryType: "single_unit"}}},
		cities:      []string{"Mostar", "Sarajevo"},
		search:      []application.CustomerBusinessSummary{{CustomerBusinessBase: base}},
		recommended: []application.RecommendedStay{{CustomerBusinessBase: base, Stay: application.StaySummary{PricePerNight: &price, InventoryType: "single_unit"}}},
	}
	handler := NewHandler(application.NewService(queries))

	t.Run("popular response preserves Flutter shape", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ListPopular(recorder, httptest.NewRequest(http.MethodGet, "/discovery/businesses?type=stays&city=%20Sarajevo%20&limit=2&offset=4", nil))
		body := responseObject(t, recorder)
		item := body["items"].([]any)[0].(map[string]any)
		if recorder.Code != http.StatusOK || body["nextOffset"] != float64(5) || item["type"] != "stays" || item["stayDetails"].(map[string]any)["pricePerNight"] != float64(12000) {
			t.Fatalf("unexpected response: status=%d body=%#v", recorder.Code, body)
		}
		if queries.popularType != domain.BusinessTypeStay || queries.popularCity != "sarajevo" || queries.popularLimit != 2 || queries.popularOffset != 4 {
			t.Fatalf("unexpected query: %#v", queries)
		}
	})

	t.Run("cities response is wrapped in items", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ListCities(recorder, httptest.NewRequest(http.MethodGet, "/discovery/cities", nil))
		if recorder.Code != http.StatusOK || len(responseObject(t, recorder)["items"].([]any)) != 2 {
			t.Fatalf("unexpected cities response: %s", recorder.Body.String())
		}
	})

	t.Run("recommendations use authenticated user city", func(t *testing.T) {
		role, city := usersdomain.UserRoleCustomer, " Sarajevo "
		request := usershttp.WithCurrentUser(httptest.NewRequest(http.MethodGet, "/discovery/recommended-stays", nil), usersdomain.User{ID: "customer", Role: &role, City: &city})
		recorder := httptest.NewRecorder()
		handler.ListRecommendedStays(recorder, request)
		if recorder.Code != http.StatusOK || queries.recommendedCity != "sarajevo" {
			t.Fatalf("unexpected recommendation response: status=%d city=%q", recorder.Code, queries.recommendedCity)
		}
	})

	t.Run("invalid detail id keeps standard error contract", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/discovery/businesses/bad", nil)
		route := chi.NewRouteContext()
		route.URLParams.Add("businessID", "bad")
		recorder := httptest.NewRecorder()
		handler.GetDetail(recorder, request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route)))
		if recorder.Code != http.StatusUnprocessableEntity || responseObject(t, recorder)["error"].(map[string]any)["code"] != "validation_error" {
			t.Fatalf("unexpected error response: %s", recorder.Body.String())
		}
	})
}

func responseObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}
