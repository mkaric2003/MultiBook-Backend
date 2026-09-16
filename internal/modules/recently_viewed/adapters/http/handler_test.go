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
	discoveryapp "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

type httpStub struct{ err error }

func (s *httpStub) Record(context.Context, string, uuid.UUID) error { return s.err }
func (s *httpStub) ListBusinessIDs(context.Context, string, domain.BusinessType, int) ([]uuid.UUID, error) {
	return []uuid.UUID{uuid.New(), uuid.New()}, s.err
}
func (s *httpStub) ListActiveByIDs(context.Context, []uuid.UUID) ([]discoveryapp.CustomerBusinessSummary, error) {
	now := time.Now()
	return []discoveryapp.CustomerBusinessSummary{
		{CustomerBusinessBase: discoveryapp.CustomerBusinessBase{ID: uuid.New(), Type: domain.BusinessTypeStay, Status: domain.BusinessStatusActive, PhotoPaths: []string{}, FeaturedCollectionIDs: []string{}, CreatedAt: now, UpdatedAt: now}},
		{CustomerBusinessBase: discoveryapp.CustomerBusinessBase{ID: uuid.New(), Type: domain.BusinessTypeService, Status: domain.BusinessStatusActive, PhotoPaths: []string{}, FeaturedCollectionIDs: []string{}, CreatedAt: now, UpdatedAt: now}},
	}, s.err
}

func TestRecentlyViewedHTTPContract(t *testing.T) {
	stub := &httpStub{}
	handler := NewHandler(application.NewService(stub, stub, stub))
	role := usersdomain.UserRoleCustomer
	customer := usersdomain.User{ID: "customer", Role: &role}

	recorder := httptest.NewRecorder()
	handler.Record(recorder, recordRequest(uuid.New().String(), customer))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("record status %d: %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.List(recorder, listRequest("stays", "10", customer))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", recorder.Code, recorder.Body.String())
	}
	items := responseObject(t, recorder)["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["type"] != "stays" || items[1].(map[string]any)["type"] != "services" {
		t.Fatalf("incorrect summary contract: %#v", items)
	}
}

func TestRecentlyViewedHTTPErrors(t *testing.T) {
	stub := &httpStub{}
	handler := NewHandler(application.NewService(stub, stub, stub))
	customerRole := usersdomain.UserRoleCustomer
	providerRole := usersdomain.UserRoleProvider
	customer := usersdomain.User{ID: "customer", Role: &customerRole}

	for _, test := range []struct {
		name   string
		invoke func(*httptest.ResponseRecorder)
		status int
	}{
		{"invalid UUID", func(recorder *httptest.ResponseRecorder) {
			handler.Record(recorder, recordRequest("invalid", customer))
		}, http.StatusUnprocessableEntity},
		{"invalid type", func(recorder *httptest.ResponseRecorder) {
			handler.List(recorder, listRequest("invalid", "10", customer))
		}, http.StatusUnprocessableEntity},
		{"provider", func(recorder *httptest.ResponseRecorder) {
			handler.Record(recorder, recordRequest(uuid.NewString(), usersdomain.User{ID: "provider", Role: &providerRole}))
		}, http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.invoke(recorder)
			if recorder.Code != test.status {
				t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
			}
		})
	}

	stub.err = application.ErrBusinessNotFound
	recorder := httptest.NewRecorder()
	handler.Record(recorder, recordRequest(uuid.NewString(), customer))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing business status %d", recorder.Code)
	}
}

func recordRequest(businessID string, user usersdomain.User) *http.Request {
	request := usershttp.WithCurrentUser(httptest.NewRequest(http.MethodPut, "/recently-viewed/"+businessID, nil), user)
	route := chi.NewRouteContext()
	route.URLParams.Add("businessID", businessID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
}

func listRequest(businessType, limit string, user usersdomain.User) *http.Request {
	return usershttp.WithCurrentUser(httptest.NewRequest(http.MethodGet, "/recently-viewed?type="+businessType+"&limit="+limit, nil), user)
}

func responseObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}
