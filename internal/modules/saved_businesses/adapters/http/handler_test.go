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
	"github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

type httpStub struct {
	err error
	ids []uuid.UUID
}

func (s *httpStub) Save(context.Context, string, uuid.UUID) error   { return s.err }
func (s *httpStub) Remove(context.Context, string, uuid.UUID) error { return s.err }
func (s *httpStub) IsSaved(context.Context, string, uuid.UUID) (bool, error) {
	return true, s.err
}
func (s *httpStub) ListBusinessIDs(context.Context, string) ([]uuid.UUID, error) {
	return s.ids, s.err
}
func (s *httpStub) ListActiveByIDs(context.Context, []uuid.UUID) ([]discoveryapp.CustomerBusinessSummary, error) {
	now := time.Now()
	return []discoveryapp.CustomerBusinessSummary{
		{CustomerBusinessBase: discoveryapp.CustomerBusinessBase{ID: uuid.New(), Type: domain.BusinessTypeStay, Status: domain.BusinessStatusActive, PhotoPaths: []string{}, FeaturedCollectionIDs: []string{}, CreatedAt: now, UpdatedAt: now}},
		{CustomerBusinessBase: discoveryapp.CustomerBusinessBase{ID: uuid.New(), Type: domain.BusinessTypeService, Status: domain.BusinessStatusActive, PhotoPaths: []string{}, FeaturedCollectionIDs: []string{}, CreatedAt: now, UpdatedAt: now}},
	}, s.err
}

func TestSavedHTTPContract(t *testing.T) {
	stub := &httpStub{}
	handler := NewHandler(application.NewService(stub, stub, stub))
	role := usersdomain.UserRoleCustomer
	customer := usersdomain.User{ID: "customer", Role: &role}
	for _, test := range []struct {
		name    string
		method  string
		handler http.HandlerFunc
		status  int
	}{{"save", http.MethodPut, handler.Save, 204}, {"remove", http.MethodDelete, handler.Remove, 204}, {"check", http.MethodGet, handler.IsSaved, 200}, {"list", http.MethodGet, handler.List, 200}} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.handler(recorder, requestWithBusinessID(test.method, uuid.New(), customer))
			if recorder.Code != test.status {
				t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
			}
			if test.name == "check" && responseObject(t, recorder)["isSaved"] != true {
				t.Fatal("missing saved status")
			}
			if test.name == "list" {
				items := responseObject(t, recorder)["items"].([]any)
				if len(items) != 2 || items[0].(map[string]any)["type"] != "stays" || items[1].(map[string]any)["type"] != "services" {
					t.Fatalf("incorrect summary contract: %#v", items)
				}
			}
		})
	}

	providerRole := usersdomain.UserRoleProvider
	recorder := httptest.NewRecorder()
	handler.Save(recorder, requestWithBusinessID(http.MethodPut, uuid.New(), usersdomain.User{ID: "provider", Role: &providerRole}))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("provider status %d", recorder.Code)
	}

	stub.err = application.ErrBusinessNotFound
	recorder = httptest.NewRecorder()
	handler.Save(recorder, requestWithBusinessID(http.MethodPut, uuid.New(), customer))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unavailable business status %d", recorder.Code)
	}
}

func requestWithBusinessID(method string, id uuid.UUID, user usersdomain.User) *http.Request {
	request := usershttp.WithCurrentUser(httptest.NewRequest(method, "/saved-businesses/"+id.String(), nil), user)
	route := chi.NewRouteContext()
	route.URLParams.Add("businessID", id.String())
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
}

func responseObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}
